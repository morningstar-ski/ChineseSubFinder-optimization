param(
    [string]$WorkspaceRoot = "C:\Users\yang\Desktop\csf\ChineseSubFinder-provider-pack",
    [string]$CandidateRoot = "D:\tmp\csf-local-candidate",
    [string]$CandidateImage = "chinesesubfinder:local-candidate",
    [string]$CandidateContainer = "chinesesubfinder-local-candidate",
    [string]$ExternalMoviesRoot = "",
    [string]$ExternalSeriesRoot = "",
    [string]$DockerMoviesSource = "",
    [string]$DockerSeriesSource = "",
    [string]$ConfigDockerVolume = "",
    [string]$BrowserDockerVolume = "",
    [int]$Port = 19235,
    [int]$StaticPort = 19237,
    [switch]$PrepareOnly,
    [switch]$RunStaticChecks,
    [switch]$BuildImage,
    [switch]$StartContainer,
    [switch]$RunE2EMatrix,
    [switch]$AcceptNoSubFound,
    [string[]]$PrimaryChineseSuppliers = @(),
    [string[]]$EnglishFallbackSuppliers = @(),
    [string]$ExpectedRouteKey = "",
    [string]$ExpectedWinningSupplier = "",
    [switch]$EnableSubtitleCatTranslatedChineseFallback,
    [switch]$EnableLLMFallback,
    [string]$LLMProvider = "deepseek",
    [string]$LLMBaseUrl = "",
    [string]$LLMApiKey = "",
    [string]$LLMModel = "deepseek-v4-flash",
    [int]$JobTimeoutSeconds = 900,
    [string]$SampleSpecPath = "",
    [string]$SampleFolderName = "Local Candidate Matrix (2024)",
    [string]$SampleBaseName = "Local.Candidate.Matrix.2024.1080p.WEB-DL",
    [switch]$AllowDirtyDockerState
)

$ErrorActionPreference = "Stop"
. (Join-Path $PSScriptRoot "local_acceptance_matrix_utils.ps1")

if (-not [string]::IsNullOrWhiteSpace($SampleSpecPath) -and (Test-Path -LiteralPath $SampleSpecPath)) {
    $sampleSpec = Get-Content -LiteralPath $SampleSpecPath -Raw -Encoding UTF8 | ConvertFrom-Json
    if ([string]::IsNullOrWhiteSpace($ExternalMoviesRoot) -and $null -ne $sampleSpec.external_movies_root) {
        $ExternalMoviesRoot = [string]$sampleSpec.external_movies_root
    }
    if ([string]::IsNullOrWhiteSpace($ExternalSeriesRoot) -and $null -ne $sampleSpec.external_series_root) {
        $ExternalSeriesRoot = [string]$sampleSpec.external_series_root
    }
    if ([string]::IsNullOrWhiteSpace($DockerMoviesSource) -and $null -ne $sampleSpec.docker_movies_source) {
        $DockerMoviesSource = [string]$sampleSpec.docker_movies_source
    }
    if ([string]::IsNullOrWhiteSpace($DockerSeriesSource) -and $null -ne $sampleSpec.docker_series_source) {
        $DockerSeriesSource = [string]$sampleSpec.docker_series_source
    }
}

function New-RoundId {
    return (Get-Date -Format "yyyyMMdd-HHmmss-fff")
}

function Ensure-Dir([string]$Path) {
    if (-not (Test-Path -LiteralPath $Path)) {
        New-Item -ItemType Directory -Path $Path -Force | Out-Null
    }
}

function Write-JsonFile($Value, [string]$Path) {
    Write-JsonUtf8File -Value $Value -Path $Path -Depth 12
}

function Read-JsonFileUtf8([string]$Path) {
    return Get-Content -LiteralPath $Path -Raw -Encoding UTF8 | ConvertFrom-Json
}

function Write-BaselineSettingsSnapshot([string]$SourceSettingsPath) {
    $baselinePath = Join-Path (Join-Path $CandidateRoot "artifacts") "baseline-settings.json"
    Copy-Item -LiteralPath $SourceSettingsPath -Destination $baselinePath -Force
}

function Use-ConfigDockerVolume {
    return -not [string]::IsNullOrWhiteSpace($ConfigDockerVolume)
}

function Use-BrowserDockerVolume {
    return -not [string]::IsNullOrWhiteSpace($BrowserDockerVolume)
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
        $CandidateImage,
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
        $CandidateImage,
        "-lc", "cp -f /shadow/ChineseSubFinderSettings.json /volume/ChineseSubFinderSettings.json"
    )
    & docker @dockerArgs | Out-Null
    if ($LASTEXITCODE -ne 0) {
        throw "Failed to sync settings into Docker volume $ConfigDockerVolume"
    }
}

function Merge-MissingSettings($Target, $Defaults) {
    if ($null -eq $Target) {
        return ($Defaults | ConvertTo-Json -Depth 20 | ConvertFrom-Json)
    }
    if ($null -eq $Defaults) {
        return $Target
    }

    $defaultEntries = @()
    if ($Defaults -is [System.Collections.IDictionary]) {
        foreach ($key in $Defaults.Keys) {
            $defaultEntries += [pscustomobject]@{
                Name = [string]$key
                Value = $Defaults[$key]
            }
        }
    } else {
        $defaultEntries = @($Defaults.PSObject.Properties | Where-Object { $_.MemberType -eq "NoteProperty" })
    }

    if ($defaultEntries.Count -eq 0) {
        return $Target
    }

    foreach ($entry in $defaultEntries) {
        $targetProp = $Target.PSObject.Properties[$entry.Name]
        if ($null -eq $targetProp) {
            Add-Member -InputObject $Target -NotePropertyName $entry.Name -NotePropertyValue ($entry.Value | ConvertTo-Json -Depth 20 | ConvertFrom-Json)
            continue
        }

        $defaultValue = $entry.Value
        $targetValue = $targetProp.Value
        if ($null -eq $defaultValue -or $null -eq $targetValue) {
            continue
        }

        $shouldMergeNested = $false
        if ($defaultValue -is [System.Collections.IDictionary]) {
            $shouldMergeNested = $defaultValue.Keys.Count -gt 0
        } else {
            $nestedProps = @($defaultValue.PSObject.Properties | Where-Object { $_.MemberType -eq "NoteProperty" })
            $shouldMergeNested = $nestedProps.Count -gt 0
        }

        if ($shouldMergeNested -and -not ($defaultValue -is [System.Collections.IEnumerable] -and -not ($defaultValue -is [string]) -and -not ($defaultValue -is [System.Collections.IDictionary]))) {
            Merge-MissingSettings -Target $targetValue -Defaults $defaultValue | Out-Null
        }
    }

    return $Target
}

function Convert-ExistingSettingsForLocalRuntime($SettingsObject) {
    $settings = Merge-MissingSettings -Target $SettingsObject -Defaults (New-DefaultSettings)

    $settings.common_settings.movie_paths = @("/media/movies")
    $settings.common_settings.series_paths = @("/media/series")
    $settings.common_settings.local_static_file_port = "19037"

    if ($null -ne $settings.experimental_function -and $null -ne $settings.experimental_function.local_chrome_settings) {
        $settings.experimental_function.local_chrome_settings.enabled = $true
        if ($null -eq $settings.experimental_function.local_chrome_settings.PSObject.Properties["configured"]) {
            Add-Member -InputObject $settings.experimental_function.local_chrome_settings -NotePropertyName "configured" -NotePropertyValue $false
        }
    }

    if ($null -ne $settings.subtitle_sources -and $null -ne $settings.subtitle_sources.subhd_settings) {
        if ($null -eq $settings.subtitle_sources.subhd_settings.PSObject.Properties["ocr_backend"]) {
            Add-Member -InputObject $settings.subtitle_sources.subhd_settings -NotePropertyName "ocr_backend" -NotePropertyValue "ddddocr"
        } else {
            $settings.subtitle_sources.subhd_settings.ocr_backend = "ddddocr"
        }
        if ($null -eq $settings.subtitle_sources.subhd_settings.PSObject.Properties["external_ocr_url"]) {
            Add-Member -InputObject $settings.subtitle_sources.subhd_settings -NotePropertyName "external_ocr_url" -NotePropertyValue ""
        }
    }

    if ($null -ne $settings.experimental_function -and $null -ne $settings.experimental_function.llm_subtitle_fallback) {
        $settings.experimental_function.llm_subtitle_fallback.python_executable = "/opt/csf-ocr/bin/python3"
        $settings.experimental_function.llm_subtitle_fallback.subflow_root_dir = "/opt/subflow"
        $settings.experimental_function.llm_subtitle_fallback.log_dir = "/config/llm-logs"
    }

    return $settings
}

function Invoke-Checked {
    param(
        [string]$Name,
        [string]$Command,
        [string]$WorkingDirectory,
        [string]$LogPath
    )
    Write-Host "Running: $Name"
    $previous = Get-Location
    try {
        Set-Location -LiteralPath $WorkingDirectory
        $previousErrorActionPreference = $ErrorActionPreference
        $ErrorActionPreference = "Continue"
        powershell -NoProfile -ExecutionPolicy Bypass -Command $Command *> $LogPath
        $ErrorActionPreference = $previousErrorActionPreference
        if ($LASTEXITCODE -ne 0) {
            throw "$Name failed. See $LogPath"
        }
    } finally {
        if ($null -ne $previousErrorActionPreference) {
            $ErrorActionPreference = $previousErrorActionPreference
        }
        Set-Location $previous
    }
}

function Read-ResidueReport([string]$Path) {
    $jsonFile = Get-ChildItem -LiteralPath $Path -Filter "residue-audit-*.json" -File |
        Sort-Object LastWriteTime -Descending |
        Select-Object -First 1
    if ($null -eq $jsonFile) {
        throw "No residue json report found in $Path"
    }
    return Read-JsonFileUtf8 $jsonFile.FullName
}

function Assert-CleanDockerState($Report) {
    $badContainers = @($Report.containers | Where-Object { $_.action -eq "delete" })
    $badImages = @($Report.images | Where-Object { $_.action -eq "delete" })
    if (($badContainers.Count -gt 0 -or $badImages.Count -gt 0) -and -not $AllowDirtyDockerState) {
        Write-Host "Dirty Docker state blocks this round."
        Write-Host "Old containers:"
        $badContainers | Format-Table -AutoSize
        Write-Host "Old images:"
        $badImages | Format-Table -AutoSize
        throw "Clean old CSF containers/images first, or rerun with -AllowDirtyDockerState only for prepare/reporting."
    }
}

function Assert-CandidateRootReady {
    if (Use-ConfigDockerVolume) {
        Sync-SettingsFromConfigVolume
    }
    $required = @("tmp", "artifacts")
    if (Use-ConfigDockerVolume) {
        $required += "config-shadow"
    } else {
        $required += "config"
    }
    if (-not (Use-BrowserDockerVolume)) {
        $required += "browser"
    }
    foreach ($relative in $required) {
        $path = Join-Path $CandidateRoot $relative
        if (-not (Test-Path -LiteralPath $path)) {
            throw "Missing candidate root path: $path"
        }
    }
    $settingsPath = Join-Path (Get-SettingsRoot) "ChineseSubFinderSettings.json"
    if (-not (Test-Path -LiteralPath $settingsPath)) {
        throw "Missing candidate settings template: $settingsPath"
    }
}

function Convert-UNCPathToDockerHostMountSource {
    param(
        [string]$UNCPath
    )

    if ([string]::IsNullOrWhiteSpace($UNCPath)) {
        return ""
    }
    if ($UNCPath -notmatch '^[\\]{2}[^\\]+\\[^\\]+') {
        return ""
    }

    $trimmed = $UNCPath.TrimStart('\')
    return "/run/desktop/mnt/host/uC/" + ($trimmed -replace '\\', '/')
}

function Resolve-DockerBindSource {
    param(
        [string]$RequestedDockerSource,
        [string]$ExternalRoot,
        [string]$FallbackLocalRoot
    )

    $requested = [string]$RequestedDockerSource
    if ($requested -like "/run/desktop/mnt/fnos-real/*") {
        $derived = Convert-UNCPathToDockerHostMountSource -UNCPath $ExternalRoot
        if (-not [string]::IsNullOrWhiteSpace($derived)) {
            return $derived
        }
    }
    if (-not [string]::IsNullOrWhiteSpace($requested)) {
        return $requested
    }

    $derivedFromExternal = Convert-UNCPathToDockerHostMountSource -UNCPath $ExternalRoot
    if (-not [string]::IsNullOrWhiteSpace($derivedFromExternal)) {
        return $derivedFromExternal
    }
    if (-not [string]::IsNullOrWhiteSpace($ExternalRoot)) {
        return $ExternalRoot
    }
    return $FallbackLocalRoot
}

function Start-CandidateContainer {
    $existingContainer = docker ps -a --format "{{.Names}}" | Where-Object { $_ -eq $CandidateContainer }
    if ($null -ne $existingContainer) {
        docker rm -f $CandidateContainer | Out-Null
    }
    $movieMountSource = Resolve-DockerBindSource `
        -RequestedDockerSource $DockerMoviesSource `
        -ExternalRoot $ExternalMoviesRoot `
        -FallbackLocalRoot (Join-Path $CandidateRoot "media\movies")
    $seriesMountSource = Resolve-DockerBindSource `
        -RequestedDockerSource $DockerSeriesSource `
        -ExternalRoot $ExternalSeriesRoot `
        -FallbackLocalRoot (Join-Path $CandidateRoot "media\series")

    $dockerArgs = @(
        "run", "-d",
        "--name", $CandidateContainer,
        "-e", "TZ=Asia/Shanghai",
        "-e", "PERMS=false",
        "-e", "PUID=1026",
        "-e", "PGID=100",
        "-e", "CSF_DDDDOCR_PYTHON=/opt/csf-ocr/bin/python3",
        "-e", "CSF_LLM_SUBTITLE_FALLBACK_PYTHON=/opt/csf-ocr/bin/python3",
        "-e", "CSF_LLM_SUBTITLE_FALLBACK_SUBFLOW_ROOT=/opt/subflow",
        "-e", "UMASK=022",
        "-p", "${Port}:19035",
        "-p", "${StaticPort}:19037",
        "-v", "${CandidateRoot}\tmp:/tmp"
    )
    if (Use-ConfigDockerVolume) {
        $dockerArgs += @("-v", "${ConfigDockerVolume}:/config")
    } else {
        $dockerArgs += @("-v", "${CandidateRoot}\config:/config")
    }
    if (Use-BrowserDockerVolume) {
        $dockerArgs += @("-v", "${BrowserDockerVolume}:/root/.cache/rod/browser")
    } else {
        $dockerArgs += @("-v", "${CandidateRoot}\browser:/root/.cache/rod/browser")
    }

    if ($movieMountSource.StartsWith("/")) {
        $dockerArgs += @("--mount", "type=bind,source=$movieMountSource,target=/media/movies")
    } else {
        $dockerArgs += @("-v", "${movieMountSource}:/media/movies")
    }
    if ($seriesMountSource.StartsWith("/")) {
        $dockerArgs += @("--mount", "type=bind,source=$seriesMountSource,target=/media/series")
    } else {
        $dockerArgs += @("-v", "${seriesMountSource}:/media/series")
    }
    $dockerArgs += $CandidateImage
    & docker @dockerArgs | Out-Null
}

function Wait-ForCandidateReady {
    param(
        [int]$Port,
        [int]$TimeoutSeconds = 180
    )
    $deadline = (Get-Date).AddSeconds($TimeoutSeconds)
    $lastError = ""
    while ((Get-Date) -lt $deadline) {
        try {
            $systemStatus = Invoke-RestMethod -Method Get -Uri "http://127.0.0.1:$Port/system-status" -TimeoutSec 20
            return $systemStatus
        } catch {
            $lastError = $_.Exception.Message
            Start-Sleep -Seconds 5
        }
    }
    throw "Candidate container did not answer /system-status on port $Port within $TimeoutSeconds seconds. Last error: $lastError"
}

function New-DefaultSettings {
    return [ordered]@{
        SpeedDevMode = $false
        user_info = [ordered]@{
            username = "localtest"
            password = "localtest123"
        }
        common_settings = [ordered]@{
            interval_or_assign_or_custom = 0
            scan_interval = "@every 6h"
            threads = 1
            run_scan_at_start_up = $false
            movie_paths = @("/media/movies")
            series_paths = @("/media/series")
            local_static_file_port = "19037"
        }
        subtitle_sources = [ordered]@{
            assrt_settings = [ordered]@{ enabled = $false; token = "" }
            subdl_settings = [ordered]@{ enabled = $false; key = "" }
            subhd_settings = [ordered]@{ enabled = $true; ocr_backend = "ddddocr"; external_ocr_url = "" }
            subtitle_best_settings = [ordered]@{ enabled = $false; api_key = "" }
            opensubtitles_settings = [ordered]@{ enabled = $false; api_key = ""; username = ""; password = "" }
            tvsubtitles_settings = [ordered]@{ enabled = $true }
            moviesubtitles_settings = [ordered]@{ enabled = $true }
            subtitlecat_settings = [ordered]@{ enabled = $true; enable_translated_chinese_fallback = $false }
        }
        advanced_settings = [ordered]@{
            proxy_settings = [ordered]@{
                use_proxy = $false
                use_which_proxy_protocol = "http"
                local_http_proxy_server_port = "19036"
                input_proxy_address = "127.0.0.1"
                input_proxy_port = "10809"
                need_pwd = $false
                input_proxy_username = ""
                input_proxy_password = ""
            }
            tmdb_api_settings = [ordered]@{ enable = $false; api_key = ""; use_alternate_base_url = $false }
            debug_mode = $false
            save_full_season_tmp_subtitles = $false
            sub_type_priority = 0
            sub_name_formatter = 0
            save_multi_sub = $false
            custom_video_exts = @()
            fix_time_line = $false
            topic = 1
            scan_logic = [ordered]@{ skip_chinese_movie = $false; skip_chinese_series = $false }
            task_queue = [ordered]@{
                max_retry_times = 3
                one_job_time_out = 300
                interval = 10
                expiration_time = 90
                download_sub_during_x_days = 7
                one_sub_download_interval = 12
                check_pulic_ip_target_site = ""
            }
            download_file_cache = [ordered]@{ ttl = 4320; unit = "hour" }
        }
        emby_settings = [ordered]@{
            enable = $false
            address_url = ""
            api_key = ""
            max_request_video_number = 500
            skip_watched = $false
            movie_paths_mapping = @{}
            series_paths_mapping = @{}
            auto_or_manual = $true
            threads = 4
        }
        developer_settings = [ordered]@{ enable = $false; bark_server_address = "" }
        timeline_fixer_settings = [ordered]@{ max_offset_time = 700; min_offset = 0.2 }
        experimental_function = [ordered]@{
            auto_change_sub_encode = [ordered]@{ enable = $false; des_encode_type = 0 }
            chs_cht_changer = [ordered]@{ enable = $false; des_chinese_language_type = 0 }
            remote_chrome_settings = [ordered]@{ enable = $false; remote_docker_url = ""; remote_user_data_dir = "" }
            api_key_settings = [ordered]@{ enabled = $false; key = "" }
            local_chrome_settings = [ordered]@{ enabled = $true; configured = $false; local_chrome_exe_f_path = "" }
            share_sub_settings = [ordered]@{ share_sub_enabled = $false }
            extend_log = [ordered]@{ SysLog = [ordered]@{ enable = $false; network = ""; address = ""; priority = 0; tag = "" } }
            llm_subtitle_fallback = [ordered]@{
                enable = $false
                provider = "openai"
                base_url = ""
                api_key = ""
                model = "deepseek-v4-flash"
                python_executable = "/opt/csf-ocr/bin/python3"
                subflow_root_dir = "/opt/subflow"
                translate_style = ""
                only_when_no_chinese_candidate = $true
                keep_english_source_copy = $false
                log_dir = "/config/llm-logs"
                source_language = "en"
                target_language = "zh"
            }
        }
    }
}

$roundId = New-RoundId
$reportRoot = Join-Path $CandidateRoot "reports"
$roundRoot = Join-Path $reportRoot $roundId

Ensure-Dir $CandidateRoot
Ensure-Dir $roundRoot
foreach ($name in @("media", "media\movies", "media\series", "tmp", "artifacts")) {
    Ensure-Dir (Join-Path $CandidateRoot $name)
}
if (Use-ConfigDockerVolume) {
    Ensure-Dir (Join-Path $CandidateRoot "config-shadow")
    Sync-SettingsFromConfigVolume
} else {
    Ensure-Dir (Join-Path $CandidateRoot "config")
}
if (-not (Use-BrowserDockerVolume)) {
    Ensure-Dir (Join-Path $CandidateRoot "browser")
}

$settingsPath = Join-Path (Get-SettingsRoot) "ChineseSubFinderSettings.json"
if (-not (Test-Path -LiteralPath $settingsPath)) {
    Write-JsonFile (New-DefaultSettings) $settingsPath
} else {
    $existingSettings = Read-JsonFileUtf8 $settingsPath
    $adaptedSettings = Convert-ExistingSettingsForLocalRuntime $existingSettings
    Write-JsonFile $adaptedSettings $settingsPath
}
if (Use-ConfigDockerVolume) {
    Sync-SettingsToConfigVolume
}
Write-BaselineSettingsSnapshot -SourceSettingsPath $settingsPath

$metadata = [ordered]@{
    round_id = $roundId
    workspace_root = $WorkspaceRoot
    candidate_root = $CandidateRoot
    candidate_image = $CandidateImage
    candidate_container = $CandidateContainer
    external_movies_root = $ExternalMoviesRoot
    external_series_root = $ExternalSeriesRoot
    docker_movies_source = $DockerMoviesSource
    docker_series_source = $DockerSeriesSource
    config_docker_volume = $ConfigDockerVolume
    browser_docker_volume = $BrowserDockerVolume
    port = $Port
    static_port = $StaticPort
    status = "prepared"
}
Write-JsonFile $metadata (Join-Path $roundRoot "round-metadata.json")
Write-JsonFile ([ordered]@{
    phase = "prepare"
    round_id = $roundId
    candidate_root = $CandidateRoot
    candidate_image = $CandidateImage
    candidate_container = $CandidateContainer
    external_movies_root = $ExternalMoviesRoot
    external_series_root = $ExternalSeriesRoot
    docker_movies_source = $DockerMoviesSource
    docker_series_source = $DockerSeriesSource
    config_docker_volume = $ConfigDockerVolume
    browser_docker_volume = $BrowserDockerVolume
    static_checks = [bool]$RunStaticChecks
    build_image = [bool]$BuildImage
    start_container = [bool]$StartContainer
    e2e_matrix = [bool]$RunE2EMatrix
}) (Join-Path $roundRoot "round-state.json")

& powershell -ExecutionPolicy Bypass -File (Join-Path $WorkspaceRoot "scripts\local_residue_audit.ps1") `
    -WorkspaceRoot $WorkspaceRoot `
    -CandidateRoot $CandidateRoot `
    -ReportRoot $roundRoot `
    -CandidateImage $CandidateImage `
    -CandidateContainer $CandidateContainer | Tee-Object -FilePath (Join-Path $roundRoot "pre-run-residue-audit.txt")

$preReport = Read-ResidueReport $roundRoot
Assert-CleanDockerState $preReport
Assert-CandidateRootReady

if ($RunStaticChecks) {
    Invoke-Checked `
        -Name "go targeted tests" `
        -WorkingDirectory $WorkspaceRoot `
        -LogPath (Join-Path $roundRoot "go-targeted-tests.log") `
        -Command "go test ./pkg/sub_helper ./pkg/sub_parser_hub ./pkg/logic/mark_system ./pkg/downloader ./pkg/save_sub_helper ./pkg/settings ./pkg/logic/pre_download_process ./pkg/logic/sub_supplier ./pkg/logic/sub_supplier/subhd -count=1"

    Invoke-Checked `
        -Name "frontend build" `
        -WorkingDirectory (Join-Path $WorkspaceRoot "frontend") `
        -LogPath (Join-Path $roundRoot "frontend-build.log") `
        -Command "npm run build"
}

if ($BuildImage) {
    Invoke-Checked `
        -Name "candidate docker build" `
        -WorkingDirectory $WorkspaceRoot `
        -LogPath (Join-Path $roundRoot "docker-build.log") `
        -Command "docker build --build-arg INSTALL_BROWSER=true -t $CandidateImage ."
}

if ($StartContainer) {
    if (-not $BuildImage -and -not (docker image inspect $CandidateImage 2>$null)) {
        throw "Candidate image $CandidateImage not found. Use -BuildImage first."
    }
    Start-CandidateContainer
    $systemStatus = Wait-ForCandidateReady -Port $Port
    Write-JsonFile $systemStatus (Join-Path $roundRoot "system-status.json")
}

if ($RunE2EMatrix) {
    if (-not $StartContainer) {
        try {
            Invoke-RestMethod -Method Get -Uri "http://127.0.0.1:$Port/system-status" -TimeoutSec 20 | Out-Null
        } catch {
            throw "RunE2EMatrix requires an answering candidate container on port $Port. Use -StartContainer or start it first."
        }
    }
    $e2eArgs = @(
        "-ExecutionPolicy", "Bypass",
        "-File", (Join-Path $WorkspaceRoot "scripts\local_e2e_matrix.ps1"),
        "-CandidateRoot", $CandidateRoot,
        "-CandidateContainer", $CandidateContainer,
        "-BaseUrl", "http://127.0.0.1:$Port",
        "-JobTimeoutSeconds", $JobTimeoutSeconds,
        "-HelperImage", $CandidateImage
    )
    if (-not [string]::IsNullOrWhiteSpace($ConfigDockerVolume)) {
        $e2eArgs += @("-ConfigDockerVolume", $ConfigDockerVolume)
    }
    if (-not [string]::IsNullOrWhiteSpace($SampleSpecPath)) {
        $e2eArgs += @("-SampleSpecPath", $SampleSpecPath)
    } else {
        $e2eArgs += @(
            "-SampleFolderName", $SampleFolderName,
            "-SampleBaseName", $SampleBaseName
        )
    }
    if (-not [string]::IsNullOrWhiteSpace($ExternalMoviesRoot)) {
        $e2eArgs += @("-ExternalMoviesRoot", $ExternalMoviesRoot)
    }
    if (-not [string]::IsNullOrWhiteSpace($ExternalSeriesRoot)) {
        $e2eArgs += @("-ExternalSeriesRoot", $ExternalSeriesRoot)
    }
    if (-not [string]::IsNullOrWhiteSpace($DockerMoviesSource)) {
        $e2eArgs += @("-DockerMoviesSource", $DockerMoviesSource)
    }
    if (-not [string]::IsNullOrWhiteSpace($DockerSeriesSource)) {
        $e2eArgs += @("-DockerSeriesSource", $DockerSeriesSource)
    }
    if ($EnableSubtitleCatTranslatedChineseFallback) {
        $e2eArgs += "-EnableSubtitleCatTranslatedChineseFallback"
    }
    if ($PrimaryChineseSuppliers.Count -gt 0) {
        $e2eArgs += @("-PrimaryChineseSuppliers", $PrimaryChineseSuppliers)
    }
    if ($EnglishFallbackSuppliers.Count -gt 0) {
        $e2eArgs += @("-EnglishFallbackSuppliers", $EnglishFallbackSuppliers)
    }
    if (-not [string]::IsNullOrWhiteSpace($ExpectedRouteKey)) {
        $e2eArgs += @("-ExpectedRouteKey", $ExpectedRouteKey)
    }
    if (-not [string]::IsNullOrWhiteSpace($ExpectedWinningSupplier)) {
        $e2eArgs += @("-ExpectedWinningSupplier", $ExpectedWinningSupplier)
    }
    if ($AcceptNoSubFound) {
        $e2eArgs += "-AcceptNoSubFound"
    }
    if ($EnableLLMFallback) {
        $e2eArgs += @(
            "-EnableLLMFallback",
            "-LLMProvider", $LLMProvider,
            "-LLMBaseUrl", $LLMBaseUrl,
            "-LLMApiKey", $LLMApiKey,
            "-LLMModel", $LLMModel
        )
    }
    Write-Host "Running: local e2e matrix"
    & powershell @e2eArgs *> (Join-Path $roundRoot "local-e2e-matrix.log")
    if ($LASTEXITCODE -ne 0) {
        throw "local e2e matrix failed. See $(Join-Path $roundRoot "local-e2e-matrix.log")"
    }
}

& powershell -ExecutionPolicy Bypass -File (Join-Path $WorkspaceRoot "scripts\local_residue_audit.ps1") `
    -WorkspaceRoot $WorkspaceRoot `
    -CandidateRoot $CandidateRoot `
    -ReportRoot $roundRoot `
    -CandidateImage $CandidateImage `
    -CandidateContainer $CandidateContainer | Tee-Object -FilePath (Join-Path $roundRoot "post-run-residue-audit.txt")

Write-Host ""
Write-Host "Prepared local candidate round: $roundRoot"
if ($PrepareOnly -and -not $RunStaticChecks -and -not $BuildImage -and -not $StartContainer) {
    Write-Host "No container was started and no image was built by this prepare step."
}
