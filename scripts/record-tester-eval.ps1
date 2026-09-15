[CmdletBinding()]
param(
    [Parameter(Mandatory)]
    [ValidatePattern("^[A-Za-z0-9][A-Za-z0-9 _-]*$")]
    [string]$Scenario,

    [Parameter(Mandatory)]
    [ValidatePattern("^[A-Za-z0-9][A-Za-z0-9 _.-]*$")]
    [string]$Configuration,

    [Parameter(Mandatory)]
    [ValidateSet("Pass", "Fail", "Blocked")]
    [string]$QualityOutcome,

    [Parameter(Mandatory)]
    [ValidateNotNullOrEmpty()]
    [string]$QualityEvidence,

    [Parameter(Mandatory)]
    [ValidateNotNullOrEmpty()]
    [string]$ImplementationSummary,

    [Parameter(Mandatory)]
    [ValidateSet("Pass", "Fail", "Blocked")]
    [string]$AcceptanceVerdict,

    [Parameter(Mandatory)]
    [ValidatePattern("^[a-fA-F0-9]{64}$")]
    [string]$TestStateFingerprint,

    [Parameter(Mandatory)]
    [ValidateNotNullOrEmpty()]
    [string]$SurfaceProfile,

    [Parameter(Mandatory)]
    [ValidateNotNullOrEmpty()]
    [string[]]$AcceptanceResults,

    [Parameter(Mandatory)]
    [ValidateNotNullOrEmpty()]
    [string[]]$EvidenceSignatures,

    [string[]]$FindingSignatures = @(),
    [string]$BaselineReport,
    [string]$PhaseMetricsPath,
    [int]$Turns = -1,
    [int]$ToolCalls = -1,
    [double]$InputTokens = -1,
    [double]$CachedInputTokens = -1,
    [double]$OutputTokens = -1,
    [double]$ElapsedSeconds = -1,
    [double]$EstimatedCostUsd = -1
)

$ErrorActionPreference = "Stop"
$baseRecorder = Join-Path $PSScriptRoot "record-agent-eval.ps1"

function Get-Sha256Text([string]$Value) {
    $bytes = [System.Text.Encoding]::UTF8.GetBytes($Value)
    $hash = [System.Security.Cryptography.SHA256]::HashData($bytes)
    return [System.Convert]::ToHexString($hash).ToLowerInvariant()
}

function Get-NormalizedArray([string[]]$Values) {
    return @($Values | ForEach-Object { $_.Trim() } | Where-Object { $_ -ne "" } | Sort-Object -Unique)
}

$normalizedAcceptance = Get-NormalizedArray $AcceptanceResults
$normalizedEvidence = Get-NormalizedArray $EvidenceSignatures
$normalizedFindings = Get-NormalizedArray $FindingSignatures

if ($normalizedAcceptance.Count -eq 0) {
    throw "At least one acceptance result is required."
}
if ($normalizedEvidence.Count -eq 0) {
    throw "At least one evidence signature is required."
}

$acceptanceCounts = [ordered]@{
    total = 0
    pass = 0
    fail = 0
    blocked = 0
    not_run = 0
}
foreach ($row in $normalizedAcceptance) {
    if ($row -notmatch '^(AC\d+)=(Pass|Fail|Blocked|Not run)$') {
        throw "Acceptance result '$row' must use AC##=Pass|Fail|Blocked|Not run."
    }
    $acceptanceCounts.total++
    switch ($Matches[2]) {
        "Pass" { $acceptanceCounts.pass++ }
        "Fail" { $acceptanceCounts.fail++ }
        "Blocked" { $acceptanceCounts.blocked++ }
        "Not run" { $acceptanceCounts.not_run++ }
    }
}

switch ($AcceptanceVerdict) {
    "Pass" {
        if ($acceptanceCounts.fail -ne 0 -or $acceptanceCounts.blocked -ne 0 -or $acceptanceCounts.not_run -ne 0 -or $normalizedFindings.Count -ne 0) {
            throw "AcceptanceVerdict Pass requires every AC row to Pass and no P0-P3 finding signatures."
        }
    }
    "Fail" {
        if ($acceptanceCounts.fail -eq 0 -or $normalizedFindings.Count -eq 0) {
            throw "AcceptanceVerdict Fail requires at least one failed AC row and one finding signature."
        }
    }
    "Blocked" {
        if ($acceptanceCounts.blocked -eq 0 -and $acceptanceCounts.not_run -eq 0) {
            throw "AcceptanceVerdict Blocked requires at least one Blocked or Not run AC row."
        }
    }
}

$recordArgs = @{
    Scenario = $Scenario
    Configuration = $Configuration
    QualityOutcome = $QualityOutcome
    QualityEvidence = $QualityEvidence
    ImplementationSummary = $ImplementationSummary
    ReviewVerdict = "Not applicable"
    TestVerdict = $AcceptanceVerdict
}
if ($BaselineReport) { $recordArgs.BaselineReport = $BaselineReport }
if ($PhaseMetricsPath) { $recordArgs.PhaseMetricsPath = $PhaseMetricsPath }
if ($Turns -ge 0) { $recordArgs.Turns = $Turns }
if ($ToolCalls -ge 0) { $recordArgs.ToolCalls = $ToolCalls }
if ($InputTokens -ge 0) { $recordArgs.InputTokens = $InputTokens }
if ($CachedInputTokens -ge 0) { $recordArgs.CachedInputTokens = $CachedInputTokens }
if ($OutputTokens -ge 0) { $recordArgs.OutputTokens = $OutputTokens }
if ($ElapsedSeconds -ge 0) { $recordArgs.ElapsedSeconds = $ElapsedSeconds }
if ($EstimatedCostUsd -ge 0) { $recordArgs.EstimatedCostUsd = $EstimatedCostUsd }

$reportOutput = & $baseRecorder @recordArgs
$reportPath = [string]($reportOutput | Select-Object -Last 1)
$resolvedReport = (Resolve-Path -LiteralPath $reportPath).Path
$manifestPath = [System.IO.Path]::ChangeExtension($resolvedReport, ".json")
$manifest = Get-Content -Raw -LiteralPath $manifestPath | ConvertFrom-Json

$testerObservation = [ordered]@{
    acceptance_verdict = $AcceptanceVerdict
    test_state_fingerprint = $TestStateFingerprint.ToLowerInvariant()
    surface_profile = $SurfaceProfile.Trim()
    acceptance_results = $normalizedAcceptance
    acceptance_results_sha256 = Get-Sha256Text ($normalizedAcceptance -join "`n")
    acceptance_counts = $acceptanceCounts
    finding_signatures = $normalizedFindings
    finding_signatures_sha256 = Get-Sha256Text ($normalizedFindings -join "`n")
    finding_count = $normalizedFindings.Count
    evidence_signatures = $normalizedEvidence
    evidence_signatures_sha256 = Get-Sha256Text ($normalizedEvidence -join "`n")
    evidence_count = $normalizedEvidence.Count
}

$manifest | Add-Member -NotePropertyName agent_role -NotePropertyValue "tester" -Force
$manifest | Add-Member -NotePropertyName tester_observation -NotePropertyValue $testerObservation -Force
$manifest.schema_version = 3
$manifest | ConvertTo-Json -Depth 10 | Set-Content -LiteralPath $manifestPath -Encoding utf8

$stateText = $TestStateFingerprint.ToLowerInvariant()
$append = @(
    "",
    "## Tester evaluation observation",
    "",
    "| Field | Value |",
    "| --- | --- |",
    "| Acceptance verdict | $AcceptanceVerdict |",
    "| Tested state | ``$stateText`` |",
    "| Surface profile | $($SurfaceProfile.Trim()) |",
    "| Acceptance rows | $($acceptanceCounts.total) |",
    "| Failed rows | $($acceptanceCounts.fail) |",
    "| Blocked rows | $($acceptanceCounts.blocked) |",
    "| Not-run rows | $($acceptanceCounts.not_run) |",
    "| Finding signatures | $($normalizedFindings.Count) |",
    "| Evidence signatures | $($normalizedEvidence.Count) |",
    "",
    "Tester high-vs-medium adoption is decided by `scripts/compare-tester-evals.ps1`; the generic per-run decision above is not the tester adoption gate."
)
Add-Content -LiteralPath $resolvedReport -Value $append -Encoding utf8

Write-Output $resolvedReport
