param(
    [string]$WorkspaceRoot = "C:\Users\yang\Desktop\csf\ChineseSubFinder-provider-pack",
    [string]$CandidateRoot = "D:\tmp\csf-local-candidate",
    [string]$CandidateImage = "chinesesubfinder:local-candidate",
    [string]$CandidateContainer = "chinesesubfinder-local-candidate",
    [int]$Port = 19235,
    [int]$StaticPort = 19237,
    [string]$SamplePoolRoot = "D:\tmp\csf-real-media-stage",
    [string]$ConfigDockerVolume = "",
    [string]$BrowserDockerVolume = "",
    [switch]$AllowDirtyDockerState,
    [string]$LLMProvider = "deepseek",
    [string]$LLMBaseUrl = "",
    [string]$LLMApiKey = "",
    [string]$LLMModel = "deepseek-v4-flash",
    [int]$LLMJobTimeoutSeconds = 1800
)

$ErrorActionPreference = "Stop"
. (Join-Path $WorkspaceRoot "scripts\local_acceptance_runner.ps1")

Invoke-AcceptanceProfile `
    -WorkspaceRoot $WorkspaceRoot `
    -ProfileName "full" `
    -CandidateRoot $CandidateRoot `
    -CandidateImage $CandidateImage `
    -CandidateContainer $CandidateContainer `
    -Port $Port `
    -StaticPort $StaticPort `
    -SamplePoolRoot $SamplePoolRoot `
    -ConfigDockerVolume $ConfigDockerVolume `
    -BrowserDockerVolume $BrowserDockerVolume `
    -AllowDirtyDockerState:$AllowDirtyDockerState `
    -LLMProvider $LLMProvider `
    -LLMBaseUrl $LLMBaseUrl `
    -LLMApiKey $LLMApiKey `
    -LLMModel $LLMModel `
    -LLMJobTimeoutSeconds $LLMJobTimeoutSeconds

Write-Host ""
Write-Host "Local full acceptance completed."
