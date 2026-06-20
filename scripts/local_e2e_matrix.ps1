param(
    [string]$CandidateRoot = "D:\tmp\csf-local-candidate",
    [string]$CandidateContainer = "chinesesubfinder-local-candidate",
    [string]$BaseUrl = "http://127.0.0.1:19235",
    [string]$Username = "",
    [string]$Password = "",
    [int]$JobTimeoutSeconds = 900,
    [string]$SampleSpecPath = "",
    [ValidateSet("movie","series")]
    [string]$SampleKind = "movie",
    [string]$SampleFolderName = "Local Candidate Matrix (2024)",
    [string]$SampleBaseName = "Local.Candidate.Matrix.2024.1080p.WEB-DL",
    [string]$ExternalMoviesRoot = "",
    [string]$ExternalSeriesRoot = "",
    [string]$DockerMoviesSource = "",
    [string]$DockerSeriesSource = "",
    [string]$ConfigDockerVolume = "",
    [string[]]$PrimaryChineseSuppliers = @(),
    [string[]]$EnglishFallbackSuppliers = @(),
    [string]$ExpectedRouteKey = "",
    [string]$ExpectedWinningSupplier = "",
    [switch]$KeepGeneratedSubtitles,
    [switch]$AcceptNoSubFound,
    [switch]$EnableSubtitleCatTranslatedChineseFallback,
    [switch]$EnableLLMFallback,
    [string]$LLMProvider = "deepseek",
    [string]$LLMBaseUrl = "",
    [string]$LLMApiKey = "",
    [string]$LLMModel = "deepseek-v4-flash",
    [string]$HelperImage = "chinesesubfinder:local-candidate"
)

$ErrorActionPreference = "Stop"
. (Join-Path $PSScriptRoot "local_acceptance_matrix_utils.ps1")
$primarySuppliersSpecified = $PSBoundParameters.ContainsKey("PrimaryChineseSuppliers")
$englishSuppliersSpecified = $PSBoundParameters.ContainsKey("EnglishFallbackSuppliers")
$script:LatestAccessToken = ""

if (-not [string]::IsNullOrWhiteSpace($SampleSpecPath)) {
    $sampleSpec = Get-Content -LiteralPath $SampleSpecPath -Raw -Encoding UTF8 | ConvertFrom-Json
    if ($null -ne $sampleSpec.sample_kind) { $SampleKind = [string]$sampleSpec.sample_kind }
    if ($null -ne $sampleSpec.sample_folder_name) { $SampleFolderName = [string]$sampleSpec.sample_folder_name }
    if ($null -ne $sampleSpec.sample_base_name) { $SampleBaseName = [string]$sampleSpec.sample_base_name }
    if ($null -ne $sampleSpec.external_movies_root) { $ExternalMoviesRoot = [string]$sampleSpec.external_movies_root }
    if ($null -ne $sampleSpec.external_series_root) { $ExternalSeriesRoot = [string]$sampleSpec.external_series_root }
    if ($null -ne $sampleSpec.docker_movies_source) { $DockerMoviesSource = [string]$sampleSpec.docker_movies_source }
    if ($null -ne $sampleSpec.docker_series_source) { $DockerSeriesSource = [string]$sampleSpec.docker_series_source }
}

function Ensure-Dir([string]$Path) {
    if (-not (Test-Path -LiteralPath $Path)) {
        New-Item -ItemType Directory -Path $Path -Force | Out-Null
    }
}

function Write-JsonFile($Value, [string]$Path) {
    Write-JsonUtf8File -Value $Value -Path $Path -Depth 20
}

function Read-JsonFileUtf8([string]$Path) {
    return Get-Content -LiteralPath $Path -Raw -Encoding UTF8 | ConvertFrom-Json
}

function Get-RunLockName {
    $parts = @(
        [string]$CandidateContainer,
        [string]$BaseUrl,
        [string]$ConfigDockerVolume
    ) -join "|"
    $bytes = [System.Text.Encoding]::UTF8.GetBytes($parts)
    $sha1 = [System.Security.Cryptography.SHA1]::Create()
    try {
        $hash = $sha1.ComputeHash($bytes)
    } finally {
        $sha1.Dispose()
    }
    $hex = [System.BitConverter]::ToString($hash).Replace("-", "")
    return "Global\CSFLocalE2EMatrix-$hex"
}

function Acquire-RunLock {
    $script:RunLockMutex = New-Object System.Threading.Mutex($false, (Get-RunLockName))
    $acquired = $false
    try {
        $acquired = $script:RunLockMutex.WaitOne([TimeSpan]::FromMinutes(10))
    } catch [System.Threading.AbandonedMutexException] {
        $acquired = $true
    }
    if (-not $acquired) {
        throw "Timed out waiting for local E2E run lock for $CandidateContainer at $BaseUrl"
    }
    $script:RunLockAcquired = $true
}

function Release-RunLock {
    if ($null -eq $script:RunLockMutex) {
        return
    }
    try {
        if ($script:RunLockAcquired) {
            $script:RunLockMutex.ReleaseMutex()
            $script:RunLockAcquired = $false
        }
    } catch {
    } finally {
        $script:RunLockMutex.Dispose()
        $script:RunLockMutex = $null
    }
}

function Clone-JsonObject($Value) {
    if ($null -eq $Value) {
        return $null
    }
    return ($Value | ConvertTo-Json -Depth 30 | ConvertFrom-Json)
}

function Use-ConfigDockerVolume {
    return -not [string]::IsNullOrWhiteSpace($ConfigDockerVolume)
}

function Get-SettingsRoot {
    if (Use-ConfigDockerVolume) {
        return (Join-Path $CandidateRoot "config-shadow")
    }
    return (Join-Path $CandidateRoot "config")
}

function Sync-SettingsFromConfigVolume {
    if (-not (Use-ConfigDockerVolume)) {
        return
    }
    $shadowRoot = Get-SettingsRoot
    Ensure-Dir $shadowRoot
    $shadowSettingsPath = Join-Path $shadowRoot "ChineseSubFinderSettings.json"
    if (Test-Path -LiteralPath $shadowSettingsPath) {
        Remove-Item -LiteralPath $shadowSettingsPath -Force
    }
    $dockerArgs = @(
        "run", "--rm",
        "-v", "${ConfigDockerVolume}:/volume",
        "-v", "${shadowRoot}:/shadow",
        "--entrypoint", "sh",
        $HelperImage,
        "-lc", "if [ -f /volume/ChineseSubFinderSettings.json ]; then cp -f /volume/ChineseSubFinderSettings.json /shadow/ChineseSubFinderSettings.json; fi"
    )
    & docker @dockerArgs | Out-Null
    if ($LASTEXITCODE -ne 0) {
        throw "Failed to sync settings from Docker volume $ConfigDockerVolume"
    }
    if (-not (Test-Path -LiteralPath $shadowSettingsPath)) {
        throw "Docker volume $ConfigDockerVolume does not contain ChineseSubFinderSettings.json"
    }
}

function Sync-SettingsToConfigVolume {
    if (-not (Use-ConfigDockerVolume)) {
        return
    }
    $shadowRoot = Get-SettingsRoot
    $settingsPath = Join-Path $shadowRoot "ChineseSubFinderSettings.json"
    if (-not (Test-Path -LiteralPath $settingsPath)) {
        return
    }
    $dockerArgs = @(
        "run", "--rm",
        "-v", "${ConfigDockerVolume}:/volume",
        "-v", "${shadowRoot}:/shadow",
        "--entrypoint", "sh",
        $HelperImage,
        "-lc", "cp -f /shadow/ChineseSubFinderSettings.json /volume/ChineseSubFinderSettings.json"
    )
    & docker @dockerArgs | Out-Null
    if ($LASTEXITCODE -ne 0) {
        throw "Failed to sync settings into Docker volume $ConfigDockerVolume"
    }
}

function Resolve-LoginCredentials([string]$ConfigFilePath) {
    if ((-not [string]::IsNullOrWhiteSpace($Username)) -and (-not [string]::IsNullOrWhiteSpace($Password))) {
        return
    }
    if (-not (Test-Path -LiteralPath $ConfigFilePath)) {
        throw "Config file not found for login credential resolution: $ConfigFilePath"
    }
    $settings = Read-JsonFileUtf8 $ConfigFilePath
    if ([string]::IsNullOrWhiteSpace($Username)) {
        $script:Username = [string]$settings.user_info.username
    }
    if ([string]::IsNullOrWhiteSpace($Password)) {
        $script:Password = [string]$settings.user_info.password
    }
    if ([string]::IsNullOrWhiteSpace($script:Username) -or [string]::IsNullOrWhiteSpace($script:Password)) {
        throw "Could not resolve login credentials from current candidate config."
    }
}

function Invoke-Json {
    param(
        [ValidateSet("GET", "POST", "PUT")]
        [string]$Method,
        [string]$Path,
        [object]$Body = $null,
        [string]$Token = ""
    )
    $headers = @{}
    if ($Token -ne "") {
        $headers.Authorization = "Bearer $Token"
    }
    $uri = $BaseUrl.TrimEnd("/") + $Path
    try {
        if ($null -eq $Body) {
            return Invoke-RestMethod -Method $Method -Uri $uri -Headers $headers -TimeoutSec 60
        }
        $jsonBody = $Body | ConvertTo-Json -Depth 20
        $payload = [System.Text.UTF8Encoding]::new($false).GetBytes($jsonBody)
        return Invoke-RestMethod -Method $Method -Uri $uri -Headers $headers -Body $payload -ContentType "application/json; charset=utf-8" -TimeoutSec 60
    } catch {
        $statusCode = $null
        if ($null -ne $_.Exception.Response -and $null -ne $_.Exception.Response.StatusCode) {
            $statusCode = [int]$_.Exception.Response.StatusCode
        }
        $canRetryWithFreshToken = $Token -ne "" -and $Path -ne "/login" -and $statusCode -eq 401
        if (-not $canRetryWithFreshToken) {
            throw
        }

        $relogin = Invoke-RestMethod -Method POST -Uri ($BaseUrl.TrimEnd("/") + "/login") -Body (@{ username = $Username; password = $Password } | ConvertTo-Json -Depth 5) -ContentType "application/json; charset=utf-8" -TimeoutSec 60
        if ($null -eq $relogin -or [string]::IsNullOrWhiteSpace([string]$relogin.access_token)) {
            throw
        }
        $script:LatestAccessToken = [string]$relogin.access_token
        $headers.Authorization = "Bearer $($script:LatestAccessToken)"

        if ($null -eq $Body) {
            return Invoke-RestMethod -Method $Method -Uri $uri -Headers $headers -TimeoutSec 60
        }
        $jsonBody = $Body | ConvertTo-Json -Depth 20
        $payload = [System.Text.UTF8Encoding]::new($false).GetBytes($jsonBody)
        return Invoke-RestMethod -Method $Method -Uri $uri -Headers $headers -Body $payload -ContentType "application/json; charset=utf-8" -TimeoutSec 60
    }
}

function Assert-True([bool]$Condition, [string]$Message) {
    if (-not $Condition) {
        throw $Message
    }
}

function Copy-SanitizedSettings($SettingsObject) {
    $copy = $SettingsObject | ConvertTo-Json -Depth 20 | ConvertFrom-Json
    if ($null -ne $copy.user_info) {
        $copy.user_info.password = "******"
    }
    if ($null -ne $copy.subtitle_sources) {
        if ($null -ne $copy.subtitle_sources.assrt_settings) { $copy.subtitle_sources.assrt_settings.token = "" }
        if ($null -ne $copy.subtitle_sources.subdl_settings) { $copy.subtitle_sources.subdl_settings.key = "" }
        if ($null -ne $copy.subtitle_sources.subtitle_best_settings) { $copy.subtitle_sources.subtitle_best_settings.api_key = "" }
        if ($null -ne $copy.subtitle_sources.opensubtitles_settings) {
            $copy.subtitle_sources.opensubtitles_settings.api_key = ""
            $copy.subtitle_sources.opensubtitles_settings.username = ""
            $copy.subtitle_sources.opensubtitles_settings.password = ""
        }
    }
    if ($null -ne $copy.experimental_function -and $null -ne $copy.experimental_function.llm_subtitle_fallback) {
        $copy.experimental_function.llm_subtitle_fallback.api_key = ""
    }
    return $copy
}

function Normalize-BaselineSettings($SettingsObject) {
    if ($null -ne $SettingsObject.subtitle_sources -and $null -ne $SettingsObject.subtitle_sources.subtitlecat_settings) {
        $SettingsObject.subtitle_sources.subtitlecat_settings.enable_translated_chinese_fallback = $false
    }
    if ($null -ne $SettingsObject.experimental_function -and $null -ne $SettingsObject.experimental_function.llm_subtitle_fallback) {
        $SettingsObject.experimental_function.llm_subtitle_fallback.enable = $false
        $SettingsObject.experimental_function.llm_subtitle_fallback.api_key = ""
        $SettingsObject.experimental_function.llm_subtitle_fallback.only_when_no_chinese_candidate = $true
    }
    return $SettingsObject
}

function ConvertTo-LowerNameSet([string[]]$Names) {
    $set = @{}
    foreach ($name in $Names) {
        if ([string]::IsNullOrWhiteSpace($name)) { continue }
        foreach ($part in ($name -split ',')) {
            if ([string]::IsNullOrWhiteSpace($part)) { continue }
            $normalized = $part.Trim().ToLowerInvariant()
            if ($normalized -eq "__none__") { continue }
            $set[$normalized] = $true
        }
    }
    return $set
}

function Test-EnabledBySet($Set, [string]$Name, [bool]$AllowAllWhenEmpty = $true) {
    if ($Set.Count -eq 0) {
        if (-not $AllowAllWhenEmpty) {
            return $false
        }
        return $true
    }
    return $Set.ContainsKey($Name.ToLowerInvariant())
}

function Use-ImplicitSubtitleCatTranslatedIsolation {
    return $EnableSubtitleCatTranslatedChineseFallback -and (-not $primarySuppliersSpecified) -and (-not $englishSuppliersSpecified)
}

function Assert-RouteSettings($SettingsObject) {
    $primarySet = ConvertTo-LowerNameSet $PrimaryChineseSuppliers
    $englishSet = ConvertTo-LowerNameSet $EnglishFallbackSuppliers
    $isImplicitTranslatedIsolation = Use-ImplicitSubtitleCatTranslatedIsolation
    $primaryAllowAll = (-not $primarySuppliersSpecified) -and (-not $isImplicitTranslatedIsolation)
    $englishAllowAll = (-not $englishSuppliersSpecified) -and (-not $primarySuppliersSpecified) -and (-not $isImplicitTranslatedIsolation)

    $expectSubhd = Test-EnabledBySet $primarySet "subhd" $primaryAllowAll
    $expectSubhdLimit = if ($expectSubhd) { 20 } else { 0 }
    $expectTvsubtitles = (Test-EnabledBySet $primarySet "tvsubtitles" $primaryAllowAll) -or (Test-EnabledBySet $englishSet "tvsubtitles" $englishAllowAll)
    $expectTvsubtitlesLimit = if ($expectTvsubtitles) { -1 } else { 0 }
    $expectMoviesubtitles = (Test-EnabledBySet $primarySet "moviesubtitles" $primaryAllowAll) -or (Test-EnabledBySet $englishSet "moviesubtitles" $englishAllowAll)
    $expectMoviesubtitlesLimit = if ($expectMoviesubtitles) { -1 } else { 0 }
    $expectSubtitleCatLimit = if ((Test-EnabledBySet $englishSet "subtitlecat" $englishAllowAll) -or $EnableSubtitleCatTranslatedChineseFallback) { -1 } else { 0 }
    $expectTranslatedChineseFallback = [bool]$EnableSubtitleCatTranslatedChineseFallback

    Assert-True ($SettingsObject.subtitle_sources.subhd_settings.enabled -eq $expectSubhd) "subhd enabled route mismatch."
    Assert-True ([int]$SettingsObject.advanced_settings.suppliers_settings.subhd.daily_download_limit -eq $expectSubhdLimit) "subhd daily limit route mismatch."
    Assert-True ($SettingsObject.subtitle_sources.tvsubtitles_settings.enabled -eq $expectTvsubtitles) "tvsubtitles enabled route mismatch."
    Assert-True ([int]$SettingsObject.advanced_settings.suppliers_settings.tvsubtitles.daily_download_limit -eq $expectTvsubtitlesLimit) "tvsubtitles daily limit route mismatch."
    Assert-True ($SettingsObject.subtitle_sources.moviesubtitles_settings.enabled -eq $expectMoviesubtitles) "moviesubtitles enabled route mismatch."
    Assert-True ([int]$SettingsObject.advanced_settings.suppliers_settings.moviesubtitles.daily_download_limit -eq $expectMoviesubtitlesLimit) "moviesubtitles daily limit route mismatch."
    Assert-True ([int]$SettingsObject.advanced_settings.suppliers_settings.subtitlecat.daily_download_limit -eq $expectSubtitleCatLimit) "subtitlecat daily limit route mismatch."
    Assert-True ($SettingsObject.subtitle_sources.subtitlecat_settings.enable_translated_chinese_fallback -eq $expectTranslatedChineseFallback) "subtitlecat translated chinese fallback route mismatch."
    Assert-True ($SettingsObject.experimental_function.llm_subtitle_fallback.enable -eq [bool]$EnableLLMFallback) "llm fallback switch route mismatch."
}

function Apply-SupplierRouteSettings($SettingsObject) {
    $primarySet = ConvertTo-LowerNameSet $PrimaryChineseSuppliers
    $englishSet = ConvertTo-LowerNameSet $EnglishFallbackSuppliers
    $isImplicitTranslatedIsolation = Use-ImplicitSubtitleCatTranslatedIsolation
    $primaryAllowAll = (-not $primarySuppliersSpecified) -and (-not $isImplicitTranslatedIsolation)
    $englishAllowAll = (-not $englishSuppliersSpecified) -and (-not $primarySuppliersSpecified) -and (-not $isImplicitTranslatedIsolation)

    $SettingsObject.subtitle_sources.assrt_settings.enabled = (Test-EnabledBySet $primarySet "assrt" $primaryAllowAll)
    $SettingsObject.subtitle_sources.subdl_settings.enabled = (Test-EnabledBySet $primarySet "subdl" $primaryAllowAll) -or (Test-EnabledBySet $englishSet "subdl" $englishAllowAll)
    $SettingsObject.subtitle_sources.subhd_settings.enabled = (Test-EnabledBySet $primarySet "subhd" $primaryAllowAll)
    $SettingsObject.subtitle_sources.opensubtitles_settings.enabled = (Test-EnabledBySet $primarySet "opensubtitles" $primaryAllowAll) -or (Test-EnabledBySet $englishSet "opensubtitles" $englishAllowAll)
    $SettingsObject.subtitle_sources.tvsubtitles_settings.enabled = (Test-EnabledBySet $primarySet "tvsubtitles" $primaryAllowAll) -or (Test-EnabledBySet $englishSet "tvsubtitles" $englishAllowAll)
    $SettingsObject.subtitle_sources.moviesubtitles_settings.enabled = (Test-EnabledBySet $primarySet "moviesubtitles" $primaryAllowAll) -or (Test-EnabledBySet $englishSet "moviesubtitles" $englishAllowAll)
    $SettingsObject.subtitle_sources.subtitlecat_settings.enabled = $true

    $suppliers = $SettingsObject.advanced_settings.suppliers_settings
    if ($null -ne $suppliers) {
        $suppliers.xunlei.daily_download_limit = if (Test-EnabledBySet $primarySet "xunlei" $primaryAllowAll) { -1 } else { 0 }
        $suppliers.shooter.daily_download_limit = if (Test-EnabledBySet $primarySet "shooter" $primaryAllowAll) { -1 } else { 0 }
        $suppliers.assrt.daily_download_limit = if (Test-EnabledBySet $primarySet "assrt" $primaryAllowAll) { -1 } else { 0 }
        $suppliers.subdl.daily_download_limit = if ((Test-EnabledBySet $primarySet "subdl" $primaryAllowAll) -or (Test-EnabledBySet $englishSet "subdl" $englishAllowAll)) { -1 } else { 0 }
        $suppliers.subtitle_best.daily_download_limit = if (Test-EnabledBySet $primarySet "subtitle_best" $primaryAllowAll) { -1 } else { 0 }
        $suppliers.opensubtitles.daily_download_limit = if ((Test-EnabledBySet $primarySet "opensubtitles" $primaryAllowAll) -or (Test-EnabledBySet $englishSet "opensubtitles" $englishAllowAll)) { -1 } else { 0 }
        $suppliers.tvsubtitles.daily_download_limit = if ((Test-EnabledBySet $primarySet "tvsubtitles" $primaryAllowAll) -or (Test-EnabledBySet $englishSet "tvsubtitles" $englishAllowAll)) { -1 } else { 0 }
        $suppliers.moviesubtitles.daily_download_limit = if ((Test-EnabledBySet $primarySet "moviesubtitles" $primaryAllowAll) -or (Test-EnabledBySet $englishSet "moviesubtitles" $englishAllowAll)) { -1 } else { 0 }
        $suppliers.subtitlecat.daily_download_limit = if ((Test-EnabledBySet $englishSet "subtitlecat" $englishAllowAll) -or $EnableSubtitleCatTranslatedChineseFallback) { -1 } else { 0 }
        $suppliers.subhd.daily_download_limit = if (Test-EnabledBySet $primarySet "subhd" $primaryAllowAll) { 20 } else { 0 }
    }

    if ($englishSuppliersSpecified -and $englishSet.Count -gt 0 -and -not (Test-EnabledBySet $englishSet "subtitlecat" $englishAllowAll)) {
        $SettingsObject.subtitle_sources.subtitlecat_settings.enable_translated_chinese_fallback = $false
    }

    return $SettingsObject
}

function Get-ExplicitRequestedSupplierNames {
    $names = New-Object 'System.Collections.Generic.HashSet[string]' ([System.StringComparer]::OrdinalIgnoreCase)
    foreach ($raw in $PrimaryChineseSuppliers + $EnglishFallbackSuppliers) {
        if ([string]::IsNullOrWhiteSpace($raw)) { continue }
        foreach ($part in ($raw -split ',')) {
            if ([string]::IsNullOrWhiteSpace($part)) { continue }
            $normalized = $part.Trim()
            if ($normalized -eq "__none__") { continue }
            $names.Add($normalized) | Out-Null
        }
    }
    return @($names)
}

function Get-RequestedIsolationSupplier {
    if ($EnableSubtitleCatTranslatedChineseFallback -and (-not $primarySuppliersSpecified) -and (-not $englishSuppliersSpecified)) {
        return "subtitlecat_translated"
    }

    $explicitPrimary = @(Get-ExplicitRequestedSupplierNames | Where-Object { (ConvertTo-LowerNameSet $PrimaryChineseSuppliers).ContainsKey($_.ToLowerInvariant()) })
    $explicitEnglish = @(Get-ExplicitRequestedSupplierNames | Where-Object { (ConvertTo-LowerNameSet $EnglishFallbackSuppliers).ContainsKey($_.ToLowerInvariant()) })

    if ($explicitPrimary.Count -eq 1 -and $explicitEnglish.Count -eq 0) {
        return $explicitPrimary[0].ToLowerInvariant()
    }
    if ($explicitEnglish.Count -eq 1 -and $explicitPrimary.Count -eq 0) {
        return $explicitEnglish[0].ToLowerInvariant()
    }

    return ""
}

function Get-ExpectedRouteHint {
    if (-not [string]::IsNullOrWhiteSpace($ExpectedRouteKey)) {
        return $ExpectedRouteKey
    }
    if ($EnableLLMFallback) {
        return "$SampleKind.llm_fallback"
    }
    if ($AcceptNoSubFound) {
        return "$SampleKind.safe_fail"
    }
    if ($EnableSubtitleCatTranslatedChineseFallback) {
        return "$SampleKind.subtitlecat_translated"
    }
    return ""
}

function Assert-RequestedSupplierPrerequisites($SettingsObject) {
    $explicitSuppliers = Get-ExplicitRequestedSupplierNames
    if ($explicitSuppliers.Count -eq 0) {
        return
    }

    $missing = New-Object System.Collections.Generic.List[string]
    foreach ($supplierName in $explicitSuppliers) {
        switch ($supplierName.ToLowerInvariant()) {
            "assrt" {
                if ([string]::IsNullOrWhiteSpace([string]$SettingsObject.subtitle_sources.assrt_settings.token)) {
                    $missing.Add("assrt token") | Out-Null
                }
            }
            "subdl" {
                if ([string]::IsNullOrWhiteSpace([string]$SettingsObject.subtitle_sources.subdl_settings.key)) {
                    $missing.Add("subdl api key") | Out-Null
                }
            }
            "subtitle_best" {
                if ([string]::IsNullOrWhiteSpace([string]$SettingsObject.subtitle_sources.subtitle_best_settings.api_key)) {
                    $missing.Add("subtitle_best api key") | Out-Null
                }
            }
            "opensubtitles" {
                if ([string]::IsNullOrWhiteSpace([string]$SettingsObject.subtitle_sources.opensubtitles_settings.api_key)) {
                    $missing.Add("opensubtitles api key") | Out-Null
                }
                if ([string]::IsNullOrWhiteSpace([string]$SettingsObject.subtitle_sources.opensubtitles_settings.username)) {
                    $missing.Add("opensubtitles username") | Out-Null
                }
                if ([string]::IsNullOrWhiteSpace([string]$SettingsObject.subtitle_sources.opensubtitles_settings.password)) {
                    $missing.Add("opensubtitles password") | Out-Null
                }
            }
        }
    }

    if ($missing.Count -gt 0) {
        $details = ($missing.ToArray() | Sort-Object -Unique) -join ", "
        throw "Requested supplier prerequisites are missing in the pulled FnOS config: $details"
    }
}

function Get-RequestedSupplierPolicyWarnings {
    $warnings = New-Object System.Collections.Generic.List[string]
    $primarySet = ConvertTo-LowerNameSet $PrimaryChineseSuppliers
    $englishSet = ConvertTo-LowerNameSet $EnglishFallbackSuppliers

    $primaryCapable = ConvertTo-LowerNameSet @("xunlei", "shooter", "assrt", "opensubtitles", "subhd", "subtitle_best")
    $englishCapable = ConvertTo-LowerNameSet @("subdl", "opensubtitles", "moviesubtitles", "subtitlecat")

    foreach ($supplierName in (Get-ExplicitRequestedSupplierNames)) {
        $normalized = $supplierName.ToLowerInvariant()
        if ($primarySet.ContainsKey($normalized) -and (-not $primaryCapable.ContainsKey($normalized))) {
            $warnings.Add("Requested primary supplier '$normalized' is not wired into the backend primary Chinese chain.") | Out-Null
        }
        if ($englishSet.ContainsKey($normalized) -and (-not $englishCapable.ContainsKey($normalized))) {
            $warnings.Add("Requested English fallback supplier '$normalized' is not wired into the backend default English fallback chain.") | Out-Null
        }
        if ($englishSet.ContainsKey($normalized) -and $primaryCapable.ContainsKey($normalized) -and (-not $primarySet.ContainsKey($normalized))) {
            $warnings.Add("Requested English fallback supplier '$normalized' also participates in the backend primary Chinese chain when enabled, so it cannot be isolated as English-only under the current runtime policy.") | Out-Null
        }
    }

    return @($warnings | Select-Object -Unique)
}

function Get-HostMovieRoot {
    if (-not [string]::IsNullOrWhiteSpace($ExternalMoviesRoot)) {
        return $ExternalMoviesRoot
    }
    return (Join-Path $CandidateRoot "media\movies")
}

function Get-HostSeriesRoot {
    if (-not [string]::IsNullOrWhiteSpace($ExternalSeriesRoot)) {
        return $ExternalSeriesRoot
    }
    return (Join-Path $CandidateRoot "media\\series")
}

function Resolve-VideoPaths {
    if ($SampleKind -eq "series") {
        $hostSeriesRoot = Get-HostSeriesRoot
        $hostVideoPath = Join-Path (Join-Path $hostSeriesRoot $SampleFolderName) ($SampleBaseName + ".mkv")
        $containerFolder = $SampleFolderName -replace '\\', '/'
        $containerVideoPath = "/media/series/$containerFolder/$SampleBaseName.mkv"
        if (-not (Test-Path -LiteralPath $hostVideoPath)) {
            throw "Sample video not found: $hostVideoPath"
        }
        return [ordered]@{
            sample_kind = $SampleKind
            host_video_path = $hostVideoPath
            container_video_path = $containerVideoPath
            host_series_root = $hostSeriesRoot
        }
    }

    $hostMovieRoot = Get-HostMovieRoot
    $hostVideoPath = Join-Path (Join-Path $hostMovieRoot $SampleFolderName) ($SampleBaseName + ".mkv")
    $containerFolder = $SampleFolderName -replace '\\', '/'
    $containerVideoPath = "/media/movies/$containerFolder/$SampleBaseName.mkv"
    if (-not (Test-Path -LiteralPath $hostVideoPath)) {
        throw "Sample video not found: $hostVideoPath"
    }
    return [ordered]@{
        sample_kind = $SampleKind
        host_video_path = $hostVideoPath
        container_video_path = $containerVideoPath
        host_movie_root = $hostMovieRoot
    }
}

function Test-ContainerVideoPathExists([string]$ContainerPath) {
    if ([string]::IsNullOrWhiteSpace($CandidateContainer)) {
        return $false
    }
    $escapedPath = $ContainerPath.Replace("'", "'""'""'")
    $dockerArgs = @(
        "exec", $CandidateContainer,
        "sh", "-lc", "test -f '$escapedPath'"
    )
    & docker @dockerArgs | Out-Null
    return ($LASTEXITCODE -eq 0)
}

function Assert-ContainerVideoPathReady($ResolvedVideo) {
    if (-not (Test-ContainerVideoPathExists $ResolvedVideo.container_video_path)) {
        $hint = "Restart the candidate container with the same media spec or matching DockerMoviesSource/DockerSeriesSource so /media points at the real FnOS library."
        throw "Container sample path is not mounted: $($ResolvedVideo.container_video_path). $hint"
    }
}

function Get-NewSubtitleFilesForVideo([string]$VideoPath, [datetime]$StartedAt) {
    $dir = Split-Path -Parent $VideoPath
    $base = [System.IO.Path]::GetFileNameWithoutExtension($VideoPath)
    return @(Get-ChildItem -LiteralPath $dir -File -ErrorAction SilentlyContinue |
        Where-Object {
            $_.BaseName.StartsWith($base, [System.StringComparison]::OrdinalIgnoreCase) -and
            $_.Extension -match '^\.(srt|ass|ssa)$' -and
            $_.LastWriteTime -ge $StartedAt
        })
}

function Get-SubtitleFilesForVideo([string]$VideoPath) {
    $dir = Split-Path -Parent $VideoPath
    $base = [System.IO.Path]::GetFileNameWithoutExtension($VideoPath)
    return @(Get-ChildItem -LiteralPath $dir -File -ErrorAction SilentlyContinue |
        Where-Object {
            $_.BaseName.StartsWith($base, [System.StringComparison]::OrdinalIgnoreCase) -and
            $_.Extension -match '^\.(srt|ass|ssa)$'
        })
}

function Get-SubtitleSample([string]$Path) {
    $utf8 = [System.Text.UTF8Encoding]::new($false, $true)
    $text = [System.IO.File]::ReadAllText($Path, $utf8)
    $lines = @($text -split "`r?`n")
    if ($lines.Count -gt 24) {
        $lines = $lines[0..23]
    }
    return ($lines -join "`n")
}

function Test-HasChineseCharacters([string]$Text) {
    return [regex]::IsMatch($Text, '[\u4e00-\u9fff]')
}

function Get-LatestLLMLogActivity {
    $llmLogRoot = Join-Path $CandidateRoot "config\llm-logs"
    if (-not (Test-Path -LiteralPath $llmLogRoot)) {
        return $null
    }

    $latest = Get-ChildItem -LiteralPath $llmLogRoot -Recurse -File -ErrorAction SilentlyContinue |
        Sort-Object LastWriteTime -Descending |
        Select-Object -First 1
    if ($null -eq $latest) {
        return $null
    }

    return [ordered]@{
        full_name = $latest.FullName
        last_write_time = $latest.LastWriteTime.ToString("s")
        length = $latest.Length
    }
}

function Remove-GeneratedSubtitleFiles([System.Collections.IEnumerable]$Files) {
    foreach ($file in $Files) {
        if ($null -eq $file) { continue }
        if (Test-Path -LiteralPath $file.FullName) {
            Remove-Item -LiteralPath $file.FullName -Force -ErrorAction SilentlyContinue
        }
    }
}

function Wait-ForJob {
    param(
        [string]$Token,
        [string]$JobId,
        [string]$RoundRoot
    )
    $deadline = (Get-Date).AddSeconds($JobTimeoutSeconds)
    $samples = @()
    $lastObservedJob = $null
    $llmGraceApplied = $false
    $llmGraceSeconds = [Math]::Max(900, [int]([Math]::Ceiling($JobTimeoutSeconds * 0.5)))
    while ((Get-Date) -lt $deadline) {
        $jobsResponse = Invoke-Json -Method GET -Path "/v1/jobs/list" -Token $Token
        $job = @($jobsResponse.all_jobs | Where-Object { $_.id -eq $JobId } | Select-Object -First 1)
        $samples += [ordered]@{
            observed_at = (Get-Date).ToString("s")
            found = ($job.Count -gt 0)
            job = if ($job.Count -gt 0) { $job[0] } else { $null }
        }
        if ($job.Count -eq 0) {
            Start-Sleep -Seconds 3
            continue
        }
        $lastObservedJob = $job[0]
        $jobStatus = [int]$job[0].job_status
        if ($jobStatus -in @(2, 3, 5)) {
            Write-JsonFile $samples (Join-Path $RoundRoot "job-status-samples.json")
            return $job[0]
        }
        Start-Sleep -Seconds 5
    }

    if ($EnableLLMFallback -and -not $llmGraceApplied -and $null -ne $lastObservedJob -and [int]$lastObservedJob.job_status -eq 4) {
        $llmGraceApplied = $true
        $deadline = (Get-Date).AddSeconds($llmGraceSeconds)
        $samples += [ordered]@{
            observed_at = (Get-Date).ToString("s")
            found = $true
            job = $lastObservedJob
            grace_extension_seconds = $llmGraceSeconds
            reason = "llm_fallback_job_still_downloading"
            latest_llm_log = Get-LatestLLMLogActivity
        }

        while ((Get-Date) -lt $deadline) {
            $jobsResponse = Invoke-Json -Method GET -Path "/v1/jobs/list" -Token $Token
            $job = @($jobsResponse.all_jobs | Where-Object { $_.id -eq $JobId } | Select-Object -First 1)
            $samples += [ordered]@{
                observed_at = (Get-Date).ToString("s")
                found = ($job.Count -gt 0)
                job = if ($job.Count -gt 0) { $job[0] } else { $null }
            }
            if ($job.Count -eq 0) {
                Start-Sleep -Seconds 3
                continue
            }
            $lastObservedJob = $job[0]
            $jobStatus = [int]$job[0].job_status
            if ($jobStatus -in @(2, 3, 5)) {
                Write-JsonFile $samples (Join-Path $RoundRoot "job-status-samples.json")
                return $job[0]
            }
            Start-Sleep -Seconds 5
        }
    }

    Write-JsonFile $samples (Join-Path $RoundRoot "job-status-samples.json")
    $lastStatus = if ($null -ne $lastObservedJob) { [int]$lastObservedJob.job_status } else { -1 }
    throw "Job timed out: $JobId status=$lastStatus llm_grace_applied=$llmGraceApplied"
}

$roundId = (Get-Date -Format "yyyyMMdd-HHmmss-fff") + "-e2e-matrix"
$roundRoot = Join-Path (Join-Path $CandidateRoot "reports") $roundId
Acquire-RunLock
if (Use-ConfigDockerVolume) {
    Sync-SettingsFromConfigVolume
}
$configFile = Join-Path (Get-SettingsRoot) "ChineseSubFinderSettings.json"
$baselineConfigFile = Join-Path (Join-Path $CandidateRoot "artifacts") "baseline-settings.json"
Resolve-LoginCredentials -ConfigFilePath $configFile
$originalConfigText = $null
if (Test-Path -LiteralPath $baselineConfigFile) {
    $originalConfigText = Get-Content -LiteralPath $baselineConfigFile -Raw -Encoding UTF8
} elseif (Test-Path -LiteralPath $configFile) {
    $originalConfigText = Get-Content -LiteralPath $configFile -Raw -Encoding UTF8
}
$script:TokenForCleanup = ""
$script:CleanupAttempted = $false

function Restore-RoundSettings {
    if ($script:CleanupAttempted) {
        return
    }
    $script:CleanupAttempted = $true

    if ($null -ne $originalConfigText) {
        [System.IO.File]::WriteAllText($configFile, $originalConfigText, [System.Text.UTF8Encoding]::new($false))
        Sync-SettingsToConfigVolume
    }

    if ($script:TokenForCleanup -eq "") {
        return
    }

    try {
        if ($null -ne $originalConfigText) {
            Invoke-Json -Method PUT -Path "/v1/settings" -Token $script:TokenForCleanup -Body (Read-JsonFileUtf8 $configFile) | Out-Null
            return
        }

        if ($EnableLLMFallback) {
            $settingsCleanup = Invoke-Json -Method GET -Path "/v1/settings" -Token $script:TokenForCleanup
            $settingsCleanup.subtitle_sources.subtitlecat_settings.enable_translated_chinese_fallback = $false
            $settingsCleanup.experimental_function.llm_subtitle_fallback.api_key = ""
            $settingsCleanup.experimental_function.llm_subtitle_fallback.enable = $false
            Invoke-Json -Method PUT -Path "/v1/settings" -Token $script:TokenForCleanup -Body $settingsCleanup | Out-Null
        }
    } catch {
        Write-Warning "Could not restore round settings: $($_.Exception.Message)"
    }
}

trap {
    $failureMessage = $_.Exception.Message
    try {
        $failure = [ordered]@{
            failed_at = (Get-Date).ToString("s")
            message = $failureMessage
            script_stack_trace = $_.ScriptStackTrace
            invocation = $_.InvocationInfo.PositionMessage
        }
        Write-JsonFile $failure (Join-Path $roundRoot "failure.json")
        if ($null -ne $summary) {
            $summary.failed_at = $failure.failed_at
            $summary.failure_message = $failure.message
            $summary.failure_written = $true
            if ($null -eq $summary.checks) {
                $summary.checks = [ordered]@{}
            }
            $summary.checks.failure = "failed"
            Write-JsonFile $summary (Join-Path $roundRoot "e2e-summary.json")
        }
    } catch {
    }
    try {
        $restoreCommand = Get-Command -Name Restore-RoundSettings -CommandType Function -ErrorAction SilentlyContinue
        if ($null -ne $restoreCommand) {
            Restore-RoundSettings
        }
    } finally {
        $releaseCommand = Get-Command -Name Release-RunLock -CommandType Function -ErrorAction SilentlyContinue
        if ($null -ne $releaseCommand) {
            Release-RunLock
        }
    }
    [Console]::Error.WriteLine("local_e2e_matrix failure: " + $failureMessage)
    throw
}

Ensure-Dir $roundRoot
Ensure-Dir (Join-Path $CandidateRoot "artifacts")

$summary = [ordered]@{
    round_id = $roundId
    round_root = $roundRoot
    base_url = $BaseUrl
    sample_folder_name = $SampleFolderName
    sample_base_name = $SampleBaseName
    sample_kind = $SampleKind
    external_movies_root = $ExternalMoviesRoot
    external_series_root = $ExternalSeriesRoot
    docker_movies_source = $DockerMoviesSource
    docker_series_source = $DockerSeriesSource
    started_at = (Get-Date).ToString("s")
    requested_primary_suppliers = @($PrimaryChineseSuppliers)
    requested_english_fallback_suppliers = @($EnglishFallbackSuppliers)
    requested_isolation_supplier = Get-RequestedIsolationSupplier
    expected_route_key = Get-ExpectedRouteHint
    policy_warnings = @()
    checks = [ordered]@{}
}

$policyWarnings = @(Get-RequestedSupplierPolicyWarnings)
if ($policyWarnings.Count -gt 0) {
    $summary.policy_warnings = @($policyWarnings)
    Write-JsonFile @($policyWarnings) (Join-Path $roundRoot "policy-warnings.json")
    foreach ($warning in $policyWarnings) {
        Write-Warning $warning
    }
}

function Resolve-RouteKey {
    param(
        [string]$SampleKind,
        [bool]$EnableSubtitleCatTranslatedChineseFallback,
        [bool]$EnableLLMFallback,
        [bool]$NoSubFoundAccepted,
        [bool]$HasChineseSample,
        [string]$StageEvidence = ""
    )

    if (-not [string]::IsNullOrWhiteSpace($StageEvidence)) {
        switch ($StageEvidence) {
            "primary_chinese" { return "$SampleKind.native_chinese" }
            "translated_chinese" { return "$SampleKind.subtitlecat_translated" }
            "llm_fallback" { return "$SampleKind.llm_fallback" }
            "english_fallback" { return "$SampleKind.english_fallback" }
            "safe_fail" { return "$SampleKind.safe_fail" }
        }
    }
    if ($EnableLLMFallback) {
        return "$SampleKind.llm_fallback"
    }
    if ($NoSubFoundAccepted) {
        return "$SampleKind.safe_fail"
    }
    if ($HasChineseSample) {
        if ($EnableSubtitleCatTranslatedChineseFallback) {
            return "$SampleKind.subtitlecat_translated"
        }
        return "$SampleKind.native_chinese"
    }
    return "$SampleKind.english_fallback"
}

function Get-JobSupplierEvidence {
    param(
        [datetime]$SinceTime
    )

    $sinceUtc = $SinceTime.ToUniversalTime().ToString("yyyy-MM-ddTHH:mm:ssZ")
    $capturePath = Join-Path (Join-Path $CandidateRoot "tmp") "job-supplier-evidence.log"
    if (Test-Path -LiteralPath $capturePath) {
        Remove-Item -LiteralPath $capturePath -Force
    }
    $psi = New-Object System.Diagnostics.ProcessStartInfo
    $psi.FileName = "docker"
    $psi.Arguments = ('logs --since "{0}" "{1}"' -f $sinceUtc, $CandidateContainer)
    $psi.UseShellExecute = $false
    $psi.RedirectStandardError = $true
    $process = New-Object System.Diagnostics.Process
    $process.StartInfo = $psi
    $processExitCode = -1
    $stderr = ""
    try {
        $null = $process.Start()
        $stderr = $process.StandardError.ReadToEnd()
        $process.WaitForExit()
        $processExitCode = $process.ExitCode
    } finally {
        $process.Dispose()
    }
    [System.IO.File]::WriteAllText($capturePath, $stderr, [System.Text.UTF8Encoding]::new($false))
    if ($processExitCode -ne 0) {
        throw "Failed to read container logs for supplier evidence."
    }
    $lines = @()
    if (Test-Path -LiteralPath $capturePath) {
        $lines = Get-Content -LiteralPath $capturePath -Encoding UTF8
    }

    $orgLine = $null
    foreach ($line in $lines) {
        if ($line -like "*OrgSubName:*") {
            $orgLine = $line
        }
    }

    $supplier = ""
    if ($null -ne $orgLine) {
        $match = [regex]::Match([string]$orgLine, 'OrgSubName:\s+\[([^\]]+)\]')
        if ($match.Success) {
            $supplier = $match.Groups[1].Value
        }
    }

    $stageLine = $null
    foreach ($line in $lines) {
        if ($line -like "*SubtitleRouteStage*") {
            $stageLine = $line
        }
    }

    $stage = ""
    if ($null -ne $stageLine) {
        $match = [regex]::Match([string]$stageLine, 'SubtitleRouteStage\s+([a-z_]+)')
        if ($match.Success) {
            $stage = $match.Groups[1].Value
        }
    }

    return [ordered]@{
        log_since_utc = $sinceUtc
        actual_supplier = $supplier
        org_sub_name_line = if ($null -ne $orgLine) { [string]$orgLine } else { "" }
        route_stage = $stage
        route_stage_line = if ($null -ne $stageLine) { [string]$stageLine } else { "" }
    }
}

function Get-ExpectedWinningSupplier {
    if (-not [string]::IsNullOrWhiteSpace($ExpectedWinningSupplier)) {
        return $ExpectedWinningSupplier.ToLowerInvariant()
    }

    if ($EnableSubtitleCatTranslatedChineseFallback -and (-not $primarySuppliersSpecified) -and (-not $englishSuppliersSpecified)) {
        return "subtitlecat_translated"
    }

    $explicitPrimary = @(Get-ExplicitRequestedSupplierNames | Where-Object { (ConvertTo-LowerNameSet $PrimaryChineseSuppliers).ContainsKey($_.ToLowerInvariant()) })
    $explicitEnglish = @(Get-ExplicitRequestedSupplierNames | Where-Object { (ConvertTo-LowerNameSet $EnglishFallbackSuppliers).ContainsKey($_.ToLowerInvariant()) })

    if ($summary.route_key -eq "$SampleKind.native_chinese" -and $explicitPrimary.Count -eq 1) {
        return $explicitPrimary[0].ToLowerInvariant()
    }
    if ($summary.route_key -eq "$SampleKind.english_fallback" -and $explicitEnglish.Count -eq 1) {
        return $explicitEnglish[0].ToLowerInvariant()
    }

    return ""
}

$status = Invoke-Json -Method GET -Path "/system-status"
Write-JsonFile $status (Join-Path $roundRoot "system-status.json")
$summary.checks.system_status = "passed"

$login = Invoke-Json -Method POST -Path "/login" -Body @{ username = $Username; password = $Password }
Write-JsonFile $login (Join-Path $roundRoot "login.json")
Assert-True ($login.access_token -ne "") "Login did not return access token."
$token = $login.access_token
$script:LatestAccessToken = $token
$script:TokenForCleanup = $token
$summary.checks.login = "passed"

$baselineSettings = Read-JsonFileUtf8 $configFile
$runtimeSettings = Normalize-BaselineSettings (Clone-JsonObject $baselineSettings)
$runtimeSettings = Apply-SupplierRouteSettings $runtimeSettings
Assert-RequestedSupplierPrerequisites $runtimeSettings
Invoke-Json -Method PUT -Path "/v1/settings" -Token $token -Body $runtimeSettings | Out-Null
Start-Sleep -Seconds 2

$daemonStart = Invoke-Json -Method POST -Path "/v1/daemon/start" -Token $token
Write-JsonFile $daemonStart (Join-Path $roundRoot "daemon-start.json")
Start-Sleep -Seconds 5
$daemonStatus = Invoke-Json -Method GET -Path "/v1/daemon/status" -Token $token
Write-JsonFile $daemonStatus (Join-Path $roundRoot "daemon-status.json")
$summary.checks.daemon_start = "passed"

$settings = Invoke-Json -Method GET -Path "/v1/settings" -Token $token
Write-JsonFile (Copy-SanitizedSettings $settings) (Join-Path $roundRoot "settings-before.json")

Assert-True ($settings.experimental_function.local_chrome_settings.enabled -eq $true) "Docker built-in local Chrome should be enabled by default."
Assert-True ($settings.subtitle_sources.subtitlecat_settings.enabled -eq $true) "SubtitleCat english fallback should stay enabled by default."
$summary.checks.subtitlecat_default = if ($settings.subtitle_sources.subtitlecat_settings.enable_translated_chinese_fallback -eq $false) { "passed" } else { "needs-explicit-switch" }
$summary.checks.llm_default = if ($settings.experimental_function.llm_subtitle_fallback.enable -eq $false) { "passed" } else { "needs-explicit-switch" }
$summary.checks.default_settings = "passed"

if ($EnableSubtitleCatTranslatedChineseFallback -or $EnableLLMFallback) {
    $runtimeSettingsWithFallbacks = Clone-JsonObject $runtimeSettings
    if ($EnableSubtitleCatTranslatedChineseFallback) {
        $runtimeSettingsWithFallbacks.subtitle_sources.subtitlecat_settings.enable_translated_chinese_fallback = $true
    }
    if ($EnableLLMFallback) {
        $runtimeSettingsWithFallbacks.experimental_function.llm_subtitle_fallback.enable = $true
        $runtimeSettingsWithFallbacks.experimental_function.llm_subtitle_fallback.provider = $LLMProvider
        $runtimeSettingsWithFallbacks.experimental_function.llm_subtitle_fallback.base_url = $LLMBaseUrl
        $runtimeSettingsWithFallbacks.experimental_function.llm_subtitle_fallback.api_key = $LLMApiKey
        $runtimeSettingsWithFallbacks.experimental_function.llm_subtitle_fallback.model = $LLMModel
        $runtimeSettingsWithFallbacks.experimental_function.llm_subtitle_fallback.only_when_no_chinese_candidate = $true
    }
    $updated = Invoke-Json -Method PUT -Path "/v1/settings" -Token $token -Body $runtimeSettingsWithFallbacks
    Write-JsonFile $updated (Join-Path $roundRoot "settings-update-response.json")
    $summary.checks.settings_update = "passed"
    $runtimeSettings = $runtimeSettingsWithFallbacks
}

$settingsAfter = Invoke-Json -Method GET -Path "/v1/settings" -Token $token
Write-JsonFile (Copy-SanitizedSettings $settingsAfter) (Join-Path $roundRoot "settings-after.json")
Assert-RouteSettings $settingsAfter
if ($EnableSubtitleCatTranslatedChineseFallback) {
    Assert-True ($settingsAfter.subtitle_sources.subtitlecat_settings.enable_translated_chinese_fallback -eq $true) "SubtitleCat translated switch did not persist."
}
if ($EnableLLMFallback) {
    Assert-True ($settingsAfter.experimental_function.llm_subtitle_fallback.enable -eq $true) "LLM fallback switch did not persist."
}
$summary.checks.route_settings = "passed"

$jobStartedAt = Get-Date
$resolvedVideo = Resolve-VideoPaths
$videoPath = $resolvedVideo.host_video_path
$containerVideoPath = $resolvedVideo.container_video_path
Write-JsonFile $resolvedVideo (Join-Path $roundRoot "sample-video.json")
Assert-ContainerVideoPathReady $resolvedVideo
$beforeSubFiles = @(Get-SubtitleFilesForVideo $videoPath)
Write-JsonFile @($beforeSubFiles | ForEach-Object {
    [ordered]@{
        full_name = $_.FullName
        name = $_.Name
        last_write_time = $_.LastWriteTime.ToString("s")
        length = $_.Length
    }
}) (Join-Path $roundRoot "subtitle-files-before.json")

$jobRequest = [ordered]@{
    video_type = if ($SampleKind -eq "series") { 1 } else { 0 }
    physical_video_file_full_path = $containerVideoPath
    task_priority_level = 3
    media_server_inside_video_id = if ($SampleKind -eq "series") { "local-candidate-series-001" } else { "local-candidate-movie-001" }
    is_bluray = $false
}
$jobRequestJson = $jobRequest | ConvertTo-Json -Depth 20
[System.IO.File]::WriteAllText((Join-Path $roundRoot "add-job-request.json"), $jobRequestJson, [System.Text.UTF8Encoding]::new($false))
$job = Invoke-Json -Method POST -Path "/v1/video/list/add" -Token $token -Body $jobRequest
Write-JsonFile $job (Join-Path $roundRoot "add-job-response.json")
Assert-True ($null -ne $job) "Add job returned null response."
Assert-True ($job -is [psobject]) "Add job returned unexpected response type: $($job.GetType().FullName)"
Assert-True (-not [string]::IsNullOrWhiteSpace([string]$job.job_id)) ("Add job did not return job_id. message=" + [string]$job.message)
$summary.checks.add_job = "passed"

$finalStatus = Wait-ForJob -Token $token -JobId $job.job_id -RoundRoot $roundRoot
Write-JsonFile $finalStatus (Join-Path $roundRoot "job-final-status.json")
$jobStatus = [int]$finalStatus.job_status
$summary.job_terminal_status = $jobStatus
$summary.job_error_info = [string]$finalStatus.error_info
$acceptNoSubFoundHit = $AcceptNoSubFound -and $jobStatus -eq 2 -and ([string]$finalStatus.error_info) -eq "No Sub Found"
$summary.checks.job_poll = if ($acceptNoSubFoundHit) { "no_sub_found_expected" } else { "passed" }
Assert-True ($jobStatus -eq 3 -or $jobStatus -eq 5 -or $acceptNoSubFoundHit) "Job did not reach a successful terminal state. status=$($finalStatus.job_status) error=$($finalStatus.error_info)"

if ($acceptNoSubFoundHit) {
    Write-JsonFile @() (Join-Path $roundRoot "subtitle-files.json")
    Write-JsonFile @() (Join-Path $roundRoot "subtitle-files-after-all.json")
    Write-JsonFile @() (Join-Path $roundRoot "copied-subtitle-artifacts.json")
    $summary.subtitle_file_count = 0
    $summary.final_output_has_chinese = $false
    $summary.route_key = Resolve-RouteKey -SampleKind $SampleKind -EnableSubtitleCatTranslatedChineseFallback:$EnableSubtitleCatTranslatedChineseFallback -EnableLLMFallback:$EnableLLMFallback -NoSubFoundAccepted:$true -HasChineseSample:$false
    $summary.checks.no_sub_safe_failure = "passed"
    $summary.completed_at = (Get-Date).ToString("s")
    Write-JsonFile $summary (Join-Path $roundRoot "e2e-summary.json")

    Restore-RoundSettings
    Release-RunLock

    Write-Host "Local E2E matrix report: $roundRoot"
    exit 0
}

$subFiles = @(Get-NewSubtitleFilesForVideo -VideoPath $videoPath -StartedAt $jobStartedAt)
$subtitleReports = @($subFiles | ForEach-Object {
    [ordered]@{
        full_name = $_.FullName
        name = $_.Name
        length = $_.Length
        last_write_time = $_.LastWriteTime.ToString("s")
        sample = Get-SubtitleSample $_.FullName
    }
})
Write-JsonFile $subtitleReports (Join-Path $roundRoot "subtitle-files.json")
$allSubtitleReports = @(Get-SubtitleFilesForVideo $videoPath | ForEach-Object {
    [ordered]@{
        full_name = $_.FullName
        name = $_.Name
        length = $_.Length
        last_write_time = $_.LastWriteTime.ToString("s")
    }
})
Write-JsonFile $allSubtitleReports (Join-Path $roundRoot "subtitle-files-after-all.json")
$copiedArtifacts = @()
foreach ($entry in $subFiles) {
    $dest = Join-Path $roundRoot $entry.Name
    Copy-Item -LiteralPath $entry.FullName -Destination $dest -Force
    $copiedArtifacts += $dest
}
Write-JsonFile $copiedArtifacts (Join-Path $roundRoot "copied-subtitle-artifacts.json")
$summary.subtitle_file_count = $subFiles.Count
Assert-True ($subFiles.Count -gt 0) "No new subtitle files were generated for this round."
$summary.checks.subtitle_content = "passed"

$hasChineseSample = $false
foreach ($entry in $subtitleReports) {
    if (Test-HasChineseCharacters $entry.sample) {
        $hasChineseSample = $true
        break
    }
}
$summary.final_output_has_chinese = $hasChineseSample
$supplierEvidence = Get-JobSupplierEvidence -SinceTime $jobStartedAt
Write-JsonFile $supplierEvidence (Join-Path $roundRoot "supplier-evidence.json")
$summary.actual_supplier = [string]$supplierEvidence.actual_supplier
$summary.org_sub_name_line = [string]$supplierEvidence.org_sub_name_line
$summary.route_stage = [string]$supplierEvidence.route_stage
$summary.route_stage_line = [string]$supplierEvidence.route_stage_line
$summary.route_key = Resolve-RouteKey -SampleKind $SampleKind -EnableSubtitleCatTranslatedChineseFallback:$EnableSubtitleCatTranslatedChineseFallback -EnableLLMFallback:$EnableLLMFallback -NoSubFoundAccepted:$false -HasChineseSample:$hasChineseSample -StageEvidence $summary.route_stage

if ($EnableSubtitleCatTranslatedChineseFallback) {
    $summary.checks.subtitlecat_translated_route = if ($hasChineseSample) { "translated_output" } else { "fell_back_to_english" }
}

if ($EnableLLMFallback) {
    Assert-True $hasChineseSample "Translated subtitle output did not contain Chinese characters."
    $summary.checks.llm_output_language = "passed"
}
$expectedWinningSupplier = Get-ExpectedWinningSupplier
if (-not [string]::IsNullOrWhiteSpace($expectedWinningSupplier)) {
    Assert-True (-not [string]::IsNullOrWhiteSpace([string]$summary.actual_supplier)) "Expected supplier $expectedWinningSupplier but no OrgSubName supplier evidence was captured."
    Assert-True ([string]$summary.actual_supplier -eq $expectedWinningSupplier) "Supplier assertion failed. expected=$expectedWinningSupplier actual=$($summary.actual_supplier)"
    $summary.checks.supplier_assertion = "passed"
}
if (-not [string]::IsNullOrWhiteSpace($ExpectedRouteKey)) {
    Assert-True ($summary.route_key -eq $ExpectedRouteKey) "Route assertion failed. expected=$ExpectedRouteKey actual=$($summary.route_key)"
    $summary.checks.route_assertion = "passed"
}
$summary.completed_at = (Get-Date).ToString("s")
Write-JsonFile $summary (Join-Path $roundRoot "e2e-summary.json")

if (-not $KeepGeneratedSubtitles) {
    Remove-GeneratedSubtitleFiles $subFiles
}

Restore-RoundSettings
Release-RunLock

Write-Host "Local E2E matrix report: $roundRoot"
exit 0
