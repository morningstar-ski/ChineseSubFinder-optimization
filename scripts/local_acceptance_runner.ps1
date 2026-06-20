. (Join-Path $PSScriptRoot "local_acceptance_matrix_utils.ps1")

function Invoke-SamplePoolAudit {
    param(
        [string]$WorkspaceRoot,
        [string]$SamplePoolRoot
    )

    $json = & powershell -NoProfile -ExecutionPolicy Bypass `
        -File (Join-Path $WorkspaceRoot "scripts\local_sample_pool_audit.ps1") `
        -SamplePoolRoot $SamplePoolRoot `
        -AsJson
    if ($LASTEXITCODE -ne 0) {
        throw "Sample pool audit failed for $SamplePoolRoot"
    }
    return ($json | ConvertFrom-Json)
}

function Resolve-RunnableSamplePoolRoot {
    param(
        [string]$WorkspaceRoot,
        [string]$SamplePoolRoot
    )

    $directAudit = Invoke-SamplePoolAudit -WorkspaceRoot $WorkspaceRoot -SamplePoolRoot $SamplePoolRoot
    if ([int]$directAudit.total_specs -gt 0 -and [int]$directAudit.docker_visible_video_count -eq [int]$directAudit.total_specs) {
        Write-Host "Sample pool is directly Docker-visible: $SamplePoolRoot"
        return $SamplePoolRoot
    }

    $runtimeRoot = "D:\tmp\csf-real-media-runtime"
    $runtimeSpecRoot = Join-Path $runtimeRoot "sample-specs"
    if (Test-Path -LiteralPath $runtimeSpecRoot) {
        $runtimeAudit = Invoke-SamplePoolAudit -WorkspaceRoot $WorkspaceRoot -SamplePoolRoot $runtimeSpecRoot
        if ([int]$runtimeAudit.total_specs -gt 0 -and [int]$runtimeAudit.docker_visible_video_count -eq [int]$runtimeAudit.total_specs) {
            Write-Host "Reuse materialized runtime sample pool: $runtimeSpecRoot"
            return $runtimeSpecRoot
        }
    }

    Write-Host "Materialize runtime sample pool from: $SamplePoolRoot"
    & powershell -NoProfile -ExecutionPolicy Bypass `
        -File (Join-Path $WorkspaceRoot "scripts\local_materialize_sample_pool.ps1") `
        -SourceSamplePoolRoot $SamplePoolRoot `
        -OutputRoot $runtimeRoot
    if ($LASTEXITCODE -ne 0) {
        throw "Materialize runtime sample pool failed for $SamplePoolRoot"
    }

    $materializedAudit = Invoke-SamplePoolAudit -WorkspaceRoot $WorkspaceRoot -SamplePoolRoot $runtimeSpecRoot
    if ([int]$materializedAudit.total_specs -le 0 -or [int]$materializedAudit.docker_visible_video_count -ne [int]$materializedAudit.total_specs) {
        throw "Materialized runtime sample pool is still not fully Docker-visible: $runtimeSpecRoot"
    }

    return $runtimeSpecRoot
}

function Invoke-AcceptanceRound {
    param(
        [string]$WorkspaceRoot,
        [string]$CandidateRoot,
        [string]$CandidateImage,
        [string]$CandidateContainer,
        [int]$Port,
        [int]$StaticPort,
        [string]$SamplePoolRoot,
        [string]$ConfigDockerVolume = "",
        [string]$BrowserDockerVolume = "",
        [switch]$AllowDirtyDockerState,
        [hashtable]$Spec,
        [string]$LLMProvider = "deepseek",
        [string]$LLMBaseUrl = "",
        [string]$LLMApiKey = "",
        [string]$LLMModel = "deepseek-v4-flash",
        [int]$LLMJobTimeoutSeconds = 1800
    )

    $sampleSpecPath = Join-Path $SamplePoolRoot ([string]$Spec.sample_spec)
    Write-Host ""
    Write-Host "==== $($Spec.name) ===="

    $args = @(
        "-NoProfile",
        "-ExecutionPolicy", "Bypass",
        "-File", (Join-Path $WorkspaceRoot "scripts\local_candidate_round.ps1"),
        "-WorkspaceRoot", $WorkspaceRoot,
        "-CandidateRoot", $CandidateRoot,
        "-CandidateImage", $CandidateImage,
        "-CandidateContainer", $CandidateContainer,
        "-Port", $Port,
        "-StaticPort", $StaticPort,
        "-SampleSpecPath", $sampleSpecPath
    )

    if (-not [string]::IsNullOrWhiteSpace($ConfigDockerVolume)) {
        $args += @("-ConfigDockerVolume", $ConfigDockerVolume)
    }
    if (-not [string]::IsNullOrWhiteSpace($BrowserDockerVolume)) {
        $args += @("-BrowserDockerVolume", $BrowserDockerVolume)
    }
    if ($AllowDirtyDockerState) {
        $args += "-AllowDirtyDockerState"
    }

    if ($Spec.run_static_checks) { $args += "-RunStaticChecks" }
    if ($Spec.build_image) { $args += "-BuildImage" }
    if ($Spec.start_container) { $args += "-StartContainer" }
    if ($Spec.run_e2e_matrix) { $args += "-RunE2EMatrix" }
    if ($Spec.accept_no_sub_found) { $args += "-AcceptNoSubFound" }
    if ($Spec.enable_subtitlecat_translated_chinese_fallback) { $args += "-EnableSubtitleCatTranslatedChineseFallback" }
    if ($Spec.enable_llm_fallback) { $args += "-EnableLLMFallback" }

    if ($null -ne $Spec.primary_chinese_suppliers -and $Spec.primary_chinese_suppliers.Count -gt 0) {
        $args += @("-PrimaryChineseSuppliers", [string[]]$Spec.primary_chinese_suppliers)
    }
    if ($null -ne $Spec.english_fallback_suppliers -and $Spec.english_fallback_suppliers.Count -gt 0) {
        $args += @("-EnglishFallbackSuppliers", [string[]]$Spec.english_fallback_suppliers)
    }
    if (-not [string]::IsNullOrWhiteSpace([string]$Spec.expected_route_key)) {
        $args += @("-ExpectedRouteKey", [string]$Spec.expected_route_key)
    }
    if (-not [string]::IsNullOrWhiteSpace([string]$Spec.expected_winning_supplier)) {
        $args += @("-ExpectedWinningSupplier", [string]$Spec.expected_winning_supplier)
    }

    if ($Spec.enable_llm_fallback) {
        $args += @(
            "-LLMProvider", $LLMProvider,
            "-LLMBaseUrl", $LLMBaseUrl,
            "-LLMApiKey", $LLMApiKey,
            "-LLMModel", $LLMModel,
            "-JobTimeoutSeconds", $LLMJobTimeoutSeconds
        )
    }

    # Keep rounds strictly sequential. They mutate one shared container and
    # shared auth/session state, so parallel execution causes false harness noise.
    & powershell @args
    if ($LASTEXITCODE -ne 0) {
        throw "$($Spec.name) failed."
    }
}

function Invoke-AcceptanceProfile {
    param(
        [string]$WorkspaceRoot,
        [string]$ProfileName,
        [string]$CandidateRoot,
        [string]$CandidateImage,
        [string]$CandidateContainer,
        [int]$Port,
        [int]$StaticPort,
        [string]$SamplePoolRoot,
        [string]$ConfigDockerVolume = "",
        [string]$BrowserDockerVolume = "",
        [switch]$AllowDirtyDockerState,
        [string]$LLMProvider = "deepseek",
        [string]$LLMBaseUrl = "",
        [string]$LLMApiKey = "",
        [string]$LLMModel = "deepseek-v4-flash",
        [int]$LLMJobTimeoutSeconds = 1800
    )

    $resolvedSamplePoolRoot = Resolve-RunnableSamplePoolRoot -WorkspaceRoot $WorkspaceRoot -SamplePoolRoot $SamplePoolRoot
    $specs = @(Get-AcceptanceProfileSpecs -WorkspaceRoot $WorkspaceRoot -ProfileName $ProfileName)
    $skippedLLM = @()

    foreach ($spec in $specs) {
        if ($spec.requires_llm -and [string]::IsNullOrWhiteSpace($LLMApiKey)) {
            $skippedLLM += [string]$spec.name
            continue
        }

        Invoke-AcceptanceRound `
            -WorkspaceRoot $WorkspaceRoot `
            -CandidateRoot $CandidateRoot `
            -CandidateImage $CandidateImage `
            -CandidateContainer $CandidateContainer `
            -Port $Port `
            -StaticPort $StaticPort `
            -SamplePoolRoot $resolvedSamplePoolRoot `
            -ConfigDockerVolume $ConfigDockerVolume `
            -BrowserDockerVolume $BrowserDockerVolume `
            -AllowDirtyDockerState:$AllowDirtyDockerState `
            -Spec $spec `
            -LLMProvider $LLMProvider `
            -LLMBaseUrl $LLMBaseUrl `
            -LLMApiKey $LLMApiKey `
            -LLMModel $LLMModel `
            -LLMJobTimeoutSeconds $LLMJobTimeoutSeconds
    }

    if ($skippedLLM.Count -gt 0) {
        Write-Host ""
        Write-Host ("Skip LLM rounds because LLMApiKey is empty: " + ($skippedLLM -join ", "))
    }
}
