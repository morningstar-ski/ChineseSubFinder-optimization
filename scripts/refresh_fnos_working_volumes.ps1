param(
    [string]$HelperImage = "chinesesubfinder:local-candidate",
    [string]$SourceConfigVolume = "csf_fnos_config_full_20260619",
    [string]$SourceBrowserVolume = "csf_fnos_browser_full_20260619",
    [string]$WorkingConfigVolume = "csf_fnos_config_working",
    [string]$WorkingBrowserVolume = "csf_fnos_browser_working",
    [int]$PUID = 1026,
    [int]$PGID = 100
)

$ErrorActionPreference = "Stop"

function Invoke-DockerVolumeCopy([string]$SourceVolume, [string]$TargetVolume) {
    docker volume create $TargetVolume | Out-Null
    docker run --rm `
        -v "${SourceVolume}:/src" `
        -v "${TargetVolume}:/dest" `
        --entrypoint sh `
        $HelperImage `
        -lc "rm -rf /dest/* /dest/.[!.]* /dest/..?* 2>/dev/null || true; cd /src && tar -cf - . | tar -xf - -C /dest"
    if ($LASTEXITCODE -ne 0) {
        throw "Failed to copy Docker volume from $SourceVolume to $TargetVolume"
    }
}

function Set-VolumeOwnership([string]$TargetVolume) {
    $ownershipScript = @"
set -e
chown -R ${PUID}:${PGID} /dest
"@

    docker run --rm `
        -v "${TargetVolume}:/dest" `
        --entrypoint sh `
        $HelperImage `
        -lc $ownershipScript
    if ($LASTEXITCODE -ne 0) {
        throw "Failed to fix ownership in $TargetVolume"
    }
}

function Reset-WorkingConfigRuntimeState([string]$TargetVolume) {
    $cleanupScript = @'
set -e
rm -f /dest/ChineseSubFinder-Cache.db
rm -rf /dest/Logs/*
rm -rf /dest/cache/CSF-DebugThings
rm -rf /dest/cache/CSF-ShareSubCache
rm -rf /dest/cache/CSF-SubFixCache
rm -rf /dest/cache/CSF-VideoAndSubPreviewCache
rm -rf /dest/cache/rod
rm -rf /dest/cache/tmp
rm -rf /dest/cache/CSF-CacheCenter/task_queue
rm -rf /dest/cache/CSF-CacheCenter/download_files
rm -f /dest/cache/CSF-CacheCenter/local_task_queue_cache_center.db
mkdir -p /dest/Logs
mkdir -p /dest/cache/CSF-CacheCenter/task_queue/local_task_queue
mkdir -p /dest/cache/CSF-CacheCenter/download_files/local_task_queue
'@

    docker run --rm `
        -v "${TargetVolume}:/dest" `
        --entrypoint sh `
        $HelperImage `
        -lc $cleanupScript
    if ($LASTEXITCODE -ne 0) {
        throw "Failed to reset runtime state in $TargetVolume"
    }
}

Invoke-DockerVolumeCopy -SourceVolume $SourceConfigVolume -TargetVolume $WorkingConfigVolume
Reset-WorkingConfigRuntimeState -TargetVolume $WorkingConfigVolume
Set-VolumeOwnership -TargetVolume $WorkingConfigVolume
Invoke-DockerVolumeCopy -SourceVolume $SourceBrowserVolume -TargetVolume $WorkingBrowserVolume
Set-VolumeOwnership -TargetVolume $WorkingBrowserVolume

docker run --rm `
    -v "${WorkingConfigVolume}:/dest" `
    --entrypoint sh `
    $HelperImage `
    -lc "ls -lah /dest | sed -n '1,20p'; echo '---'; stat -c '%a %u %g %n' /dest/ChineseSubFinderSettings.json; echo '---'; find /dest/cache/CSF-CacheCenter -maxdepth 3 | sed -n '1,40p'"

Write-Output "WORKING_CONFIG_VOLUME=$WorkingConfigVolume"
Write-Output "WORKING_BROWSER_VOLUME=$WorkingBrowserVolume"
