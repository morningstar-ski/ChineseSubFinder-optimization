param(
    [string]$SourceSamplePoolRoot = "D:\tmp\csf-real-media-stage",
    [string]$OutputRoot = "D:\tmp\csf-real-media-runtime",
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

function Assert-PathUnderTmp([string]$Path) {
    $full = [System.IO.Path]::GetFullPath($Path)
    $tmpRoot = [System.IO.Path]::GetFullPath("D:\tmp")
    if (-not $full.StartsWith($tmpRoot, [System.StringComparison]::OrdinalIgnoreCase)) {
        throw "Refusing to rewrite output outside D:\tmp: $Path"
    }
}

function Copy-IfExists([string]$SourcePath, [string]$DestinationPath, $CopiedFiles) {
    if (-not (Test-Path -LiteralPath $SourcePath)) {
        return
    }
    Ensure-Dir (Split-Path -Parent $DestinationPath)
    Copy-Item -LiteralPath $SourcePath -Destination $DestinationPath -Force
    $CopiedFiles.Add($DestinationPath) | Out-Null
}

Assert-PathUnderTmp $OutputRoot
if (Test-Path -LiteralPath $OutputRoot) {
    Remove-Item -LiteralPath $OutputRoot -Recurse -Force
}

$outputMovieRoot = Join-Path $OutputRoot "movies-root"
$outputSeriesRoot = Join-Path $OutputRoot "series-root"
$outputSpecRoot = Join-Path $OutputRoot "sample-specs"
Ensure-Dir $outputMovieRoot
Ensure-Dir $outputSeriesRoot
Ensure-Dir $outputSpecRoot

$specFiles = @(Get-ChildItem -LiteralPath $SourceSamplePoolRoot -Filter *.json -File -ErrorAction Stop | Sort-Object Name)
$rows = @()
$totalCopiedFiles = 0

foreach ($specFile in $specFiles) {
    $spec = Read-JsonFileUtf8 $specFile.FullName
    $sampleKind = if ($null -ne $spec.sample_kind -and -not [string]::IsNullOrWhiteSpace([string]$spec.sample_kind)) {
        [string]$spec.sample_kind
    } else {
        "movie"
    }
    $sampleFolderName = [string]$spec.sample_folder_name
    $sampleBaseName = [string]$spec.sample_base_name
    $copiedFiles = New-Object 'System.Collections.Generic.List[string]'

    if ($sampleKind -eq "series") {
        $sourceSeriesRoot = [string]$spec.external_series_root
        $sourceSeasonDir = Join-Path $sourceSeriesRoot $sampleFolderName
        $sourceShowDir = Split-Path -Parent $sourceSeasonDir
        $destSeasonDir = Join-Path $outputSeriesRoot $sampleFolderName
        $episodeBase = Join-Path $sourceSeasonDir $sampleBaseName
        $destEpisodeBase = Join-Path $destSeasonDir $sampleBaseName
        $showRelativeDir = Split-Path -Parent $sampleFolderName
        $destShowDir = Join-Path $outputSeriesRoot $showRelativeDir

        Copy-IfExists ($episodeBase + ".mkv") ($destEpisodeBase + ".mkv") $copiedFiles
        Copy-IfExists ($episodeBase + ".nfo") ($destEpisodeBase + ".nfo") $copiedFiles
        Copy-IfExists (Join-Path $sourceSeasonDir "season.nfo") (Join-Path $destSeasonDir "season.nfo") $copiedFiles
        Copy-IfExists (Join-Path $sourceShowDir "tvshow.nfo") (Join-Path $destShowDir "tvshow.nfo") $copiedFiles
    } else {
        $sourceMovieRoot = [string]$spec.external_movies_root
        $sourceMovieDir = Join-Path $sourceMovieRoot $sampleFolderName
        $destMovieDir = Join-Path $outputMovieRoot $sampleFolderName
        $movieBase = Join-Path $sourceMovieDir $sampleBaseName
        $destMovieBase = Join-Path $destMovieDir $sampleBaseName

        Copy-IfExists ($movieBase + ".mkv") ($destMovieBase + ".mkv") $copiedFiles
        Copy-IfExists ($movieBase + ".nfo") ($destMovieBase + ".nfo") $copiedFiles
    }

    $rewrittenSpec = [ordered]@{
        sample_kind = $sampleKind
        sample_folder_name = $sampleFolderName
        sample_base_name = $sampleBaseName
        external_movies_root = $outputMovieRoot
        external_series_root = $outputSeriesRoot
        docker_movies_source = ""
        docker_series_source = ""
    }
    Write-JsonFile $rewrittenSpec (Join-Path $outputSpecRoot $specFile.Name)

    $totalCopiedFiles += $copiedFiles.Count
    $rows += [pscustomobject]@{
        spec_name = $specFile.Name
        sample_kind = $sampleKind
        copied_file_count = $copiedFiles.Count
        copied_files = @($copiedFiles)
    }
}

$report = [ordered]@{
    generated_at = (Get-Date).ToString("s")
    source_sample_pool_root = $SourceSamplePoolRoot
    output_root = $OutputRoot
    output_spec_root = $outputSpecRoot
    total_specs = $rows.Count
    total_copied_files = $totalCopiedFiles
    rows = $rows
}

Write-JsonFile $report (Join-Path $OutputRoot "materialize-report.json")

if ($AsJson) {
    $report | ConvertTo-Json -Depth 12
    exit 0
}

Write-Host "Materialized sample pool: $OutputRoot"
Write-Host "Spec root: $outputSpecRoot"
Write-Host ("Specs: {0}, copied files: {1}" -f $report.total_specs, $report.total_copied_files)
