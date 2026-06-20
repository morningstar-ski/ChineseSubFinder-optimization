param(
    [string]$WorkspaceRoot = "C:\Users\yang\Desktop\csf\ChineseSubFinder-provider-pack",
    [string]$TmpRoot = "D:\tmp",
    [string]$CandidateRoot = "D:\tmp\csf-local-candidate",
    [string]$ReportRoot = "D:\tmp\csf-local-candidate\reports",
    [string]$CandidateLLMLogRoot = "D:\tmp\csf-local-candidate\config-prepull-snapshot\llm-logs",
    [string]$CandidateImage = "chinesesubfinder:local-candidate",
    [string]$CandidateContainer = "chinesesubfinder-local-candidate",
    [int]$KeepLatestReportDirs = 10,
    [int]$KeepLatestResidueAudits = 2,
    [int]$KeepLatestSamplePoolAudits = 1,
    [int]$KeepLatestSubtitleContentAudits = 1,
    [int]$KeepLatestRouteCoverageSnapshots = 1,
    [int]$KeepLatestSupplierStatusDirs = 1,
    [int]$KeepLatestPolicyWarningE2EDirs = 2,
    [int]$KeepLatestLLMTaskDirs = 2,
    [string]$CleanupManifestPath = "D:\tmp\csf-local-candidate\cleanup-proposal.json",
    [switch]$WriteCleanupManifest,
    [switch]$AsJson
)

$ErrorActionPreference = "Stop"
. (Join-Path $WorkspaceRoot "scripts\local_acceptance_matrix_utils.ps1")

function Get-ContainerRows {
    $rows = @()
    $lines = docker ps -a --format "{{.Names}}|{{.Image}}|{{.Status}}|{{.Ports}}" 2>$null
    foreach ($line in $lines) {
        if ([string]::IsNullOrWhiteSpace($line)) { continue }
        $parts = $line -split "\|", 4
        if ($parts.Count -lt 4) { continue }
        if ($parts[1] -like "chinesesubfinder*") {
            $action = "delete"
            $reason = "old chinesesubfinder test container"
            if ($parts[0] -eq $CandidateContainer -and $parts[1] -eq $CandidateImage) {
                $action = "keep"
                $reason = "active local candidate container"
            }
            $rows += [pscustomobject]@{
                name   = $parts[0]
                image  = $parts[1]
                status = $parts[2]
                ports  = $parts[3]
                action = $action
                reason = $reason
            }
        }
    }
    return $rows
}

function Get-ImageRows {
    $rows = @()
    $lines = docker images --format "{{.Repository}}|{{.Tag}}|{{.ID}}|{{.CreatedSince}}|{{.Size}}" 2>$null
    foreach ($line in $lines) {
        if ([string]::IsNullOrWhiteSpace($line)) { continue }
        $parts = $line -split "\|", 5
        if ($parts.Count -lt 5) { continue }
        if ($parts[0] -like "chinesesubfinder*") {
            $imageRef = "$($parts[0]):$($parts[1])"
            $action = "delete"
            $reason = "old chinesesubfinder image tag"
            if ($imageRef -eq $CandidateImage) {
                $action = "keep"
                $reason = "active local candidate image"
            }
            $rows += [pscustomobject]@{
                repository = $parts[0]
                tag        = $parts[1]
                image_id   = $parts[2]
                created    = $parts[3]
                size       = $parts[4]
                action     = $action
                reason     = $reason
            }
        }
    }
    return $rows
}

function Get-TmpRows {
    $patterns = @(
        "csf*"
    )
    $items = foreach ($pattern in $patterns) {
        Get-ChildItem -LiteralPath $TmpRoot -Force -ErrorAction SilentlyContinue | Where-Object { $_.Name -like $pattern }
    }
    $items |
        Sort-Object FullName -Unique |
        ForEach-Object {
            $insideCandidate = $_.FullName.StartsWith($CandidateRoot, [System.StringComparison]::OrdinalIgnoreCase)
            $action = "delete"
            $reason = "old csf temp artifact outside candidate root"
            if ($_.PSIsContainer -and $_.Name -in @("csf-real-media-stage", "csf-real-media-runtime")) {
                $action = "keep"
                $reason = "active local sample pool"
            }
            if ($insideCandidate) {
                $action = "keep"
                $reason = "inside active local candidate root"
            }
            [pscustomobject]@{
                full_name      = $_.FullName
                item_type      = if ($_.PSIsContainer) { "dir" } else { "file" }
                length         = if ($_.PSIsContainer) { $null } else { $_.Length }
                last_write_time = $_.LastWriteTime.ToString("s")
                inside_candidate_root = $insideCandidate
                action         = $action
                reason         = $reason
            }
        }
}

function Get-CandidateInternalRows {
    $rows = @()
    if ([string]::IsNullOrWhiteSpace($CandidateLLMLogRoot) -or -not (Test-Path -LiteralPath $CandidateLLMLogRoot)) {
        return $rows
    }

    $dirs = @(Get-ChildItem -LiteralPath $CandidateLLMLogRoot -Directory -ErrorAction SilentlyContinue |
        Sort-Object LastWriteTime -Descending)
    $keepLookup = @{}
    foreach ($dir in @($dirs | Select-Object -First $KeepLatestLLMTaskDirs)) {
        $keepLookup[$dir.FullName] = $true
    }

    foreach ($dir in $dirs) {
        $sizeBytes = (Get-ChildItem -LiteralPath $dir.FullName -Recurse -File -ErrorAction SilentlyContinue | Measure-Object Length -Sum).Sum
        $rows += [pscustomobject]@{
            full_name = $dir.FullName
            category = "llm_task_dir"
            last_write_time = $dir.LastWriteTime.ToString("s")
            size_mb = [math]::Round(($sizeBytes / 1MB), 2)
            action = if ($keepLookup.ContainsKey($dir.FullName)) { "keep" } else { "delete" }
            reason = if ($keepLookup.ContainsKey($dir.FullName)) {
                "recent llm fallback task evidence"
            } else {
                "old llm fallback task dir inside candidate root"
            }
        }
    }

    return $rows
}

function Read-JsonFileOrNull([string]$Path) {
    if (-not (Test-Path -LiteralPath $Path)) {
        return $null
    }
    try {
        return Get-Content -LiteralPath $Path -Raw -Encoding UTF8 | ConvertFrom-Json
    } catch {
        return $null
    }
}

function Get-ReportDirMetadata([System.IO.DirectoryInfo]$Dir) {
    $isE2E = $Dir.Name -like "*-e2e-matrix"
    $summaryPath = Join-Path $Dir.FullName "e2e-summary.json"
    $failurePath = Join-Path $Dir.FullName "failure.json"
    $logPath = Join-Path $Dir.FullName "local-e2e-matrix.log"
    $summary = Read-JsonFileOrNull $summaryPath

    $sampleKind = ""
    $sampleBaseName = ""
    $routeKey = ""
    $expectedRouteKey = ""
    $jobStatus = $null
    $completedAt = ""
    $failureWritten = $false
    $policyWarningCount = 0
    $actualSupplier = ""
    $requestedIsolationSupplier = ""
    if ($null -ne $summary) {
        if ($summary.PSObject.Properties.Name -contains "sample_kind") {
            $sampleKind = [string]$summary.sample_kind
        }
        if ($summary.PSObject.Properties.Name -contains "sample_base_name") {
            $sampleBaseName = [string]$summary.sample_base_name
        }
        if ($summary.PSObject.Properties.Name -contains "route_key") {
            $routeKey = [string]$summary.route_key
        }
        if ($summary.PSObject.Properties.Name -contains "expected_route_key") {
            $expectedRouteKey = [string]$summary.expected_route_key
        }
        if ($summary.PSObject.Properties.Name -contains "job_terminal_status" -and $null -ne $summary.job_terminal_status) {
            $jobStatus = [int]$summary.job_terminal_status
        }
        if ($summary.PSObject.Properties.Name -contains "completed_at") {
            $completedAt = [string]$summary.completed_at
        }
        if ($summary.PSObject.Properties.Name -contains "failure_written" -and $summary.failure_written -eq $true) {
            $failureWritten = $true
        }
        if ($summary.PSObject.Properties.Name -contains "policy_warnings" -and $null -ne $summary.policy_warnings) {
            $policyWarningCount = @($summary.policy_warnings).Count
        }
        if ($summary.PSObject.Properties.Name -contains "actual_supplier") {
            $actualSupplier = [string]$summary.actual_supplier
        }
        if ($summary.PSObject.Properties.Name -contains "requested_isolation_supplier") {
            $requestedIsolationSupplier = [string]$summary.requested_isolation_supplier
        }
    }

    $effectiveRouteKey = if (-not [string]::IsNullOrWhiteSpace($routeKey)) { $routeKey } else { $expectedRouteKey }
    $effectiveSupplier = if (-not [string]::IsNullOrWhiteSpace($actualSupplier)) { $actualSupplier } else { $requestedIsolationSupplier }

    $hasTerminalStatus = $null -ne $jobStatus -and $jobStatus -in @(2, 3, 5)
    $hasCompletedAt = -not [string]::IsNullOrWhiteSpace($completedAt)
    $validE2E = $isE2E -and $null -ne $summary -and ($hasTerminalStatus -or $hasCompletedAt)
    $evidenceKey = if (-not [string]::IsNullOrWhiteSpace($effectiveRouteKey)) { $effectiveRouteKey } elseif (-not [string]::IsNullOrWhiteSpace($sampleBaseName)) { "$sampleKind::$sampleBaseName" } else { $Dir.Name }

    [pscustomobject]@{
        full_name = $Dir.FullName
        name = $Dir.Name
        last_write_time = $Dir.LastWriteTime
        last_write_time_text = $Dir.LastWriteTime.ToString("s")
        is_e2e = $isE2E
        has_local_log = (Test-Path -LiteralPath $logPath)
        has_failure_file = (Test-Path -LiteralPath $failurePath)
        has_summary_file = (Test-Path -LiteralPath $summaryPath)
        valid_e2e = $validE2E
        sample_kind = $sampleKind
        sample_base_name = $sampleBaseName
        route_key = $effectiveRouteKey
        actual_supplier = $effectiveSupplier
        evidence_key = $evidenceKey
        job_terminal_status = $jobStatus
        failure_written = $failureWritten
        policy_warning_count = $policyWarningCount
    }
}

function Get-ReportRows {
    if (-not (Test-Path -LiteralPath $ReportRoot)) {
        return @()
    }
    $dirs = @(Get-ChildItem -LiteralPath $ReportRoot -Directory -ErrorAction SilentlyContinue |
        Sort-Object LastWriteTime -Descending)
    $records = @($dirs | ForEach-Object { Get-ReportDirMetadata $_ })
    $routePolicies = @(Get-AcceptanceRoutePolicies -WorkspaceRoot $WorkspaceRoot)
    $keepLookup = @{}
    foreach ($policy in $routePolicies) {
        $routeKey = [string]$policy.route_key
        $keepCount = [int]$policy.keep_count
        $routeRecords = @($records |
            Where-Object { $_.valid_e2e -and $_.route_key -eq $routeKey } |
            Sort-Object @{
                Expression = { if ($_.failure_written) { 1 } else { 0 } }
                Descending = $false
            }, @{
                Expression = { $_.last_write_time }
                Descending = $true
            })
        $selectedRecords = @($routeRecords |
            Select-Object -First $keepCount)
        if ($selectedRecords.Count -eq 0) {
            continue
        }
        $extraSupplierRecords = @()
        $selectedSupplierNames = @($selectedRecords |
            Where-Object { -not [string]::IsNullOrWhiteSpace($_.actual_supplier) } |
            ForEach-Object { $_.actual_supplier.ToLowerInvariant() })
        $selectedSupplierSet = @{}
        foreach ($supplierName in $selectedSupplierNames) {
            $selectedSupplierSet[$supplierName] = $true
        }
        $routeRecordsBySupplier = @($routeRecords |
            Where-Object { -not [string]::IsNullOrWhiteSpace($_.actual_supplier) } |
            Group-Object actual_supplier)
        foreach ($supplierGroup in $routeRecordsBySupplier) {
            $supplierKey = [string]$supplierGroup.Name
            if ($selectedSupplierSet.ContainsKey($supplierKey.ToLowerInvariant())) {
                continue
            }
            $extraSupplierRecords += @($supplierGroup.Group | Sort-Object @{
                Expression = { if ($_.failure_written) { 1 } else { 0 } }
                Descending = $false
            }, @{
                Expression = { $_.last_write_time }
                Descending = $true
            } | Select-Object -First 1)
        }
        $recordsToKeep = @($selectedRecords + $extraSupplierRecords)
        $extraSupplierLookup = @{}
        foreach ($extraRecord in $extraSupplierRecords) {
            $extraSupplierLookup[$extraRecord.full_name] = $true
        }
        foreach ($record in $recordsToKeep) {
            $keepLookup[$record.full_name] = [pscustomobject]@{
                action = "keep"
                reason = if ($extraSupplierLookup.ContainsKey($record.full_name)) {
                    "retained supplier isolation evidence for route $routeKey via $($record.actual_supplier)"
                } elseif ($record.failure_written) {
                    "retained fallback e2e evidence for route $routeKey"
                } else {
                    "retained clean e2e evidence for route $routeKey"
                }
            }
            $paired = $records |
                Where-Object {
                    $_.is_e2e -eq $false -and
                    $_.has_local_log -eq $true -and
                    -not $keepLookup.ContainsKey($_.full_name)
                } |
                Sort-Object @{ Expression = { [Math]::Abs(($_.last_write_time - $record.last_write_time).TotalSeconds) } }, @{ Expression = { $_.last_write_time }; Descending = $true } |
                Select-Object -First 1
            if ($null -ne $paired -and [Math]::Abs(($record.last_write_time - $paired.last_write_time).TotalSeconds) -le 900) {
                $keepLookup[$paired.full_name] = [pscustomobject]@{
                    action = "keep"
                    reason = "paired round root for route $routeKey"
                }
            }
        }
    }

    if ($KeepLatestSupplierStatusDirs -gt 0) {
        $supplierStatusDirs = @($records |
            Where-Object { $_.name -like "supplier-status-*" } |
            Sort-Object @{ Expression = { $_.last_write_time }; Descending = $true } |
            Select-Object -First $KeepLatestSupplierStatusDirs)
        foreach ($record in $supplierStatusDirs) {
            if (-not $keepLookup.ContainsKey($record.full_name)) {
                $keepLookup[$record.full_name] = [pscustomobject]@{
                    action = "keep"
                    reason = "recent supplier status evidence"
                }
            }
        }
    }

    if ($KeepLatestPolicyWarningE2EDirs -gt 0) {
        $policyWarningDirs = @($records |
            Where-Object { $_.is_e2e -and $_.valid_e2e -and $_.policy_warning_count -gt 0 } |
            Sort-Object @{ Expression = { $_.last_write_time }; Descending = $true } |
            Select-Object -First $KeepLatestPolicyWarningE2EDirs)
        foreach ($record in $policyWarningDirs) {
            if (-not $keepLookup.ContainsKey($record.full_name)) {
                $keepLookup[$record.full_name] = [pscustomobject]@{
                    action = "keep"
                    reason = "recent policy-warning e2e evidence"
                }
            }
        }
    }

    $rows = @()
    foreach ($dir in $records) {
        $action = "delete"
        $reason = "old local acceptance report directory"
        if ($keepLookup.ContainsKey($dir.full_name)) {
            $action = $keepLookup[$dir.full_name].action
            $reason = $keepLookup[$dir.full_name].reason
        } elseif ($dir.is_e2e -and $dir.has_summary_file -and $dir.valid_e2e -eq $false) {
            $reason = "invalid or incomplete e2e probe directory"
        } elseif ($dir.is_e2e -and $dir.valid_e2e) {
            $reason = "older duplicate or lower-priority e2e evidence"
        } elseif ($dir.has_local_log) {
            $reason = "older unpaired round root"
        }
        $rows += [pscustomobject]@{
            full_name       = $dir.full_name
            last_write_time = $dir.last_write_time_text
            action          = $action
            reason          = $reason
        }
    }
    return $rows
}

function Get-ReportFileRows {
    if (-not (Test-Path -LiteralPath $ReportRoot)) {
        return @()
    }

    $files = @(Get-ChildItem -LiteralPath $ReportRoot -File -ErrorAction SilentlyContinue |
        Sort-Object LastWriteTime -Descending)
    $auditKept = 0
    $samplePoolAuditKept = 0
    $subtitleContentAuditKept = 0
    $routeCoverageSnapshotKept = 0
    $rows = @()

    foreach ($file in $files) {
        $action = "delete"
        $reason = "old report-side artifact"

        if ($file.Name -like "residue-audit-*.json") {
            if ($auditKept -lt $KeepLatestResidueAudits) {
                $action = "keep"
                $reason = "recent residue audit evidence"
                $auditKept++
            } else {
                $reason = "old residue audit artifact"
            }
        } elseif ($file.Name -like "sample-pool-audit-*.json") {
            if ($samplePoolAuditKept -lt $KeepLatestSamplePoolAudits) {
                $action = "keep"
                $reason = "recent sample pool audit evidence"
                $samplePoolAuditKept++
            } else {
                $reason = "old sample pool audit artifact"
            }
        } elseif ($file.Name -like "subtitle-content-audit-*.json") {
            if ($subtitleContentAuditKept -lt $KeepLatestSubtitleContentAudits) {
                $action = "keep"
                $reason = "recent subtitle content audit evidence"
                $subtitleContentAuditKept++
            } else {
                $reason = "old subtitle content audit artifact"
            }
        } elseif ($file.Name -like "route-coverage-snapshot-*.json") {
            if ($routeCoverageSnapshotKept -lt $KeepLatestRouteCoverageSnapshots) {
                $action = "keep"
                $reason = "recent route coverage snapshot evidence"
                $routeCoverageSnapshotKept++
            } else {
                $reason = "old route coverage snapshot artifact"
            }
        } elseif ($file.Extension -eq ".log") {
            $reason = "old report-side log artifact"
        } else {
            $reason = "old report-side file artifact"
        }

        $rows += [pscustomobject]@{
            full_name       = $file.FullName
            name            = $file.Name
            extension       = $file.Extension
            length          = $file.Length
            last_write_time = $file.LastWriteTime.ToString("s")
            action          = $action
            reason          = $reason
        }
    }

    return $rows
}

function Ensure-ReportRoot {
    if (-not (Test-Path -LiteralPath $ReportRoot)) {
        New-Item -ItemType Directory -Path $ReportRoot -Force | Out-Null
    }
}

function Write-CleanupManifest($Report, [string]$ReportPath) {
    $manifest = [ordered]@{
        generated_at        = (Get-Date).ToString("s")
        source_report       = $ReportPath
        delete_containers   = @($Report.containers | Where-Object { $_.action -eq "delete" })
        delete_images       = @($Report.images | Where-Object { $_.action -eq "delete" })
        delete_tmp_items    = @($Report.tmp_items | Where-Object { $_.action -eq "delete" })
        delete_candidate_internal_items = @($Report.candidate_internal_items | Where-Object { $_.action -eq "delete" })
        delete_report_dirs  = @($Report.report_dirs | Where-Object { $_.action -eq "delete" })
        delete_report_files = @($Report.report_files | Where-Object { $_.action -eq "delete" })
        keep_tmp_items      = @($Report.tmp_items | Where-Object { $_.action -eq "keep" })
    }
    $manifestJson = $manifest | ConvertTo-Json -Depth 6
    [System.IO.File]::WriteAllText($CleanupManifestPath, $manifestJson, [System.Text.UTF8Encoding]::new($false))
}

$report = [ordered]@{
    generated_at        = (Get-Date).ToString("s")
    workspace_root      = $WorkspaceRoot
    tmp_root            = $TmpRoot
    candidate_root      = $CandidateRoot
    candidate_image     = $CandidateImage
    candidate_container = $CandidateContainer
    containers          = @(Get-ContainerRows)
    images              = @(Get-ImageRows)
    tmp_items           = @(Get-TmpRows)
    candidate_internal_items = @(Get-CandidateInternalRows)
    report_dirs         = @(Get-ReportRows)
    report_files        = @(Get-ReportFileRows)
}

Ensure-ReportRoot
$stamp = Get-Date -Format "yyyyMMdd-HHmmss-fff"
$reportPath = Join-Path $ReportRoot "residue-audit-$stamp.json"
$reportJson = $report | ConvertTo-Json -Depth 6
[System.IO.File]::WriteAllText($reportPath, $reportJson, [System.Text.UTF8Encoding]::new($false))

# Recompute report-side files after the current audit file exists so the keep/delete
# set and cleanup manifest always include this newest evidence file.
$report.report_files = @(Get-ReportFileRows)
$reportJson = $report | ConvertTo-Json -Depth 6
[System.IO.File]::WriteAllText($reportPath, $reportJson, [System.Text.UTF8Encoding]::new($false))

if ($WriteCleanupManifest) {
    Write-CleanupManifest -Report $report -ReportPath $reportPath
}

if ($AsJson) {
    $report | ConvertTo-Json -Depth 6
    exit 0
}

Write-Host "Residue audit report: $reportPath"
Write-Host ""
Write-Host "Containers:"
$report.containers | Format-Table -AutoSize
Write-Host ""
Write-Host "Images:"
$report.images | Format-Table -AutoSize
Write-Host ""
Write-Host "D:\tmp items:"
$report.tmp_items | Format-Table -AutoSize
Write-Host ""
Write-Host "Report directories:"
$report.report_dirs | Format-Table -AutoSize
Write-Host ""
Write-Host "Report files:"
$report.report_files | Format-Table -AutoSize
