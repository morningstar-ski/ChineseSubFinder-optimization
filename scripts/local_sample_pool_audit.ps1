param(
    [string]$SamplePoolRoot = "D:\tmp\csf-real-media-stage",
    [string]$ReportRoot = "D:\tmp\csf-local-candidate\reports",
    [string]$HelperImage = "chinesesubfinder:local-candidate",
    [switch]$AsJson
)

$ErrorActionPreference = "Stop"
. (Join-Path $PSScriptRoot "local_acceptance_matrix_utils.ps1")

function Ensure-Dir([string]$Path) {
    if (-not (Test-Path -LiteralPath $Path)) {
        New-Item -ItemType Directory -Path $Path -Force | Out-Null
    }
}

function Read-JsonFileUtf8([string]$Path) {
    return Get-Content -LiteralPath $Path -Raw -Encoding UTF8 | ConvertFrom-Json
}

function Write-JsonFile($Value, [string]$Path) {
    Write-JsonUtf8File -Value $Value -Path $Path -Depth 12
}

function Add-UniqueValue($Set, [string]$Value) {
    if ([string]::IsNullOrWhiteSpace($Value)) {
        return
    }
    $Set[$Value] = $true
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
        [string]$ExternalRoot
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
    return $ExternalRoot
}

function Test-DockerVisibleSample {
    param(
        [string]$MountSource,
        [string]$RelativeVideoPath
    )

    if ([string]::IsNullOrWhiteSpace($MountSource)) {
        return [ordered]@{
            visible = $false
            error   = "mount source is empty"
        }
    }

    $linuxRelative = ($RelativeVideoPath -replace '\\', '/').TrimStart('/')
    $escapedRelative = $linuxRelative.Replace("'", "'""'""'")
    $dockerArgs = @("run", "--rm")
    if ($MountSource.StartsWith("/")) {
        $dockerArgs += @("--mount", "type=bind,source=$MountSource,target=/mnt,readonly")
    } else {
        $dockerArgs += @("-v", "${MountSource}:/mnt:ro")
    }
    $dockerArgs += @("--entrypoint", "sh", $HelperImage, "-lc", "test -f '/mnt/$escapedRelative'")
    $output = & docker @dockerArgs 2>&1
    if ($LASTEXITCODE -eq 0) {
        return [ordered]@{
            visible = $true
            error   = ""
        }
    }

    return [ordered]@{
        visible = $false
        error   = (($output | Out-String).Trim())
    }
}

Ensure-Dir $ReportRoot
$specFiles = @(Get-ChildItem -LiteralPath $SamplePoolRoot -Filter *.json -File -ErrorAction Stop | Sort-Object Name)

$rows = @()
$externalMovieRoots = @{}
$externalSeriesRoots = @{}
$dockerMovieSources = @{}
$dockerSeriesSources = @{}
$hostVideoPathSeen = @{}
$duplicates = @()
$dockerVisibleCount = 0
$dockerInvisibleRows = @()

foreach ($specFile in $specFiles) {
    $spec = Read-JsonFileUtf8 $specFile.FullName
    $sampleKind = if ($null -ne $spec.sample_kind -and -not [string]::IsNullOrWhiteSpace([string]$spec.sample_kind)) {
        [string]$spec.sample_kind
    } else {
        "movie"
    }

    $sampleFolderName = [string]$spec.sample_folder_name
    $sampleBaseName = [string]$spec.sample_base_name
    $externalMoviesRoot = [string]$spec.external_movies_root
    $externalSeriesRoot = [string]$spec.external_series_root
    $dockerMoviesSource = [string]$spec.docker_movies_source
    $dockerSeriesSource = [string]$spec.docker_series_source

    Add-UniqueValue $externalMovieRoots $externalMoviesRoot
    Add-UniqueValue $externalSeriesRoots $externalSeriesRoot
    Add-UniqueValue $dockerMovieSources $dockerMoviesSource
    Add-UniqueValue $dockerSeriesSources $dockerSeriesSource

    $externalRoot = if ($sampleKind -eq "series") { $externalSeriesRoot } else { $externalMoviesRoot }
    $dockerSource = if ($sampleKind -eq "series") { $dockerSeriesSource } else { $dockerMoviesSource }
    $resolvedDockerSource = Resolve-DockerBindSource -RequestedDockerSource $dockerSource -ExternalRoot $externalRoot
    $hostVideoPath = Join-Path (Join-Path $externalRoot $sampleFolderName) ($sampleBaseName + ".mkv")
    $relativeVideoPath = Join-Path $sampleFolderName ($sampleBaseName + ".mkv")
    $containerVideoPath = if ($sampleKind -eq "series") {
        "/media/series/" + ($sampleFolderName -replace '\\', '/') + "/" + $sampleBaseName + ".mkv"
    } else {
        "/media/movies/" + ($sampleFolderName -replace '\\', '/') + "/" + $sampleBaseName + ".mkv"
    }

    $hostExists = Test-Path -LiteralPath $hostVideoPath
    $dockerVisibility = if ($hostExists) {
        Test-DockerVisibleSample -MountSource $resolvedDockerSource -RelativeVideoPath $relativeVideoPath
    } else {
        [ordered]@{
            visible = $false
            error   = "host video missing"
        }
    }
    $duplicateKey = $hostVideoPath.ToLowerInvariant()
    $isDuplicateHostVideo = $hostVideoPathSeen.ContainsKey($duplicateKey)
    if ($isDuplicateHostVideo) {
        $duplicates += $specFile.Name
    } else {
        $hostVideoPathSeen[$duplicateKey] = $true
    }
    if ($dockerVisibility.visible) {
        $dockerVisibleCount += 1
    } else {
        $dockerInvisibleRows += [pscustomobject]@{
            spec_name          = $specFile.Name
            sample_kind        = $sampleKind
            resolved_mount     = $resolvedDockerSource
            relative_video_path = ($relativeVideoPath -replace '\\', '/')
            reason             = [string]$dockerVisibility.error
        }
    }

    $rows += [pscustomobject]@{
        spec_name               = $specFile.Name
        sample_kind             = $sampleKind
        sample_folder_name      = $sampleFolderName
        sample_base_name        = $sampleBaseName
        external_root           = $externalRoot
        docker_source           = $dockerSource
        resolved_docker_source  = $resolvedDockerSource
        host_video_path         = $hostVideoPath
        container_video_path    = $containerVideoPath
        host_video_exists       = [bool]$hostExists
        docker_video_visible    = [bool]$dockerVisibility.visible
        docker_visibility_error = [string]$dockerVisibility.error
        duplicate_host_video    = [bool]$isDuplicateHostVideo
    }
}

$missingRows = @($rows | Where-Object { $_.host_video_exists -eq $false })
$report = [ordered]@{
    generated_at                  = (Get-Date).ToString("s")
    sample_pool_root              = $SamplePoolRoot
    report_root                   = $ReportRoot
    total_specs                   = $rows.Count
    existing_host_video_count     = @($rows | Where-Object { $_.host_video_exists }).Count
    missing_host_video_count      = $missingRows.Count
    docker_visible_video_count    = $dockerVisibleCount
    docker_invisible_video_count  = $rows.Count - $dockerVisibleCount
    duplicate_host_video_count    = @($rows | Where-Object { $_.duplicate_host_video }).Count
    unique_external_movie_roots   = @($externalMovieRoots.Keys | Sort-Object)
    unique_external_series_roots  = @($externalSeriesRoots.Keys | Sort-Object)
    unique_docker_movie_sources   = @($dockerMovieSources.Keys | Sort-Object)
    unique_docker_series_sources  = @($dockerSeriesSources.Keys | Sort-Object)
    missing_specs                 = @($missingRows | Select-Object spec_name, sample_kind, host_video_path)
    docker_invisible_specs        = $dockerInvisibleRows
    rows                          = $rows
}

$stamp = Get-Date -Format "yyyyMMdd-HHmmss-fff"
$reportPath = Join-Path $ReportRoot "sample-pool-audit-$stamp.json"
Write-JsonFile $report $reportPath

if ($AsJson) {
    $report | ConvertTo-Json -Depth 12
    exit 0
}

Write-Host "Sample pool audit report: $reportPath"
Write-Host ""
Write-Host ("Specs: {0}, existing: {1}, missing: {2}, duplicates: {3}" -f `
    $report.total_specs, `
    $report.existing_host_video_count, `
    $report.missing_host_video_count, `
    $report.duplicate_host_video_count)
Write-Host ("Docker-visible: {0}, Docker-invisible: {1}" -f `
    $report.docker_visible_video_count, `
    $report.docker_invisible_video_count)
Write-Host ""
Write-Host "External movie roots:"
$report.unique_external_movie_roots | ForEach-Object { Write-Host " - $_" }
Write-Host "External series roots:"
$report.unique_external_series_roots | ForEach-Object { Write-Host " - $_" }
Write-Host "Docker movie sources:"
$report.unique_docker_movie_sources | ForEach-Object { Write-Host " - $_" }
Write-Host "Docker series sources:"
$report.unique_docker_series_sources | ForEach-Object { Write-Host " - $_" }

if ($missingRows.Count -gt 0) {
    Write-Host ""
    Write-Host "Missing specs:"
    $missingRows | Select-Object spec_name, sample_kind, host_video_path | Format-Table -AutoSize
}

if ($dockerInvisibleRows.Count -gt 0) {
    Write-Host ""
    Write-Host "Docker-invisible specs:"
    $dockerInvisibleRows | Select-Object spec_name, sample_kind, resolved_mount, relative_video_path | Format-Table -AutoSize
}
