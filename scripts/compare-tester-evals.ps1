[CmdletBinding()]
param(
    [Parameter(Mandatory)]
    [string[]]$CandidateReports,

    [string]$BaselineConfiguration = "tester high",
    [string]$CandidateConfiguration = "tester medium",

    [ValidateRange(0.1, 100.0)]
    [double]$MinimumImprovementPercent = 10.0,

    [switch]$RequireConcurrencyFinancialAcceptance
)

$ErrorActionPreference = "Stop"
$root = Split-Path -Parent $PSScriptRoot
$reportRoot = Join-Path $root ".codex\agent-evals"
$efficiencyMetrics = @("input_tokens", "output_tokens", "tool_calls", "elapsed_seconds", "estimated_cost_usd")

function Get-ReportCell([object]$Value) {
    if ($null -eq $Value -or $Value -eq "") {
        return "Not recorded"
    }
    return ([string]$Value).Replace("|", "\\|").Replace("`r", " ").Replace("`n", " ")
}

function Get-ObjectProperty([object]$Object, [string]$Name) {
    if ($null -eq $Object) { return $null }
    $property = $Object.PSObject.Properties[$Name]
    if ($null -eq $property) { return $null }
    return $property.Value
}

function Get-ManifestForReport([string]$ReportPath) {
    $resolvedReport = (Resolve-Path -LiteralPath $ReportPath).Path
    $manifestPath = [System.IO.Path]::ChangeExtension($resolvedReport, ".json")
    if (-not (Test-Path -LiteralPath $manifestPath)) {
        throw "Manifest was not found beside '$ReportPath'."
    }
    return [ordered]@{
        report = $resolvedReport
        manifest_path = $manifestPath
        manifest = (Get-Content -Raw -LiteralPath $manifestPath | ConvertFrom-Json)
    }
}

function Get-Improvement([double]$Baseline, [double]$Candidate) {
    if ($Baseline -le 0) { return $null }
    return [math]::Round((($Baseline - $Candidate) / $Baseline) * 100.0, 2)
}

$results = @()
$scenarioIndex = @{}
$hardReject = $false
$inconclusive = $false
$hasDefectDetectionScenario = $false

foreach ($candidateReport in $CandidateReports) {
    $candidateRecord = Get-ManifestForReport $candidateReport
    $candidate = $candidateRecord.manifest

    if ($candidate.configuration -ne $CandidateConfiguration) {
        throw "Candidate report '$candidateReport' uses configuration '$($candidate.configuration)', expected '$CandidateConfiguration'."
    }
    if ($candidate.agent_role -ne "tester" -or $null -eq $candidate.tester_observation) {
        throw "Candidate report '$candidateReport' is not a tester evaluation report."
    }
    if (-not $candidate.baseline_report) {
        throw "Candidate report '$candidateReport' does not link a baseline report."
    }

    $baselineRecord = Get-ManifestForReport ([string]$candidate.baseline_report)
    $baseline = $baselineRecord.manifest
    if ($baseline.configuration -ne $BaselineConfiguration) {
        throw "Baseline for '$candidateReport' uses configuration '$($baseline.configuration)', expected '$BaselineConfiguration'."
    }
    if ($baseline.agent_role -ne "tester" -or $null -eq $baseline.tester_observation) {
        throw "Baseline for '$candidateReport' is not a tester evaluation report."
    }
    if ($baseline.scenario -ne $candidate.scenario) {
        throw "Scenario mismatch: baseline '$($baseline.scenario)' vs candidate '$($candidate.scenario)'."
    }

    $scenario = [string]$candidate.scenario
    if ($scenarioIndex.ContainsKey($scenario)) {
        throw "More than one candidate report was supplied for scenario '$scenario'."
    }
    $scenarioIndex[$scenario] = $true

    $b = $baseline.tester_observation
    $c = $candidate.tester_observation
    if ($b.acceptance_verdict -ne "Pass" -or [int]$b.finding_count -gt 0) {
        $hasDefectDetectionScenario = $true
    }

    $reasons = @()
    $verdict = "Pass"

    if ($baseline.quality.outcome -ne "Pass" -or $candidate.quality.outcome -ne "Pass") {
        $verdict = "Keep high"
        $reasons += "baseline/candidate evaluation quality outcome is not Pass"
        $hardReject = $true
    }

    foreach ($field in @("test_state_fingerprint", "surface_profile", "acceptance_verdict", "acceptance_results_sha256", "finding_signatures_sha256", "evidence_signatures_sha256")) {
        $baselineValue = Get-ObjectProperty $b $field
        $candidateValue = Get-ObjectProperty $c $field
        if ($null -eq $baselineValue -or $null -eq $candidateValue) {
            if ($verdict -ne "Keep high") { $verdict = "Inconclusive" }
            $reasons += "$field is not recorded for both runs"
            $inconclusive = $true
            continue
        }
        if ([string]$baselineValue -ne [string]$candidateValue) {
            $verdict = "Keep high"
            $reasons += "$field differs between high and medium"
            $hardReject = $true
        }
    }

    foreach ($countField in @("blocked", "not_run")) {
        $baselineCount = Get-ObjectProperty $b.acceptance_counts $countField
        $candidateCount = Get-ObjectProperty $c.acceptance_counts $countField
        if ($null -eq $baselineCount -or $null -eq $candidateCount) {
            if ($verdict -ne "Keep high") { $verdict = "Inconclusive" }
            $reasons += "acceptance $countField count is not recorded for both runs"
            $inconclusive = $true
            continue
        }
        if ([int]$baselineCount -ne 0 -or [int]$candidateCount -ne 0) {
            if ($verdict -ne "Keep high") { $verdict = "Inconclusive" }
            $reasons += "required scenario has blocked/not-run acceptance evidence"
            $inconclusive = $true
        }
    }

    $bestMetric = $null
    $bestImprovement = $null
    foreach ($metric in $efficiencyMetrics) {
        $baselineValue = Get-ObjectProperty $baseline.metrics $metric
        $candidateValue = Get-ObjectProperty $candidate.metrics $metric
        if ($null -eq $baselineValue -or $null -eq $candidateValue) { continue }
        $improvement = Get-Improvement ([double]$baselineValue) ([double]$candidateValue)
        if ($null -ne $improvement -and ($null -eq $bestImprovement -or $improvement -gt $bestImprovement)) {
            $bestMetric = $metric
            $bestImprovement = $improvement
        }
    }

    if ($verdict -ne "Keep high") {
        if ($null -eq $bestImprovement) {
            $verdict = "Inconclusive"
            $reasons += "no comparable efficiency metric recorded"
            $inconclusive = $true
        }
        elseif ($bestImprovement -lt $MinimumImprovementPercent) {
            $verdict = "Keep high"
            $reasons += "best measured efficiency gain is $bestImprovement% on $bestMetric, below $MinimumImprovementPercent%"
            $hardReject = $true
        }
    }

    if ($reasons.Count -eq 0) {
        $reasons += "same tested state, surface profile, acceptance results, findings and evidence coverage; best efficiency gain $bestImprovement% on $bestMetric"
    }

    $results += [ordered]@{
        scenario = $scenario
        verdict = $verdict
        reason = ($reasons -join "; ")
        acceptance_verdict = $c.acceptance_verdict
        finding_count = $c.finding_count
        best_metric = $bestMetric
        best_improvement_percent = $bestImprovement
        baseline_report = $baselineRecord.report
        candidate_report = $candidateRecord.report
    }
}

$required = @(
    "Routine acceptance",
    "Schema and migration acceptance",
    "Authorization/privacy acceptance",
    "User-visible acceptance"
)
if ($RequireConcurrencyFinancialAcceptance) {
    $required += "Concurrency/financial acceptance"
}

$missing = @()
foreach ($scenario in $required) {
    if (-not $scenarioIndex.ContainsKey($scenario)) {
        $missing += $scenario
    }
}
if ($missing.Count -gt 0) {
    $inconclusive = $true
}
if (-not $hasDefectDetectionScenario) {
    $inconclusive = $true
}

$overall = if ($hardReject) {
    "Keep high"
}
elseif ($inconclusive) {
    "Inconclusive"
}
else {
    "Adopt medium"
}

$runId = "{0}-{1}" -f (Get-Date).ToUniversalTime().ToString("yyyyMMddTHHmmssZ"), ([guid]::NewGuid().ToString("N").Substring(0, 8))
New-Item -ItemType Directory -Path $reportRoot -Force | Out-Null
$baseName = "tester-eval-gate-$runId"
$reportPath = Join-Path $reportRoot "$baseName.md"
$manifestPath = Join-Path $reportRoot "$baseName.json"

$manifest = [ordered]@{
    schema_version = 1
    run_id = $runId
    created_at_utc = (Get-Date).ToUniversalTime().ToString("o")
    agent_role = "tester"
    baseline_configuration = $BaselineConfiguration
    candidate_configuration = $CandidateConfiguration
    minimum_improvement_percent = $MinimumImprovementPercent
    require_concurrency_financial_acceptance = [bool]$RequireConcurrencyFinancialAcceptance
    required_scenarios_missing = $missing
    defect_detection_scenario_present = $hasDefectDetectionScenario
    result = $overall
    scenario_results = $results
}
$manifest | ConvertTo-Json -Depth 10 | Set-Content -LiteralPath $manifestPath -Encoding utf8

$lines = @(
    "# Tester Reasoning-Effort Gate — $runId",
    "",
    "**Result: $overall**",
    "",
    "| Field | Value |",
    "| --- | --- |",
    "| Baseline configuration | $(Get-ReportCell $BaselineConfiguration) |",
    "| Candidate configuration | $(Get-ReportCell $CandidateConfiguration) |",
    "| Minimum efficiency gain | $MinimumImprovementPercent% |",
    "| Defect-detection scenario present | $hasDefectDetectionScenario |",
    "",
    "## Scenario results",
    "",
    "| Scenario | Acceptance | Findings | Verdict | Best efficiency gain | Reason |",
    "| --- | --- | ---: | --- | ---: | --- |"
)
foreach ($result in $results) {
    $gain = if ($null -eq $result.best_improvement_percent) { "Not recorded" } else { "$($result.best_improvement_percent)% ($($result.best_metric))" }
    $lines += "| $(Get-ReportCell $result.scenario) | $(Get-ReportCell $result.acceptance_verdict) | $($result.finding_count) | $(Get-ReportCell $result.verdict) | $(Get-ReportCell $gain) | $(Get-ReportCell $result.reason) |"
}

if ($missing.Count -gt 0 -or -not $hasDefectDetectionScenario) {
    $lines += ""
    $lines += "## Missing required evidence"
    $lines += ""
    foreach ($item in $missing) { $lines += "- $item" }
    if (-not $hasDefectDetectionScenario) {
        $lines += "- At least one supplied required scenario must contain a known acceptance failure/finding so defect-detection equivalence is measured."
    }
}

$lines += ""
$lines += "## Decision rule"
$lines += ""
$lines += "- `Adopt medium`: every required pair tested the identical state and produced the same surface profile, acceptance verdict/ledger, finding set and evidence-coverage signature, with no blocked/not-run rows and at least $MinimumImprovementPercent% measured efficiency gain per scenario."
$lines += "- `Keep high`: medium changes acceptance classification, misses/adds a finding, changes evidence coverage, tests a different state, or fails the efficiency threshold."
$lines += "- `Inconclusive`: required scenarios, a defect-detection scenario, complete evidence, or comparable resource metrics are missing."
$lines += ""
$lines += "This report does not edit `.codex/agents/backlog-tester.toml`. Change production tester reasoning effort only through a separate explicit repository change after reviewing the evidence."
$lines += ""
$lines += ('Structured gate manifest: `{0}`' -f (Split-Path -Leaf $manifestPath))
$lines | Set-Content -LiteralPath $reportPath -Encoding utf8

Write-Output $reportPath
