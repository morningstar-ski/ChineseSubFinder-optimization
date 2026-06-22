param(
    [string]$BaseUrl = "http://127.0.0.1:19035",
    [string]$Username = "ymx409599725",
    [string]$Password = "Ymxxmysc123",
    [string]$OutPath = "D:\tmp\csf-sample-check-results.json",
    [string]$LibraryRoot = "\\192.168.100.4\video\link",
    [string]$SampleListPath = ""
)

$ErrorActionPreference = "Stop"

function Invoke-CsfJson {
    param(
        [string]$Method,
        [string]$Url,
        [hashtable]$Headers,
        $Body = $null
    )

    if ($null -ne $Body) {
        $json = $Body | ConvertTo-Json -Depth 8 -Compress
        $bytes = [System.Text.Encoding]::UTF8.GetBytes($json)
        $resp = Invoke-WebRequest -UseBasicParsing -Method $Method -Uri $Url -Headers $Headers -ContentType "application/json; charset=utf-8" -Body $bytes
    } else {
        $resp = Invoke-WebRequest -UseBasicParsing -Method $Method -Uri $Url -Headers $Headers
    }

    if ([string]::IsNullOrWhiteSpace($resp.Content)) {
        return $null
    }
    return $resp.Content | ConvertFrom-Json
}

function Get-VideoSubtitleCount {
    param([string]$VideoPath)

    $dirPath = Split-Path -Path $VideoPath -Parent
    $baseName = [System.IO.Path]::GetFileNameWithoutExtension($VideoPath)
    $subs = @(
        Get-ChildItem -LiteralPath $dirPath -File |
            Where-Object {
                $_.BaseName -like "$baseName*" -and
                $_.Extension -in '.srt', '.ass', '.ssa', '.sub'
            }
    )
    return $subs.Count
}

$login = Invoke-CsfJson -Method "POST" -Url ($BaseUrl + "/login") -Headers @{} -Body @{
    username = $Username
    password = $Password
}

$token = [string]$login.access_token
if ([string]::IsNullOrWhiteSpace($token)) {
    throw "login failed: no access token"
}

$headers = @{
    Authorization = "Bearer $token"
}

$samples = New-Object System.Collections.Generic.List[object]

if (-not [string]::IsNullOrWhiteSpace($SampleListPath)) {
    $rawSamples = Get-Content -LiteralPath $SampleListPath -Raw -Encoding UTF8 | ConvertFrom-Json
    foreach ($sample in $rawSamples) {
        $samples.Add([pscustomobject]@{
            label = [string]$sample.label
            video_type = [int]$sample.video_type
            video_path = [string]$sample.video_path
        }) | Out-Null
    }
} else {
    $movieVideos = Get-ChildItem -LiteralPath $LibraryRoot -Recurse -File |
        Where-Object {
            $_.Extension -in '.mkv', '.mp4', '.avi' -and
            $_.FullName -notlike "*.csf-bk" -and
            (Split-Path $_.FullName -Parent) -like "*\电影\*"
        } |
        ForEach-Object {
            [pscustomobject]@{
                video_path = $_.FullName
                subtitle_count = Get-VideoSubtitleCount -VideoPath $_.FullName
            }
        } |
        Where-Object { $_.subtitle_count -eq 0 } |
        Select-Object -First 3

    $seriesVideos = Get-ChildItem -LiteralPath $LibraryRoot -Recurse -File |
        Where-Object {
            $_.Extension -in '.mkv', '.mp4', '.avi' -and
            $_.FullName -notlike "*.csf-bk" -and
            (Split-Path $_.FullName -Parent) -like "*\电视剧\*"
        } |
        ForEach-Object {
            [pscustomobject]@{
                video_path = $_.FullName
                subtitle_count = Get-VideoSubtitleCount -VideoPath $_.FullName
            }
        } |
        Where-Object { $_.subtitle_count -eq 0 } |
        Select-Object -First 1

    $index = 1
    foreach ($movie in $movieVideos) {
        $samples.Add([pscustomobject]@{
            label = "movie_$index"
            video_type = 0
            video_path = $movie.video_path
        }) | Out-Null
        $index++
    }
    foreach ($series in $seriesVideos) {
        $samples.Add([pscustomobject]@{
            label = "series_1"
            video_type = 1
            video_path = $series.video_path
        }) | Out-Null
    }
}

if ([string]::IsNullOrWhiteSpace($SampleListPath) -and $samples.Count -lt 4) {
    throw "not enough samples discovered under $LibraryRoot"
}

$results = New-Object System.Collections.Generic.List[object]

foreach ($sample in $samples) {
    $videoPath = $sample.video_path
    $dirPath = Split-Path -Path $videoPath -Parent
    $baseName = [System.IO.Path]::GetFileNameWithoutExtension($videoPath)
    $beforeSubs = @(
        Get-ChildItem -LiteralPath $dirPath -File |
            Where-Object { $_.BaseName -like "$baseName*" -and $_.Extension -in '.srt', '.ass', '.ssa', '.sub' } |
            Select-Object -ExpandProperty FullName
    )

    $resp = Invoke-CsfJson -Method "POST" -Url ($BaseUrl + "/v1/video/list/add") -Headers $headers -Body @{
        video_type = $sample.video_type
        physical_video_file_full_path = $videoPath
        task_priority_level = 0
        media_server_inside_video_id = ""
    }

    $jobId = [string]$resp.job_id
    if ([string]::IsNullOrWhiteSpace($jobId)) {
        throw "add job failed for $($sample.label): no job_id"
    }

    $job = $null
    $jobLog = $null
    $timedOut = $false
    $deadline = (Get-Date).AddMinutes(8)
    do {
        Start-Sleep -Seconds 5
        $jobList = Invoke-CsfJson -Method "GET" -Url ($BaseUrl + "/v1/jobs/list") -Headers $headers
        $job = @($jobList.all_jobs) | Where-Object { $_.id -eq $jobId } | Select-Object -First 1
        if ($null -eq $job) {
            continue
        }
        if ([int]$job.job_status -in 2, 3, 5) {
            $jobLog = Invoke-CsfJson -Method "POST" -Url ($BaseUrl + "/v1/jobs/log") -Headers $headers -Body @{ id = $jobId }
            break
        }
    } while ((Get-Date) -lt $deadline)

    if ($null -eq $job -or [int]$job.job_status -notin 2, 3, 5) {
        $timedOut = $true
        if ($null -ne $job) {
            $jobLog = Invoke-CsfJson -Method "POST" -Url ($BaseUrl + "/v1/jobs/log") -Headers $headers -Body @{ id = $jobId }
        }
    }

    $afterSubs = @(
        Get-ChildItem -LiteralPath $dirPath -File |
            Where-Object { $_.BaseName -like "$baseName*" -and $_.Extension -in '.srt', '.ass', '.ssa', '.sub' } |
            Select-Object -ExpandProperty FullName
    )
    $newSubs = @($afterSubs | Where-Object { $_ -notin $beforeSubs })

    $results.Add([pscustomobject]@{
        label = $sample.label
        video_type = $sample.video_type
        video_path = $videoPath
        job_id = $jobId
        timed_out = $timedOut
        job = $job
        new_subtitles = $newSubs
        all_subtitles_after = $afterSubs
        job_log = $jobLog
    }) | Out-Null
}

$results | ConvertTo-Json -Depth 8 | Set-Content -LiteralPath $OutPath -Encoding UTF8
Write-Output $OutPath
