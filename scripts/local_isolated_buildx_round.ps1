param(
    [string]$WorkspaceRoot = "C:\Users\yang\Desktop\csf\ChineseSubFinder-provider-pack",
    [string]$SandboxRoot = "D:\tmp\csf-buildx-sandbox",
    [string]$BuilderName = "csf-isolated-builder",
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

function Invoke-CmdChecked([string]$Command, [string]$LogPath) {
    $previous = Get-Location
    try {
        Set-Location -LiteralPath $WorkspaceRoot
        $psi = New-Object System.Diagnostics.ProcessStartInfo
        $psi.FileName = "cmd.exe"
        $psi.Arguments = "/c $Command"
        $psi.WorkingDirectory = $WorkspaceRoot
        $psi.UseShellExecute = $false
        $psi.RedirectStandardOutput = $true
        $psi.RedirectStandardError = $true
        $psi.CreateNoWindow = $true
        $process = New-Object System.Diagnostics.Process
        $process.StartInfo = $psi
        [void]$process.Start()
        $stdout = $process.StandardOutput.ReadToEnd()
        $stderr = $process.StandardError.ReadToEnd()
        $process.WaitForExit()
        [System.IO.File]::WriteAllText($LogPath, ($stdout + $stderr), [System.Text.UTF8Encoding]::new($false))
        if ($process.ExitCode -ne 0) {
            throw "Command failed: $Command. See $LogPath"
        }
    } finally {
        Set-Location $previous
    }
}

function Remove-ContainerIfExists([string]$Name) {
    $existing = docker ps -a --format "{{.Names}}" 2>$null | Where-Object { $_ -eq $Name }
    if ($null -ne $existing) {
        cmd /c "docker rm -f $Name" *> $null
    }
}

function Remove-ImageIfExists([string]$Ref) {
    $existing = docker images --format "{{.Repository}}:{{.Tag}}" 2>$null | Where-Object { $_ -eq $Ref }
    if ($null -ne $existing) {
        cmd /c "docker rmi $Ref" *> $null
    }
}

function Remove-BuilderIfExists([string]$Name) {
    $existing = docker buildx ls 2>$null | Select-String -Pattern ("^" + [regex]::Escape($Name) + "\b")
    if ($null -ne $existing) {
        cmd /c "docker buildx rm $Name" *> $null
    }
}

Ensure-Dir $SandboxRoot
Ensure-Dir (Join-Path $SandboxRoot "logs")
Ensure-Dir (Join-Path $SandboxRoot "runtime\config")
Ensure-Dir (Join-Path $SandboxRoot "runtime\browser")
Ensure-Dir (Join-Path $SandboxRoot "runtime\media\movies")
Ensure-Dir (Join-Path $SandboxRoot "runtime\media\series")

$buildLog = Join-Path $SandboxRoot "logs\docker-buildx-build.log"
$runLog = Join-Path $SandboxRoot "logs\docker-run.log"
$systemStatusPath = Join-Path $SandboxRoot "logs\system-status.json"
$metadataPath = Join-Path $SandboxRoot "logs\metadata.json"
$builderContainerName = "buildx_buildkit_${BuilderName}0"
$preexistingBuildkitIds = @(
    docker images --format "{{.Repository}}:{{.Tag}}|{{.ID}}" 2>$null |
        Where-Object { $_ -like "moby/buildkit:*|*" } |
        ForEach-Object { ($_ -split "\|", 2)[1] }
)
$builderImageRef = ""
$builderImageId = ""
$succeeded = $false

if (-not $NoCleanup) {
    Remove-ContainerIfExists $CandidateContainer
    Remove-ImageIfExists $CandidateImage
    Remove-BuilderIfExists $BuilderName
}

try {
    docker buildx create --name $BuilderName --driver docker-container --use | Out-Null
    if ($LASTEXITCODE -ne 0) {
        throw "Failed to create buildx builder $BuilderName"
    }

    Start-Sleep -Seconds 3
    $actualBuilderContainer = docker ps -a --format "{{.Names}}" 2>$null |
        Where-Object { $_ -like "buildx_buildkit_${BuilderName}*" } |
        Select-Object -First 1
    if (-not [string]::IsNullOrWhiteSpace($actualBuilderContainer)) {
        $builderContainerName = $actualBuilderContainer
        $builderImageLine = docker inspect $builderContainerName --format "{{.Config.Image}}|{{.Image}}" 2>$null
        if ($LASTEXITCODE -eq 0 -and -not [string]::IsNullOrWhiteSpace($builderImageLine)) {
            $parts = $builderImageLine -split "\|", 2
            $builderImageRef = $parts[0]
            if ($parts.Count -gt 1) {
                $builderImageId = $parts[1]
            }
        }
    }

    $buildCommand = "docker buildx build --builder $BuilderName --load --build-arg INSTALL_BROWSER=true -t $CandidateImage ."
    Invoke-CmdChecked -Command $buildCommand -LogPath $buildLog

    if ($BuildOnly) {
        $succeeded = $true
        exit 0
    }

    $movieSource = if ([string]::IsNullOrWhiteSpace($DockerMoviesSource)) { Join-Path $SandboxRoot "runtime\media\movies" } else { $DockerMoviesSource }
    $seriesSource = if ([string]::IsNullOrWhiteSpace($DockerSeriesSource)) { Join-Path $SandboxRoot "runtime\media\series" } else { $DockerSeriesSource }
    $runCommand = "docker run -d --name $CandidateContainer -e TZ=Asia/Shanghai -e PERMS=false -e PUID=1026 -e PGID=100 -p $Port`:19035 -p $StaticPort`:19037 -v `"$((Join-Path $SandboxRoot 'runtime\config')):/config`" -v `"$((Join-Path $SandboxRoot 'runtime\browser')):/root/.cache/rod/browser`" -v `"${movieSource}:/media/movies`" -v `"${seriesSource}:/media/series`" $CandidateImage"
    Invoke-CmdChecked -Command $runCommand -LogPath $runLog

    $status = Wait-ForCandidate -PortToCheck $Port
    $status | ConvertTo-Json -Depth 10 | Set-Content -LiteralPath $systemStatusPath -Encoding UTF8
    [ordered]@{
        sandbox_root = $SandboxRoot
        builder_name = $BuilderName
        builder_container = $builderContainerName
        builder_image_ref = $builderImageRef
        builder_image_id = $builderImageId
        candidate_image = $CandidateImage
        candidate_container = $CandidateContainer
        port = $Port
        static_port = $StaticPort
    } | ConvertTo-Json -Depth 10 | Set-Content -LiteralPath $metadataPath -Encoding UTF8
    $succeeded = $true
} finally {
    if ($succeeded -and -not $NoCleanup) {
        Remove-ContainerIfExists $CandidateContainer
        Remove-ImageIfExists $CandidateImage
        Remove-BuilderIfExists $BuilderName
        if (-not [string]::IsNullOrWhiteSpace($builderImageRef) -and -not [string]::IsNullOrWhiteSpace($builderImageId) -and ($preexistingBuildkitIds -notcontains $builderImageId)) {
            Remove-ImageIfExists $builderImageRef
        }
        Remove-PathSafe -Path $SandboxRoot -Root $SandboxRoot
    }
}
