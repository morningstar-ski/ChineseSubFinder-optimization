param(
    [string]$RemoteHost = "fnos-csf",
    [string]$RemoteConfigDir = "/vol1/1000/docker/csf/config",
    [string]$RemoteBrowserDir = "/vol1/1000/docker/csf/browser",
    [string]$HelperImage = "chinesesubfinder:local-candidate",
    [string]$FullConfigVolume = "csf_fnos_config_full_20260619",
    [string]$FullBrowserVolume = "csf_fnos_browser_full_20260619",
    [string]$WorkingConfigVolume = "csf_fnos_config_working",
    [string]$WorkingBrowserVolume = "csf_fnos_browser_working",
    [string]$ConfigTarPath = "D:\tmp\fnos-csf-config-full-20260619.tar",
    [string]$BrowserTarPath = "D:\tmp\fnos-csf-browser-full-20260619.tar",
    [string]$TempRoot = "D:\tmp",
    [switch]$SkipRefreshWorking
)

$ErrorActionPreference = "Stop"

function Ensure-Dir([string]$Path) {
    if (-not (Test-Path -LiteralPath $Path)) {
        New-Item -ItemType Directory -Path $Path -Force | Out-Null
    }
}

function Export-RemoteDirectoryToTar([string]$RemoteDir, [string]$LocalTarPath) {
    $parentDir = Split-Path -Path $LocalTarPath -Parent
    Ensure-Dir $parentDir
    if (Test-Path -LiteralPath $LocalTarPath) {
        Remove-Item -LiteralPath $LocalTarPath -Force
    }

    $psi = New-Object System.Diagnostics.ProcessStartInfo
    $psi.FileName = "ssh"
    $psi.Arguments = "$RemoteHost ""tar -cf - -C '$RemoteDir' ."""
    $psi.UseShellExecute = $false
    $psi.RedirectStandardOutput = $true
    $psi.RedirectStandardError = $true

    $process = New-Object System.Diagnostics.Process
    $process.StartInfo = $psi
    $null = $process.Start()

    $fileStream = [System.IO.File]::Open($LocalTarPath, [System.IO.FileMode]::Create, [System.IO.FileAccess]::Write, [System.IO.FileShare]::None)
    try {
        $process.StandardOutput.BaseStream.CopyTo($fileStream)
    } finally {
        $fileStream.Dispose()
    }

    $stderr = $process.StandardError.ReadToEnd()
    $process.WaitForExit()
    if ($process.ExitCode -ne 0) {
        throw "Failed to export $RemoteDir from ${RemoteHost}: $stderr"
    }
}

function Get-RemoteSha256([string]$RemotePath) {
    $value = & ssh $RemoteHost "sha256sum '$RemotePath' | awk '{print `$1}'"
    if ($LASTEXITCODE -ne 0 -or [string]::IsNullOrWhiteSpace($value)) {
        throw "Failed to get remote sha256 for $RemotePath"
    }
    return $value.Trim()
}

function Expand-TarToVolume([string]$TarPath, [string]$TargetVolume) {
    if (-not (Test-Path -LiteralPath $TarPath)) {
        throw "Tar file not found: $TarPath"
    }

    docker volume create $TargetVolume | Out-Null
    $tarDir = Split-Path -Path $TarPath -Parent
    $tarName = Split-Path -Path $TarPath -Leaf
    $dockerArgs = @(
        "run", "--rm",
        "-v", "${tarDir}:/tar-src",
        "-v", "${TargetVolume}:/dest",
        "--entrypoint", "sh",
        $HelperImage,
        "-lc", "rm -rf /dest/* /dest/.[!.]* /dest/..?* 2>/dev/null || true; mkdir -p /dest; tar -xf /tar-src/$tarName -C /dest"
    )
    & docker @dockerArgs | Out-Null
    if ($LASTEXITCODE -ne 0) {
        throw "Failed to expand $TarPath into Docker volume $TargetVolume"
    }
}

function Get-VolumeSettingsSha256([string]$VolumeName) {
    $dockerArgs = @(
        "run", "--rm",
        "-v", "${VolumeName}:/volume",
        "--entrypoint", "sh",
        $HelperImage,
        "-lc", "sha256sum /volume/ChineseSubFinderSettings.json | awk '{print `$1}'"
    )
    $value = & docker @dockerArgs
    if ($LASTEXITCODE -ne 0 -or [string]::IsNullOrWhiteSpace($value)) {
        throw "Failed to get settings sha256 from Docker volume $VolumeName"
    }
    return $value.Trim()
}

$stamp = Get-Date -Format "yyyyMMdd-HHmmss"
$manifestPath = Join-Path $TempRoot "fnos-csf-config-pull-$stamp.json"

$remoteSettingsPath = "$RemoteConfigDir/ChineseSubFinderSettings.json"
$remoteConfigYamlPath = "$RemoteConfigDir/config.yaml"

$remoteSettingsSha = Get-RemoteSha256 -RemotePath $remoteSettingsPath
$remoteConfigYamlSha = Get-RemoteSha256 -RemotePath $remoteConfigYamlPath

Export-RemoteDirectoryToTar -RemoteDir $RemoteConfigDir -LocalTarPath $ConfigTarPath
Export-RemoteDirectoryToTar -RemoteDir $RemoteBrowserDir -LocalTarPath $BrowserTarPath

Expand-TarToVolume -TarPath $ConfigTarPath -TargetVolume $FullConfigVolume
Expand-TarToVolume -TarPath $BrowserTarPath -TargetVolume $FullBrowserVolume

$fullVolumeSettingsSha = Get-VolumeSettingsSha256 -VolumeName $FullConfigVolume
if ($fullVolumeSettingsSha -ne $remoteSettingsSha) {
    throw "Full config volume sha mismatch: remote=$remoteSettingsSha volume=$fullVolumeSettingsSha"
}

$manifest = [ordered]@{
    pulled_at = (Get-Date).ToString("s")
    remote_host = $RemoteHost
    remote_config_dir = $RemoteConfigDir
    remote_browser_dir = $RemoteBrowserDir
    config_tar_path = $ConfigTarPath
    browser_tar_path = $BrowserTarPath
    full_config_volume = $FullConfigVolume
    full_browser_volume = $FullBrowserVolume
    working_config_volume = $WorkingConfigVolume
    working_browser_volume = $WorkingBrowserVolume
    remote_settings_sha256 = $remoteSettingsSha
    volume_settings_sha256 = $fullVolumeSettingsSha
    remote_config_yaml_sha256 = $remoteConfigYamlSha
}
$manifest | ConvertTo-Json -Depth 6 | Set-Content -LiteralPath $manifestPath -Encoding UTF8

if (-not $SkipRefreshWorking) {
    & powershell -NoProfile -ExecutionPolicy Bypass -File (Join-Path $PSScriptRoot "refresh_fnos_working_volumes.ps1") `
        -HelperImage $HelperImage `
        -SourceConfigVolume $FullConfigVolume `
        -SourceBrowserVolume $FullBrowserVolume `
        -WorkingConfigVolume $WorkingConfigVolume `
        -WorkingBrowserVolume $WorkingBrowserVolume
    if ($LASTEXITCODE -ne 0) {
        throw "Failed to refresh working volumes from refreshed full baseline"
    }
}

Write-Output "MANIFEST_PATH=$manifestPath"
Write-Output "CONFIG_TAR_PATH=$ConfigTarPath"
Write-Output "BROWSER_TAR_PATH=$BrowserTarPath"
Write-Output "REMOTE_SETTINGS_SHA256=$remoteSettingsSha"
Write-Output "FULL_CONFIG_VOLUME=$FullConfigVolume"
Write-Output "FULL_BROWSER_VOLUME=$FullBrowserVolume"
