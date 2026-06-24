param(
    [string]$WorkspaceRoot = "C:\Users\yang\Desktop\csf\ChineseSubFinder-provider-pack",
    [string]$TmpRoot = "D:\tmp",
    [string]$FrontendHost = "127.0.0.1",
    [int]$FrontendPort = 10001,
    [int]$BackendPort = 19035,
    [string]$BackendBaseUrl = "http://127.0.0.1:19035",
    [string]$FrontendBaseUrl = "http://127.0.0.1:10001",
    [string]$ComposeFile = "compose.source.yaml",
    [string]$GoTestPackages = "./pkg/save_sub_helper ./pkg/cache_center ./pkg/settings ./pkg/chs_cht_changer ./pkg/language",
    [string]$ResidueAuditReportRoot = "D:\tmp\csf-local-candidate\reports",
    [switch]$SkipFrontendReadiness,
    [switch]$SkipBackendReadiness,
    [switch]$SkipResidueAudit
)

$ErrorActionPreference = "Stop"
. (Join-Path $WorkspaceRoot "scripts\local_acceptance_matrix_utils.ps1")

function Ensure-Dir([string]$Path) {
    if (-not (Test-Path -LiteralPath $Path)) {
        New-Item -ItemType Directory -Path $Path -Force | Out-Null
    }
}

function Invoke-Step {
    param(
        [string]$Name,
        [scriptblock]$Action
    )

    $startedAt = Get-Date
    try {
        @(& $Action) | Out-Null
        return [ordered]@{
            name = $Name
            status = "passed"
            started_at = $startedAt.ToString("s")
            completed_at = (Get-Date).ToString("s")
        }
    } catch {
        return [ordered]@{
            name = $Name
            status = "failed"
            started_at = $startedAt.ToString("s")
            completed_at = (Get-Date).ToString("s")
            error = $_.Exception.Message
        }
    }
}

function Wait-ForHttpReady {
    param(
        [string]$Url,
        [int]$TimeoutSeconds = 180
    )

    $deadline = (Get-Date).AddSeconds($TimeoutSeconds)
    $lastError = ""
    while ((Get-Date) -lt $deadline) {
        try {
            return Invoke-RestMethod -Method Get -Uri $Url -TimeoutSec 15
        } catch {
            $lastError = $_.Exception.Message
            Start-Sleep -Seconds 2
        }
    }

    throw "Timed out waiting for $Url. Last error: $lastError"
}

function Test-FrontendEndpoint {
    param(
        [string]$Url
    )

    try {
        $response = Invoke-WebRequest -UseBasicParsing -Uri $Url -TimeoutSec 15
        if ($null -eq $response -or [int]$response.StatusCode -lt 200 -or [int]$response.StatusCode -ge 400) {
            return $null
        }
        return [ordered]@{
            status_code = [int]$response.StatusCode
            title_present = ($response.Content -like "*<title>*")
            content_length = if ($null -ne $response.Content) { $response.Content.Length } else { 0 }
        }
    } catch {
        return $null
    }
}

function Stop-TrackedProcess {
    param(
        [int]$ProcessId
    )

    if ($ProcessId -le 0) {
        return
    }
    try {
        $process = Get-Process -Id $ProcessId -ErrorAction SilentlyContinue
        if ($null -ne $process) {
            Stop-Process -Id $ProcessId -Force -ErrorAction SilentlyContinue
        }
    } catch {
    }
}

Ensure-Dir $TmpRoot
Ensure-Dir $ResidueAuditReportRoot

$stamp = Get-Date -Format "yyyyMMdd-HHmmss-fff"
$auditRoot = Join-Path $TmpRoot "csf-local-delivery-audit-$stamp"
Ensure-Dir $auditRoot

$frontendLogPath = Join-Path $auditRoot "frontend-dev.log"
$reportPath = Join-Path $auditRoot "delivery-audit-report.json"
$frontendRoot = Join-Path $WorkspaceRoot "frontend"
$rootSettingsPath = Join-Path $WorkspaceRoot "ChineseSubFinderSettings.json"

$frontendProcessId = 0
$frontendStartedByScript = $false
$reuseExistingFrontend = $false
$steps = New-Object System.Collections.Generic.List[object]

try {
    $steps.Add((Invoke-Step -Name "workspace_sensitive_config_guard" -Action {
        if (Test-Path -LiteralPath $rootSettingsPath) {
            throw "Workspace root still contains ChineseSubFinderSettings.json. Keep runtime settings under config\\ only."
        }
    })) | Out-Null

    $steps.Add((Invoke-Step -Name "frontend_build" -Action {
        Push-Location $frontendRoot
        try {
            & npm run build
            if ($LASTEXITCODE -ne 0) {
                throw "npm run build failed."
            }
        } finally {
            Pop-Location
        }
    })) | Out-Null

    $steps.Add((Invoke-Step -Name "go_test_targeted" -Action {
        & go test ./pkg/save_sub_helper ./pkg/cache_center ./pkg/settings ./pkg/chs_cht_changer ./pkg/language
        if ($LASTEXITCODE -ne 0) {
            throw "targeted go test failed."
        }
    })) | Out-Null

    if (-not $SkipBackendReadiness) {
        $steps.Add((Invoke-Step -Name "backend_up" -Action {
            & docker compose -f $ComposeFile up -d
            if ($LASTEXITCODE -ne 0) {
                throw "docker compose up failed."
            }
        })) | Out-Null

        $steps.Add((Invoke-Step -Name "backend_readiness" -Action {
            $status = Wait-ForHttpReady -Url "$BackendBaseUrl/system-status" -TimeoutSeconds 180
            Write-JsonUtf8File -Value $status -Path (Join-Path $auditRoot "backend-system-status.json") -Depth 12
        })) | Out-Null
    }

    if (-not $SkipFrontendReadiness) {
        $steps.Add((Invoke-Step -Name "frontend_port_guard" -Action {
            try {
                $occupied = Get-NetTCPConnection -State Listen -LocalAddress $FrontendHost -LocalPort $FrontendPort -ErrorAction Stop
            } catch {
                $occupied = @()
            }
            if ($null -ne $occupied -and @($occupied).Count -gt 0) {
                $probe = Test-FrontendEndpoint -Url $FrontendBaseUrl
                if ($null -eq $probe) {
                    throw "Frontend port $FrontendHost`:$FrontendPort is occupied but the frontend endpoint is not healthy."
                }
                $script:reuseExistingFrontend = $true
                Write-JsonUtf8File -Value ([ordered]@{
                    frontend_url = $FrontendBaseUrl
                    reused_existing_process = $true
                    probe = $probe
                }) -Path (Join-Path $auditRoot "frontend-port-guard.json") -Depth 8
            }
        })) | Out-Null

        $steps.Add((Invoke-Step -Name "frontend_dev_start" -Action {
            if ($script:reuseExistingFrontend) {
                return
            }
            $process = Start-Process -FilePath "powershell.exe" `
                -ArgumentList @(
                    "-NoProfile",
                    "-ExecutionPolicy", "Bypass",
                    "-Command",
                    "Set-Location '$frontendRoot'; `$env:CSF_FRONTEND_HOST='$FrontendHost'; `$env:CSF_FRONTEND_DEV_PORT='$FrontendPort'; `$env:CSF_BACKEND_BASE_URL='$BackendBaseUrl'; npm run dev *>&1 | Tee-Object -FilePath '$frontendLogPath'"
                ) `
                -WindowStyle Hidden `
                -PassThru
            $script:frontendProcessId = $process.Id
            $script:frontendStartedByScript = $true
            Start-Sleep -Seconds 4
            $running = Get-Process -Id $script:frontendProcessId -ErrorAction SilentlyContinue
            if ($null -eq $running) {
                throw "Frontend dev process exited immediately. Check $frontendLogPath"
            }
        })) | Out-Null

        $steps.Add((Invoke-Step -Name "frontend_readiness" -Action {
            $probe = $null
            $deadline = (Get-Date).AddSeconds(180)
            while ((Get-Date) -lt $deadline) {
                $probe = Test-FrontendEndpoint -Url $FrontendBaseUrl
                if ($null -ne $probe) {
                    break
                }
                Start-Sleep -Seconds 2
            }
            if ($null -eq $probe) {
                throw "Frontend endpoint $FrontendBaseUrl did not become healthy."
            }
            Write-JsonUtf8File -Value ([ordered]@{
                frontend_url = $FrontendBaseUrl
                reused_existing_process = $script:reuseExistingFrontend
                probe = $probe
            }) -Path (Join-Path $auditRoot "frontend-readiness.json") -Depth 6
        })) | Out-Null
    }

    if (-not $SkipResidueAudit) {
        $steps.Add((Invoke-Step -Name "residue_audit" -Action {
            & powershell -NoProfile -ExecutionPolicy Bypass `
                -File (Join-Path $WorkspaceRoot "scripts\local_residue_audit.ps1") `
                -WorkspaceRoot $WorkspaceRoot `
                -TmpRoot $TmpRoot `
                -ReportRoot $ResidueAuditReportRoot `
                -WriteCleanupManifest
            if ($LASTEXITCODE -ne 0) {
                throw "local_residue_audit.ps1 failed."
            }
        })) | Out-Null
    }
} finally {
    if ($frontendStartedByScript -and $frontendProcessId -gt 0) {
        Stop-TrackedProcess -ProcessId $frontendProcessId
    }
}

$failedSteps = @($steps | Where-Object { $_.status -ne "passed" })
$stepArray = @($steps.ToArray())
$report = [ordered]@{
    generated_at = (Get-Date).ToString("s")
    workspace_root = $WorkspaceRoot
    audit_root = $auditRoot
    frontend_base_url = $FrontendBaseUrl
    backend_base_url = $BackendBaseUrl
    steps = $stepArray
    success = ($failedSteps.Count -eq 0)
}
Write-JsonUtf8File -Value $report -Path $reportPath -Depth 10

Write-Host "Local delivery audit report: $reportPath"
if ($failedSteps.Count -gt 0) {
    exit 1
}
exit 0
