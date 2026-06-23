param(
    [string]$WorkspaceRoot = "C:\Users\yang\Desktop\csf\ChineseSubFinder-provider-pack",
    [string]$SandboxRoot = "D:\tmp\csf-dind-sandbox",
    [string]$DindContainer = "csf-dind-sandbox",
    [string]$DindImage = "docker:29-dind",
    [string]$CandidateImage = "chinesesubfinder:isolated-candidate",
    [string]$CandidateContainer = "chinesesubfinder-isolated-candidate",
    [int]$Port = 19235,
    [int]$StaticPort = 19237,
    [string]$DockerMoviesSource = "",
    [string]$DockerSeriesSource = "",
    [switch]$BuildOnly,
    [switch]$NoCleanup
)

$ErrorActionPreference = "Stop"

function Ensure-Dir([string]$Path) {
    if (-not (Test-Path -LiteralPath $Path)) {
        New-Item -ItemType Directory -Path $Path -Force | Out-Null
    }
}

function Test-PathUnder([string]$Path, [string]$Root) {
    $pathFull = [System.IO.Path]::GetFullPath($Path)
    $rootFull = [System.IO.Path]::GetFullPath($Root)
    return $pathFull.StartsWith($rootFull, [System.StringComparison]::OrdinalIgnoreCase)
}

function Remove-PathSafe([string]$Path, [string]$Root) {
    if (-not (Test-Path -LiteralPath $Path)) {
        return
    }
    if (-not (Test-PathUnder $Path $Root)) {
        throw "Refusing to remove path outside sandbox root: $Path"
    }
    Remove-Item -LiteralPath $Path -Recurse -Force
}

function Invoke-Checked([string]$Command, [string]$LogPath) {
    $previous = Get-Location
    try {
        Set-Location -LiteralPath $WorkspaceRoot
        cmd /c $Command *> $LogPath
        if ($LASTEXITCODE -ne 0) {
            throw "Command failed: $Command. See $LogPath"
        }
    } finally {
        Set-Location $previous
    }
}

function Wait-ForDockerHost([string]$Host, [int]$TimeoutSeconds = 120) {
    $deadline = (Get-Date).AddSeconds($TimeoutSeconds)
    while ((Get-Date) -lt $deadline) {
        try {
            docker --host $Host version | Out-Null
            return
        } catch {
            Start-Sleep -Seconds 3
        }
    }
    throw "Timed out waiting for Docker host $Host"
}

function Wait-ForCandidate([int]$PortToCheck, [int]$TimeoutSeconds = 180) {
    $deadline = (Get-Date).AddSeconds($TimeoutSeconds)
    $lastError = ""
    while ((Get-Date) -lt $deadline) {
        try {
            return Invoke-RestMethod -Method Get -Uri "http://127.0.0.1:$PortToCheck/system-status" -TimeoutSec 20
        } catch {
            $lastError = $_.Exception.Message
            Start-Sleep -Seconds 5
        }
    }
    throw "Candidate did not answer /system-status on $PortToCheck. Last error: $lastError"
}

function Cleanup-Sandbox {
    $existing = docker ps -a --format "{{.Names}}" 2>$null | Where-Object { $_ -eq $DindContainer }
    if ($null -ne $existing) {
        docker rm -f $DindContainer 2>$null | Out-Null
    }
    Remove-PathSafe -Path $SandboxRoot -Root $SandboxRoot
}

Ensure-Dir $SandboxRoot
Ensure-Dir (Join-Path $SandboxRoot "docker-data")
Ensure-Dir (Join-Path $SandboxRoot "logs")
Ensure-Dir (Join-Path $SandboxRoot "runtime\config")
Ensure-Dir (Join-Path $SandboxRoot "runtime\browser")
Ensure-Dir (Join-Path $SandboxRoot "runtime\media\movies")
Ensure-Dir (Join-Path $SandboxRoot "runtime\media\series")

$dockerHost = "tcp://127.0.0.1:23759"
$buildLog = Join-Path $SandboxRoot "logs\docker-build.log"
$runLog = Join-Path $SandboxRoot "logs\docker-run.log"
$systemStatusPath = Join-Path $SandboxRoot "logs\system-status.json"
$sandboxRootUnix = "/sandbox"
$candidateConfigMount = "${sandboxRootUnix}/runtime/config"
$candidateBrowserMount = "${sandboxRootUnix}/runtime/browser"
$candidateMoviesMount = "${sandboxRootUnix}/runtime/media/movies"
$candidateSeriesMount = "${sandboxRootUnix}/runtime/media/series"
$succeeded = $false

if (-not $NoCleanup) {
    $existing = docker ps -a --format "{{.Names}}" 2>$null | Where-Object { $_ -eq $DindContainer }
    if ($null -ne $existing) {
        docker rm -f $DindContainer 2>$null | Out-Null
    }
}

$dindArgs = @(
    "run", "-d",
    "--privileged",
    "--name", $DindContainer,
    "-p", "23759:2375",
    "-p", "${Port}:${Port}",
    "-p", "${StaticPort}:${StaticPort}",
    "-e", "DOCKER_TLS_CERTDIR=",
    "--mount", "type=bind,source=$((Join-Path $SandboxRoot 'docker-data')),target=/var/lib/docker",
    "--mount", "type=bind,source=$SandboxRoot,target=${sandboxRootUnix}",
    $DindImage,
    "--host=tcp://0.0.0.0:2375"
)
& docker @dindArgs | Out-Null
if ($LASTEXITCODE -ne 0) {
    throw "Failed to start dind sandbox container"
}

try {
    Wait-ForDockerHost -Host $dockerHost

    $buildCmd = "docker --host $dockerHost build --build-arg INSTALL_BROWSER=true -t $CandidateImage ."
    Invoke-Checked -Command $buildCmd -LogPath $buildLog

    if ($BuildOnly) {
        $succeeded = $true
        exit 0
    }

    $movieSource = if ([string]::IsNullOrWhiteSpace($DockerMoviesSource)) { $candidateMoviesMount } else { $DockerMoviesSource }
    $seriesSource = if ([string]::IsNullOrWhiteSpace($DockerSeriesSource)) { $candidateSeriesMount } else { $DockerSeriesSource }

    $runParts = @(
        "docker --host $dockerHost run -d",
        "--name $CandidateContainer",
        "-e TZ=Asia/Shanghai",
        "-e PERMS=false",
        "-e PUID=1026",
        "-e PGID=100",
        "-p ${Port}:19035",
        "-p ${StaticPort}:19037",
        "-v `"${candidateConfigMount}:/config`"",
        "-v `"${candidateBrowserMount}:/root/.cache/rod/browser`"",
        "-v `"${movieSource}:/media/movies`"",
        "-v `"${seriesSource}:/media/series`"",
        $CandidateImage
    )
    Invoke-Checked -Command ($runParts -join " ") -LogPath $runLog

    $status = Wait-ForCandidate -PortToCheck $Port
    $status | ConvertTo-Json -Depth 10 | Set-Content -LiteralPath $systemStatusPath -Encoding UTF8
    $succeeded = $true
} finally {
    if ($succeeded -and -not $NoCleanup) {
        Cleanup-Sandbox
    }
}
