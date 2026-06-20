param(
    [string]$WorkspaceRoot = "C:\Users\yang\Desktop\csf\ChineseSubFinder-provider-pack",
    [string]$CandidateRoot = "D:\tmp\csf-local-candidate",
    [string]$ReportRoot = "D:\tmp\csf-local-candidate\reports",
    [string[]]$RequiredRoutes = @(),
    [switch]$AsJson
)

$ErrorActionPreference = "Stop"
. (Join-Path $WorkspaceRoot "scripts\local_acceptance_matrix_utils.ps1")

function Read-JsonFileUtf8([string]$Path) {
    return Get-Content -LiteralPath $Path -Raw -Encoding UTF8 | ConvertFrom-Json
}

function Get-E2EDirs {
    if (-not (Test-Path -LiteralPath $ReportRoot)) {
        return @()
    }
    return @(Get-ChildItem -LiteralPath $ReportRoot -Directory -Filter "*-e2e-matrix" -ErrorAction SilentlyContinue |
        Sort-Object LastWriteTime -Descending)
}

$dirs = @(Get-E2EDirs)
$proofs = @()

foreach ($dir in $dirs) {
    $summaryPath = Join-Path $dir.FullName "e2e-summary.json"
    if (-not (Test-Path -LiteralPath $summaryPath)) {
        continue
    }

    try {
        $summary = Read-JsonFileUtf8 $summaryPath
    } catch {
        continue
    }

    $routeKey = ""
    if ($null -ne $summary.route_key) {
        $routeKey = [string]$summary.route_key
    }
    if ([string]::IsNullOrWhiteSpace($routeKey)) {
        continue
    }

    $routeAssertion = ""
    if ($null -ne $summary.checks -and $null -ne $summary.checks.route_assertion) {
        $routeAssertion = [string]$summary.checks.route_assertion
    }
    $translatedRoute = ""
    if ($null -ne $summary.checks -and $null -ne $summary.checks.subtitlecat_translated_route) {
        $translatedRoute = [string]$summary.checks.subtitlecat_translated_route
    }

    $proofs += [pscustomobject]@{
        route_key                     = $routeKey
        round_id                      = [string]$summary.round_id
        round_root                    = [string]$summary.round_root
        sample_kind                   = [string]$summary.sample_kind
        sample_base_name              = [string]$summary.sample_base_name
        job_terminal_status           = $summary.job_terminal_status
        final_output_has_chinese      = [bool]$summary.final_output_has_chinese
        route_assertion               = $routeAssertion
        subtitlecat_translated_route  = $translatedRoute
        completed_at                  = [string]$summary.completed_at
        dir_last_write_time           = $dir.LastWriteTime.ToString("s")
        summary_path                  = $summaryPath
    }
}

$routes = @()
$proofsByRoute = $proofs | Group-Object route_key | Sort-Object Name
foreach ($group in $proofsByRoute) {
    $orderedProofs = @($group.Group | Sort-Object completed_at, dir_last_write_time -Descending)
    $routes += [pscustomobject]@{
        route_key     = $group.Name
        proof_count   = $orderedProofs.Count
        latest_proof  = $orderedProofs[0]
        proofs        = $orderedProofs
    }
}

if ($RequiredRoutes.Count -eq 0) {
    $RequiredRoutes = @(Get-AcceptanceRequiredRoutes -WorkspaceRoot $WorkspaceRoot)
}

$presentRoutes = @($routes.route_key)
$missingRoutes = @($RequiredRoutes | Where-Object { $_ -notin $presentRoutes })

$report = [ordered]@{
    generated_at               = (Get-Date).ToString("s")
    candidate_root             = $CandidateRoot
    report_root                = $ReportRoot
    required_routes            = $RequiredRoutes
    present_route_count        = $routes.Count
    missing_required_route_count = $missingRoutes.Count
    missing_required_routes    = $missingRoutes
    coverage_ok                = ($missingRoutes.Count -eq 0)
    routes                     = $routes
}

if ($AsJson) {
    $report | ConvertTo-Json -Depth 8
    exit 0
}

if (-not (Test-Path -LiteralPath $ReportRoot)) {
    New-Item -ItemType Directory -Path $ReportRoot -Force | Out-Null
}

$stamp = Get-Date -Format "yyyyMMdd-HHmmss-fff"
$reportPath = Join-Path $ReportRoot "route-coverage-snapshot-$stamp.json"
$reportJson = $report | ConvertTo-Json -Depth 8
[System.IO.File]::WriteAllText($reportPath, $reportJson, [System.Text.UTF8Encoding]::new($false))

Write-Host "Route coverage snapshot: $reportPath"
Write-Host ""
Write-Host "Present routes: $($routes.Count)"
Write-Host "Missing required routes: $($missingRoutes.Count)"
if ($missingRoutes.Count -gt 0) {
    $missingRoutes | ForEach-Object { Write-Host " - $_" }
}
