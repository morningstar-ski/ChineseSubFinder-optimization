param(
    [string]$WorkspaceRoot = "C:\Users\yang\Desktop\csf\ChineseSubFinder-provider-pack",
    [string]$ReportRoot = "D:\tmp\csf-local-candidate\reports",
    [string[]]$RouteKeys = @(),
    [switch]$AsJson
)

$ErrorActionPreference = "Stop"
. (Join-Path $WorkspaceRoot "scripts\local_acceptance_matrix_utils.ps1")

function Read-JsonFileUtf8([string]$Path) {
    return Get-Content -LiteralPath $Path -Raw -Encoding UTF8 | ConvertFrom-Json
}

function Normalize-RouteKeys {
    param(
        [string[]]$Values
    )

    $normalized = New-Object System.Collections.Generic.List[string]
    foreach ($value in @($Values)) {
        if ([string]::IsNullOrWhiteSpace($value)) {
            continue
        }
        foreach ($part in ($value -split ',')) {
            $trimmed = $part.Trim()
            if (-not [string]::IsNullOrWhiteSpace($trimmed)) {
                $normalized.Add($trimmed) | Out-Null
            }
        }
    }

    return @($normalized | Select-Object -Unique)
}

function Get-E2EDirs {
    if (-not (Test-Path -LiteralPath $ReportRoot)) {
        return @()
    }
    return @(Get-ChildItem -LiteralPath $ReportRoot -Directory -Filter "*-e2e-matrix" -ErrorAction SilentlyContinue |
        Sort-Object LastWriteTime -Descending)
}

function Get-LatestSummariesByRoute {
    $wantedRoutes = @(Normalize-RouteKeys -Values $RouteKeys)
    $dirs = @(Get-E2EDirs)
    $picked = @{}

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

        $routeKey = [string]$summary.route_key
        if ([string]::IsNullOrWhiteSpace($routeKey)) {
            continue
        }
        if ($wantedRoutes.Count -gt 0 -and $routeKey -notin $wantedRoutes) {
            continue
        }
        if ($picked.ContainsKey($routeKey)) {
            continue
        }

        $picked[$routeKey] = [pscustomobject]@{
            route_key = $routeKey
            round_id = [string]$summary.round_id
            round_root = [string]$summary.round_root
            summary_path = $summaryPath
            summary = $summary
        }
    }

    return @($picked.Values | Sort-Object route_key)
}

function Get-CopiedSubtitlePaths([string]$RoundRoot) {
    $artifactPath = Join-Path $RoundRoot "copied-subtitle-artifacts.json"
    if (-not (Test-Path -LiteralPath $artifactPath)) {
        return @()
    }

    try {
        $artifacts = @(Read-JsonFileUtf8 $artifactPath)
    } catch {
        return @()
    }

    return @($artifacts | Where-Object { -not [string]::IsNullOrWhiteSpace([string]$_) })
}

function Test-HasChineseCharacters([string]$Text) {
    return [regex]::IsMatch($Text, '[\u4e00-\u9fff]')
}

function Test-HasAsciiLetters([string]$Text) {
    return [regex]::IsMatch($Text, '[A-Za-z]')
}

function Get-DialogueLines([string]$Path) {
    $lines = Get-Content -LiteralPath $Path -Encoding UTF8
    $dialogueLines = New-Object System.Collections.Generic.List[string]

    foreach ($rawLine in $lines) {
        $line = [string]$rawLine
        $trim = $line.Trim()
        if ($trim -eq "") { continue }
        if ($trim -match '^;') { continue }
        if ($trim -match '^[0-9]+$') { continue }
        if ($trim -match '^\d{2}:\d{2}:\d{2}') { continue }
        if ($trim -match '^\d{1,2}:\d{2}:\d{2}[,\.]\d{2,3}\s*-->\s*\d{1,2}:\d{2}:\d{2}[,\.]\d{2,3}$') { continue }
        if ($trim -match '^\[Script Info\]') { continue }
        if ($trim -match '^\[V4\+ Styles\]') { continue }
        if ($trim -match '^\[Events\]') { continue }
        if ($trim -match '^\[Aegisub Project Garbage\]') { continue }
        if ($trim -match '^Title:') { continue }
        if ($trim -match '^ScriptType:') { continue }
        if ($trim -match '^WrapStyle:') { continue }
        if ($trim -match '^ScaledBorderAndShadow:') { continue }
        if ($trim -match '^YCbCr') { continue }
        if ($trim -match '^PlayRes') { continue }
        if ($trim -match '^Format:') { continue }
        if ($trim -match '^Style:') { continue }

        if ($trim -like 'Dialogue:*') {
            $firstComma = $trim.IndexOf(',')
            if ($firstComma -ge 0) {
                $parts = $trim.Split(',', 10)
                if ($parts.Count -ge 10) {
                    $trim = $parts[9].Trim()
                }
            }
        }

        $trim = $trim -replace '\\N', ' / '
        $trim = $trim -replace '\{[^}]+\}', ''
        $trim = $trim.Trim()
        if ($trim -eq "") { continue }

        $dialogueLines.Add($trim) | Out-Null
    }

    return @($dialogueLines)
}

function Get-SubtitleContentStats([string]$Path) {
    $dialogueLines = @(Get-DialogueLines -Path $Path)
    $chineseLineCount = 0
    $englishLineCount = 0
    $englishOnlyLineCount = 0
    $mixedLanguageLineCount = 0
    $bilingualPresentationLineCount = 0
    $englishOnlySamples = New-Object System.Collections.Generic.List[string]
    $mixedLanguageSamples = New-Object System.Collections.Generic.List[string]
    $bilingualPresentationSamples = New-Object System.Collections.Generic.List[string]

    foreach ($line in $dialogueLines) {
        $hasChinese = Test-HasChineseCharacters $line
        $hasEnglish = Test-HasAsciiLetters $line
        if ($hasChinese) {
            $chineseLineCount++
        }
        if ($hasEnglish) {
            $englishLineCount++
        }
        if ($hasEnglish -and -not $hasChinese) {
            $englishOnlyLineCount++
            if ($englishOnlySamples.Count -lt 8) {
                $englishOnlySamples.Add($line) | Out-Null
            }
        }
        if ($hasEnglish -and $hasChinese) {
            $mixedLanguageLineCount++
            if ($mixedLanguageSamples.Count -lt 8) {
                $mixedLanguageSamples.Add($line) | Out-Null
            }
            if ($line -match '\s/\s' -or $line -match '^\s*[^/]+/\s*[^/]+$') {
                $bilingualPresentationLineCount++
                if ($bilingualPresentationSamples.Count -lt 8) {
                    $bilingualPresentationSamples.Add($line) | Out-Null
                }
            }
        }
    }

    $looksBilingualPresentation = $false
    if ($dialogueLines.Count -gt 0) {
        $bilingualRatio = $bilingualPresentationLineCount / $dialogueLines.Count
        if ($bilingualRatio -ge 0.25 -or [System.IO.Path]::GetFileName($Path) -match '简英|chs-eng|chi-eng|bilingual') {
            $looksBilingualPresentation = $true
        }
    }

    return [ordered]@{
        file_path = $Path
        file_name = [System.IO.Path]::GetFileName($Path)
        extension = [System.IO.Path]::GetExtension($Path)
        file_length = (Get-Item -LiteralPath $Path).Length
        dialogue_line_count = $dialogueLines.Count
        chinese_line_count = $chineseLineCount
        english_line_count = $englishLineCount
        english_only_line_count = $englishOnlyLineCount
        mixed_language_line_count = $mixedLanguageLineCount
        bilingual_presentation_line_count = $bilingualPresentationLineCount
        english_line_ratio = if ($dialogueLines.Count -gt 0) { [math]::Round(($englishLineCount / $dialogueLines.Count), 4) } else { 0 }
        english_only_line_ratio = if ($dialogueLines.Count -gt 0) { [math]::Round(($englishOnlyLineCount / $dialogueLines.Count), 4) } else { 0 }
        bilingual_presentation_line_ratio = if ($dialogueLines.Count -gt 0) { [math]::Round(($bilingualPresentationLineCount / $dialogueLines.Count), 4) } else { 0 }
        has_chinese = ($chineseLineCount -gt 0)
        has_english = ($englishLineCount -gt 0)
        looks_bilingual_presentation = $looksBilingualPresentation
        sample_lines = @($dialogueLines | Select-Object -First 8)
        english_only_samples = @($englishOnlySamples)
        mixed_language_samples = @($mixedLanguageSamples)
        bilingual_presentation_samples = @($bilingualPresentationSamples)
    }
}

function Get-RouteStageDisplay([string]$RouteKey, [string]$RouteStage) {
    if (-not [string]::IsNullOrWhiteSpace($RouteStage)) {
        return $RouteStage
    }
    if ($RouteKey -like "*.llm_fallback") {
        return "llm_fallback"
    }
    if ($RouteKey -like "*.subtitlecat_translated") {
        return "translated_chinese"
    }
    return ""
}

function Get-RouteWarnings($Row) {
    $warnings = New-Object System.Collections.Generic.List[string]
    $routeKey = [string]$Row.route_key
    $subtitleFiles = @($Row.subtitle_files)

    if ($routeKey -like "*.llm_fallback") {
        foreach ($file in $subtitleFiles) {
            if ($file.looks_bilingual_presentation) {
                continue
            }
            if ([double]$file.english_only_line_ratio -ge 0.02 -or [int]$file.english_only_line_count -ge 30) {
                $warnings.Add(("LLM fallback output still contains noticeable untranslated English: {0} english-only lines across {1} dialogue lines." -f $file.english_only_line_count, $file.dialogue_line_count)) | Out-Null
            }
        }
    }

    if ($routeKey -like "*.subtitlecat_translated") {
        foreach ($file in $subtitleFiles) {
            if ($file.looks_bilingual_presentation) {
                continue
            }
            if ([double]$file.english_only_line_ratio -ge 0.01 -or [int]$file.english_only_line_count -ge 12) {
                $warnings.Add(("Translated-Chinese fallback output still contains residual untranslated English: {0} english-only lines across {1} dialogue lines." -f $file.english_only_line_count, $file.dialogue_line_count)) | Out-Null
            }
        }
    }

    return @($warnings)
}

$wanted = @(Normalize-RouteKeys -Values $RouteKeys)
if ($wanted.Count -eq 0) {
    $wanted = @(
        "movie.native_chinese",
        "movie.subtitlecat_translated",
        "movie.english_fallback",
        "movie.llm_fallback",
        "series.native_chinese",
        "series.english_fallback",
        "series.subtitlecat_translated",
        "series.llm_fallback"
    )
}

$summaries = @(Get-LatestSummariesByRoute | Where-Object { $_.route_key -in $wanted })
$rows = @()

foreach ($entry in $summaries) {
    $subtitlePaths = @(Get-CopiedSubtitlePaths -RoundRoot $entry.round_root)
    $fileStats = @()
    foreach ($path in $subtitlePaths) {
        if (-not (Test-Path -LiteralPath $path)) {
            continue
        }
        $fileStats += [pscustomobject](Get-SubtitleContentStats -Path $path)
    }

    $rows += [pscustomobject]@{
        route_key = $entry.route_key
        round_id = $entry.round_id
        round_root = $entry.round_root
        sample_base_name = [string]$entry.summary.sample_base_name
        actual_supplier = [string]$entry.summary.actual_supplier
        final_output_has_chinese = [bool]$entry.summary.final_output_has_chinese
        route_stage = Get-RouteStageDisplay -RouteKey ([string]$entry.route_key) -RouteStage ([string]$entry.summary.route_stage)
        subtitle_files = $fileStats
    }
}

foreach ($row in $rows) {
    Add-Member -InputObject $row -NotePropertyName warnings -NotePropertyValue (Get-RouteWarnings -Row $row)
}

$report = [ordered]@{
    generated_at = (Get-Date).ToString("s")
    report_root = $ReportRoot
    wanted_routes = $wanted
    audited_route_count = $rows.Count
    rows = $rows
}

if ($AsJson) {
    $report | ConvertTo-Json -Depth 10
    exit 0
}

$stamp = Get-Date -Format "yyyyMMdd-HHmmss-fff"
$reportPath = Join-Path $ReportRoot "subtitle-content-audit-$stamp.json"
Write-JsonUtf8File -Value $report -Path $reportPath -Depth 10

Write-Host "Subtitle content audit: $reportPath"
Write-Host ""
foreach ($row in $rows) {
    Write-Host ("[{0}] supplier={1} stage={2}" -f $row.route_key, $row.actual_supplier, $row.route_stage)
    foreach ($file in @($row.subtitle_files)) {
        Write-Host (" - {0} lines={1} zh={2} en={3}" -f $file.file_name, $file.dialogue_line_count, $file.chinese_line_count, $file.english_line_count)
    }
}
