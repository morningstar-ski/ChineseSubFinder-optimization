param(
    [string]$CandidateRoot = "D:\tmp\csf-local-candidate",
    [string]$BaseUrl = "http://127.0.0.1:19235",
    [string]$ConfigDockerVolume = "",
    [string]$HelperImage = "chinesesubfinder:local-candidate",
    [string[]]$SupplierNames = @(),
    [switch]$AsJson
)

$ErrorActionPreference = "Stop"
. (Join-Path $PSScriptRoot "local_acceptance_matrix_utils.ps1")

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

function Expand-SupplierNames([string[]]$Names) {
    $expanded = New-Object System.Collections.Generic.List[string]
    foreach ($name in $Names) {
        if ([string]::IsNullOrWhiteSpace($name)) {
            continue
        }
        foreach ($part in ($name -split ',')) {
            $trimmed = $part.Trim()
            if ([string]::IsNullOrWhiteSpace($trimmed)) {
                continue
            }
            $expanded.Add($trimmed) | Out-Null
        }
    }
    return @($expanded | Select-Object -Unique)
}

function Get-BoolSetting($Object, [string]$PropertyName, [bool]$Default = $false) {
    if ($null -eq $Object) {
        return $Default
    }
    $property = $Object.PSObject.Properties[$PropertyName]
    if ($null -eq $property) {
        return $Default
    }
    return [bool]$property.Value
}

function Get-StringSetting($Object, [string]$PropertyName, [string]$Default = "") {
    if ($null -eq $Object) {
        return $Default
    }
    $property = $Object.PSObject.Properties[$PropertyName]
    if ($null -eq $property) {
        return $Default
    }
    return [string]$property.Value
}

function New-PolicyInfo {
    param(
        [bool]$Primary = $false,
        [bool]$EnglishFallback = $false,
        [bool]$TranslatedChineseFallback = $false,
        [string]$State = "not_registered",
        [string]$Note = ""
    )

    return [ordered]@{
        participates_in_primary_chain               = $Primary
        participates_in_default_english_fallback    = $EnglishFallback
        participates_in_translated_chinese_fallback = $TranslatedChineseFallback
        policy_state                                = $State
        policy_note                                 = $Note
    }
}

function Get-SupplierPolicyInfo($SettingsObject, [string]$SupplierName) {
    $subtitleSources = $SettingsObject.subtitle_sources
    switch ($SupplierName) {
        "xunlei" {
            return (New-PolicyInfo -Primary $true -State "primary" -Note "Always participates in the primary Chinese chain.")
        }
        "shooter" {
            return (New-PolicyInfo -Primary $true -State "primary" -Note "Always participates in the primary Chinese chain.")
        }
        "assrt" {
            $cfg = $subtitleSources.assrt_settings
            $enabled = (Get-BoolSetting $cfg "enabled") -and -not [string]::IsNullOrWhiteSpace((Get-StringSetting $cfg "token"))
            if ($enabled) {
                return (New-PolicyInfo -Primary $true -State "primary" -Note "Participates in the primary Chinese chain when the token is configured.")
            }
            return (New-PolicyInfo -State "disabled" -Note "Requires assrt to be enabled with a token.")
        }
        "subdl" {
            $cfg = $subtitleSources.subdl_settings
            $enabled = (Get-BoolSetting $cfg "enabled") -and -not [string]::IsNullOrWhiteSpace((Get-StringSetting $cfg "key"))
            if ($enabled) {
                return (New-PolicyInfo -EnglishFallback $true -State "english_fallback_only" -Note "Kept out of the primary Chinese chain; default English fallback supplier.")
            }
            return (New-PolicyInfo -State "disabled" -Note "Requires subdl to be enabled with an API key.")
        }
        "subtitle_best" {
            $cfg = $subtitleSources.subtitle_best_settings
            $enabled = (Get-BoolSetting $cfg "enabled") -and -not [string]::IsNullOrWhiteSpace((Get-StringSetting $cfg "api_key"))
            if ($enabled) {
                return (New-PolicyInfo -Primary $true -State "primary" -Note "Primary supplier when enabled; also backs shared subtitle.best support APIs.")
            }
            return (New-PolicyInfo -State "disabled" -Note "Supplier role is disabled here; subtitle.best shared support APIs may still be used by subhd and media-info fallback.")
        }
        "opensubtitles" {
            $cfg = $subtitleSources.opensubtitles_settings
            $enabled = (Get-BoolSetting $cfg "enabled") `
                -and -not [string]::IsNullOrWhiteSpace((Get-StringSetting $cfg "api_key")) `
                -and -not [string]::IsNullOrWhiteSpace((Get-StringSetting $cfg "username")) `
                -and -not [string]::IsNullOrWhiteSpace((Get-StringSetting $cfg "password"))
            if ($enabled) {
                return (New-PolicyInfo -Primary $true -EnglishFallback $true -State "primary_and_english_fallback" -Note "Participates in the primary Chinese chain and can also supply English fallback.")
            }
            return (New-PolicyInfo -State "disabled" -Note "Requires OpenSubtitles to be enabled with full credentials.")
        }
        "tvsubtitles" {
            $cfg = $subtitleSources.tvsubtitles_settings
            if (Get-BoolSetting $cfg "enabled") {
                return (New-PolicyInfo -State "probe_only" -Note "Health checks are allowed, but this supplier is not wired into the backend default English fallback chain.")
            }
            return (New-PolicyInfo -State "disabled" -Note "Supplier is disabled in settings.")
        }
        "moviesubtitles" {
            $cfg = $subtitleSources.moviesubtitles_settings
            if (Get-BoolSetting $cfg "enabled") {
                return (New-PolicyInfo -EnglishFallback $true -State "english_fallback_only" -Note "Movie-only tail English fallback supplier.")
            }
            return (New-PolicyInfo -State "disabled" -Note "Supplier is disabled in settings.")
        }
        "subhd" {
            $cfg = $subtitleSources.subhd_settings
            if (Get-BoolSetting $cfg "enabled") {
                return (New-PolicyInfo -Primary $true -State "primary" -Note "Primary Chinese supplier.")
            }
            return (New-PolicyInfo -State "disabled" -Note "Supplier is disabled in settings.")
        }
        "subtitlecat" {
            $cfg = $subtitleSources.subtitlecat_settings
            if ($null -eq $cfg) {
                return (New-PolicyInfo -State "disabled" -Note "SubtitleCat settings are missing.")
            }
            $translatedEnabled = Get-BoolSetting $cfg "enable_translated_chinese_fallback"
            return (New-PolicyInfo -EnglishFallback $true -TranslatedChineseFallback $translatedEnabled -State ($(if ($translatedEnabled) { "english_and_translated_fallback" } else { "english_fallback_only" })) -Note ($(if ($translatedEnabled) { "Default English fallback supplier with explicit translated-Chinese fallback enabled." } else { "Default English fallback supplier; translated-Chinese fallback requires the explicit switch." })))
        }
        default {
            return (New-PolicyInfo -State "unknown" -Note "No local policy annotation is defined for this supplier in the snapshot script.")
        }
    }
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
}

function Invoke-Json {
    param(
        [ValidateSet("GET", "POST")]
        [string]$Method,
        [string]$Path,
        [object]$Body = $null,
        [string]$Token = ""
    )

    $headers = @{}
    if (-not [string]::IsNullOrWhiteSpace($Token)) {
        $headers.Authorization = "Bearer $Token"
    }
    $uri = $BaseUrl.TrimEnd("/") + $Path
    if ($null -eq $Body) {
        return Invoke-RestMethod -Method $Method -Uri $uri -Headers $headers -TimeoutSec 90
    }
    $jsonBody = $Body | ConvertTo-Json -Depth 20
    $payload = [System.Text.UTF8Encoding]::new($false).GetBytes($jsonBody)
    return Invoke-RestMethod -Method $Method -Uri $uri -Headers $headers -Body $payload -ContentType "application/json; charset=utf-8" -TimeoutSec 90
}

$reportRoot = Join-Path $CandidateRoot "reports"
Ensure-Dir $reportRoot
Sync-SettingsFromConfigVolume
$settingsPath = Join-Path (Get-SettingsRoot) "ChineseSubFinderSettings.json"
if (-not (Test-Path -LiteralPath $settingsPath)) {
    throw "Settings file not found: $settingsPath"
}
$settings = Read-JsonFileUtf8 $settingsPath
$username = [string]$settings.user_info.username
$password = [string]$settings.user_info.password
if ([string]::IsNullOrWhiteSpace($username) -or [string]::IsNullOrWhiteSpace($password)) {
    throw "Login credentials are missing in the current candidate settings."
}

$stamp = Get-Date -Format "yyyyMMdd-HHmmss-fff"
$reportDir = Join-Path $reportRoot ("supplier-status-" + $stamp)
Ensure-Dir $reportDir

$systemStatus = Invoke-Json -Method GET -Path "/system-status"
Write-JsonFile $systemStatus (Join-Path $reportDir "system-status.json")

$login = Invoke-Json -Method POST -Path "/login" -Body @{ username = $username; password = $password }
Write-JsonFile $login (Join-Path $reportDir "login.json")
if ($null -eq $login -or [string]::IsNullOrWhiteSpace([string]$login.access_token)) {
    throw "Login did not return access token."
}
$token = [string]$login.access_token

$settingsResponse = Invoke-Json -Method GET -Path "/v1/settings" -Token $token
$settingsCopy = $settingsResponse | ConvertTo-Json -Depth 20 | ConvertFrom-Json
if ($null -ne $settingsCopy.user_info) {
    $settingsCopy.user_info.password = "******"
}
if ($null -ne $settingsCopy.subtitle_sources) {
    if ($null -ne $settingsCopy.subtitle_sources.assrt_settings) { $settingsCopy.subtitle_sources.assrt_settings.token = "" }
    if ($null -ne $settingsCopy.subtitle_sources.subdl_settings) { $settingsCopy.subtitle_sources.subdl_settings.key = "" }
    if ($null -ne $settingsCopy.subtitle_sources.subtitle_best_settings) { $settingsCopy.subtitle_sources.subtitle_best_settings.api_key = "" }
    if ($null -ne $settingsCopy.subtitle_sources.opensubtitles_settings) {
        $settingsCopy.subtitle_sources.opensubtitles_settings.api_key = ""
        $settingsCopy.subtitle_sources.opensubtitles_settings.username = ""
        $settingsCopy.subtitle_sources.opensubtitles_settings.password = ""
    }
}
if ($null -ne $settingsCopy.experimental_function -and $null -ne $settingsCopy.experimental_function.llm_subtitle_fallback) {
    $settingsCopy.experimental_function.llm_subtitle_fallback.api_key = ""
}
Write-JsonFile $settingsCopy (Join-Path $reportDir "settings.json")

$expandedSupplierNames = @(Expand-SupplierNames $SupplierNames)
$requestBody = if ($expandedSupplierNames.Count -gt 0) {
    @{ supplier_names = @($expandedSupplierNames) }
} else {
    @{ supplier_names = @() }
}
$statusReply = Invoke-Json -Method POST -Path "/v1/check-sub-supplier" -Token $token -Body $requestBody
Write-JsonFile $statusReply (Join-Path $reportDir "supplier-status.json")

$policyWarnings = @()
$summary = [ordered]@{
    generated_at = (Get-Date).ToString("s")
    report_dir = $reportDir
    base_url = $BaseUrl
    supplier_filter = @($expandedSupplierNames)
    total_suppliers = if ($null -ne $statusReply.sub_site_status) { @($statusReply.sub_site_status).Count } else { 0 }
    valid_suppliers = @($statusReply.sub_site_status | Where-Object { $_.valid -eq $true } | ForEach-Object { [string]$_.name })
    invalid_suppliers = @($statusReply.sub_site_status | Where-Object { $_.valid -eq $false } | ForEach-Object { [string]$_.name })
    suppliers = @($statusReply.sub_site_status | ForEach-Object {
        $policyInfo = Get-SupplierPolicyInfo -SettingsObject $settingsResponse -SupplierName ([string]$_.name
        )
        $policyState = [string]$policyInfo.policy_state
        if ([bool]$_.valid -and $policyState -eq "probe_only") {
            $policyWarnings += ("{0}: health probe succeeded but the supplier is not wired into the active default route chain" -f [string]$_.name)
        }
        [ordered]@{
            name = [string]$_.name
            valid = [bool]$_.valid
            enabled = [bool]$_.enabled
            speed = [int64]$_.speed
            runtime_mode = [string]$_.runtime_mode
            reason = [string]$_.reason
            last_checked_at = [string]$_.last_checked_at
            participates_in_primary_chain = [bool]$policyInfo.participates_in_primary_chain
            participates_in_default_english_fallback = [bool]$policyInfo.participates_in_default_english_fallback
            participates_in_translated_chinese_fallback = [bool]$policyInfo.participates_in_translated_chinese_fallback
            policy_state = $policyState
            policy_note = [string]$policyInfo.policy_note
        }
    })
    policy_warnings = @($policyWarnings)
}
Write-JsonFile $summary (Join-Path $reportDir "summary.json")

if ($AsJson) {
    $summary | ConvertTo-Json -Depth 20
    exit 0
}

Write-Host "Supplier status snapshot: $reportDir"
Write-Host ("Valid: {0}" -f (($summary.valid_suppliers -join ", ")))
Write-Host ("Invalid: {0}" -f (($summary.invalid_suppliers -join ", ")))
