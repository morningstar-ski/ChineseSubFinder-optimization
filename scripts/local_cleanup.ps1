param(
    [string]$WorkspaceRoot = "C:\Users\yang\Desktop\csf\ChineseSubFinder-provider-pack",
    [string]$TmpRoot = "D:\tmp",
    [string]$CandidateRoot = "D:\tmp\csf-local-candidate",
    [string]$CandidateImage = "chinesesubfinder:local-candidate",
    [string]$CandidateContainer = "chinesesubfinder-local-candidate",
    [string]$CleanupManifestPath = "D:\tmp\csf-local-candidate\cleanup-proposal.json",
    [switch]$AllowStaleManifest,
    [switch]$Execute
)

$ErrorActionPreference = "Stop"

function Test-PathUnder([string]$Path, [string]$Root) {
    $pathFull = [System.IO.Path]::GetFullPath($Path)
    $rootFull = [System.IO.Path]::GetFullPath($Root)
    return $pathFull.StartsWith($rootFull, [System.StringComparison]::OrdinalIgnoreCase)
}

function Read-JsonFileUtf8([string]$Path) {
    return Get-Content -LiteralPath $Path -Raw -Encoding UTF8 | ConvertFrom-Json
}

function Read-CleanupManifest([string]$Path) {
    if (-not (Test-Path -LiteralPath $Path)) {
        throw "Cleanup manifest not found: $Path"
    }
    $manifest = Read-JsonFileUtf8 $Path
    $reportRoot = Join-Path $CandidateRoot "reports"
    $latestAudit = Get-ChildItem -LiteralPath $reportRoot -Filter "residue-audit-*.json" -File -ErrorAction SilentlyContinue |
        Sort-Object LastWriteTime -Descending |
        Select-Object -First 1
    if (-not $AllowStaleManifest -and $null -ne $latestAudit) {
        $sourceReport = ""
        if ($null -ne $manifest.source_report) {
            $sourceReport = [string]$manifest.source_report
        }
        if ($sourceReport -ne $latestAudit.FullName) {
            throw "Cleanup manifest is stale. Latest audit is $($latestAudit.FullName), manifest points to $sourceReport. Re-run local_residue_audit.ps1 -WriteCleanupManifest first, or pass -AllowStaleManifest."
        }
    }
    return $manifest
}

function Get-DeleteContainers($Manifest) {
    if ($null -ne $Manifest.delete_containers) {
        return @($Manifest.delete_containers)
    }
    return @($Manifest.containers | Where-Object { $_.action -eq "delete" })
}

function Get-DeleteImages($Manifest) {
    if ($null -ne $Manifest.delete_images) {
        return @($Manifest.delete_images)
    }
    return @($Manifest.images | Where-Object { $_.action -eq "delete" })
}

function Get-DeleteTmpItems($Manifest) {
    if ($null -ne $Manifest.delete_tmp_items) {
        return @($Manifest.delete_tmp_items)
    }
    return @($Manifest.tmp_items | Where-Object { $_.action -eq "delete" })
}

function Get-DeleteCandidateInternalItems($Manifest) {
    if ($null -ne $Manifest.delete_candidate_internal_items) {
        return @($Manifest.delete_candidate_internal_items)
    }
    if ($null -ne $Manifest.candidate_internal_items) {
        return @($Manifest.candidate_internal_items | Where-Object { $_.action -eq "delete" })
    }
    return @()
}

function Get-DeleteReportDirs($Manifest) {
    if ($null -ne $Manifest.delete_report_dirs) {
        return @($Manifest.delete_report_dirs)
    }
    return @($Manifest.report_dirs | Where-Object { $_.action -eq "delete" })
}

function Get-DeleteReportFiles($Manifest) {
    if ($null -ne $Manifest.delete_report_files) {
        return @($Manifest.delete_report_files)
    }
    return @($Manifest.report_files | Where-Object { $_.action -eq "delete" })
}

function Stop-AndRemoveContainer([string]$Name) {
    $exists = docker ps -a --format "{{.Names}}" | Where-Object { $_ -eq $Name }
    if ($null -ne $exists) {
        docker rm -f $Name | Out-Null
    }
}

function Remove-ImageTag([string]$Ref) {
    $exists = docker images --format "{{.Repository}}:{{.Tag}}" | Where-Object { $_ -eq $Ref }
    if ($null -ne $exists) {
        docker rmi $Ref | Out-Null
    }
}

function Remove-PathSafe([string]$Path) {
    if (-not (Test-Path -LiteralPath $Path)) {
        return
    }
    if (-not (Test-PathUnder $Path $TmpRoot)) {
        throw "Refusing to remove path outside tmp root: $Path"
    }
    if (Test-PathUnder $Path $ReportRoot) {
        Remove-Item -LiteralPath $Path -Recurse -Force
        return
    }
    if (Test-PathUnder $Path $CandidateRoot) {
        throw "Refusing to remove active candidate root path: $Path"
    }
    Remove-Item -LiteralPath $Path -Recurse -Force
}

function Remove-CandidateInternalPathSafe([string]$Path) {
    if (-not (Test-Path -LiteralPath $Path)) {
        return
    }
    $allowedRoots = @(
        (Join-Path $CandidateRoot "config-prepull-snapshot\llm-logs")
    )
    $allowed = $false
    foreach ($root in $allowedRoots) {
        if (Test-PathUnder $Path $root) {
            $allowed = $true
            break
        }
    }
    if (-not $allowed) {
        throw "Refusing to remove candidate-internal path outside allowed cleanup roots: $Path"
    }
    Remove-Item -LiteralPath $Path -Recurse -Force
}

function Remove-ReportDirWithFallback($Item) {
    $Path = $Item.full_name
    if (Test-Path -LiteralPath $Path) {
        Remove-PathSafe $Path
        return
    }

    $parent = Split-Path -Path $Path -Parent
    $leaf = Split-Path -Path $Path -Leaf
    if (-not (Test-Path -LiteralPath $parent)) {
        return
    }
    if (-not (Test-PathUnder $parent $ReportRoot)) {
        throw "Refusing fallback removal outside report root: $parent"
    }

    $prefix = ""
    $prefixMatch = [regex]::Match($leaf, '^[0-9-]+')
    if ($prefixMatch.Success) {
        $prefix = $prefixMatch.Value
    }

    $candidates = @(Get-ChildItem -LiteralPath $parent -Directory -ErrorAction SilentlyContinue |
        Where-Object {
            ($prefix -ne "" -and $_.Name -like "$prefix*") -or
            ($null -ne $Item.last_write_time -and $_.LastWriteTime.ToString("s") -eq $Item.last_write_time)
        })
    foreach ($candidate in $candidates) {
        Remove-PathSafe $candidate.FullName
    }
}

$manifest = Read-CleanupManifest $CleanupManifestPath
$deleteContainers = @(Get-DeleteContainers $manifest)
$deleteImages = @(Get-DeleteImages $manifest)
$deleteTmpItems = @(Get-DeleteTmpItems $manifest)
$deleteCandidateInternalItems = @(Get-DeleteCandidateInternalItems $manifest)
$ReportRoot = Join-Path $CandidateRoot "reports"
$deleteReportDirs = @(Get-DeleteReportDirs $manifest)
$deleteReportFiles = @(Get-DeleteReportFiles $manifest)

$preview = [ordered]@{
    manifest_path = $CleanupManifestPath
    workspace_root = $WorkspaceRoot
    tmp_root = $TmpRoot
    candidate_root = $CandidateRoot
    execute = [bool]$Execute
    delete_containers = $deleteContainers
    delete_images = $deleteImages
    delete_tmp_items = $deleteTmpItems
    delete_candidate_internal_items = $deleteCandidateInternalItems
    delete_report_dirs = $deleteReportDirs
    delete_report_files = $deleteReportFiles
}

Write-Host ($preview | ConvertTo-Json -Depth 6)

if (-not $Execute) {
    Write-Host "Preview only. Re-run with -Execute after explicit confirmation."
    exit 0
}

foreach ($container in $deleteContainers) {
    if ($null -ne $container.name -and $container.action -eq "delete") {
        Stop-AndRemoveContainer $container.name
    }
}

foreach ($image in $deleteImages) {
    $ref = $null
    if ($null -ne $image.repository -and $null -ne $image.tag) {
        $ref = "$($image.repository):$($image.tag)"
    }
    if ($null -ne $ref -and $image.action -eq "delete") {
        Remove-ImageTag $ref
    }
}

foreach ($item in $deleteTmpItems) {
    if ($null -ne $item.full_name -and $item.action -eq "delete") {
        Remove-PathSafe $item.full_name
    }
}

foreach ($item in $deleteCandidateInternalItems) {
    if ($null -ne $item.full_name -and $item.action -eq "delete") {
        Remove-CandidateInternalPathSafe $item.full_name
    }
}

foreach ($item in $deleteReportDirs) {
    if ($null -ne $item.full_name -and $item.action -eq "delete") {
        Remove-ReportDirWithFallback $item
    }
}

foreach ($item in $deleteReportFiles) {
    if ($null -ne $item.full_name -and $item.action -eq "delete") {
        Remove-PathSafe $item.full_name
    }
}

Write-Host "Cleanup executed."
