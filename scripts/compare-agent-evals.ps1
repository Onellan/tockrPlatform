[CmdletBinding()]
param(
    [Parameter(Mandatory)]
    [string[]]$CandidateReports,

    [string]$BaselineConfiguration = "implementer high",
    [string]$CandidateConfiguration = "implementer medium",

    [ValidateRange(0.1, 100.0)]
    [double]$MinimumImprovementPercent = 10.0,

    [switch]$RequireUserVisibleWorkflow
)

$ErrorActionPreference = "Stop"
$root = Split-Path -Parent $PSScriptRoot
$reportRoot = Join-Path $root ".codex\agent-evals"
$efficiencyMetrics = @("input_tokens", "output_tokens", "tool_calls", "elapsed_seconds", "estimated_cost_usd")
$qualityCountMetrics = @("repair_cycles", "r1_findings", "p0_p3_findings")

function Get-ReportCell([object]$Value) {
    if ($null -eq $Value -or $Value -eq "") {
        return "Not recorded"
    }
    return ([string]$Value).Replace("|", "\\|").Replace("`r", " ").Replace("`n", " ")
}

function Get-ObjectProperty([object]$Object, [string]$Name) {
    if ($null -eq $Object) {
        return $null
    }
    $property = $Object.PSObject.Properties[$Name]
    if ($null -eq $property) {
        return $null
    }
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
    if ($Baseline -le 0) {
        return $null
    }
    return [math]::Round((($Baseline - $Candidate) / $Baseline) * 100.0, 2)
}

$results = @()
$scenarioIndex = @{}
$hardReject = $false
$inconclusive = $false

foreach ($candidateReport in $CandidateReports) {
    $candidateRecord = Get-ManifestForReport $candidateReport
    $candidate = $candidateRecord.manifest

    if ($candidate.configuration -ne $CandidateConfiguration) {
        throw "Candidate report '$candidateReport' uses configuration '$($candidate.configuration)', expected '$CandidateConfiguration'."
    }
    if (-not $candidate.baseline_report) {
        throw "Candidate report '$candidateReport' does not link a baseline report."
    }

    $baselineRecord = Get-ManifestForReport ([string]$candidate.baseline_report)
    $baseline = $baselineRecord.manifest
    if ($baseline.configuration -ne $BaselineConfiguration) {
        throw "Baseline for '$candidateReport' uses configuration '$($baseline.configuration)', expected '$BaselineConfiguration'."
    }
    if ($baseline.scenario -ne $candidate.scenario) {
        throw "Scenario mismatch: baseline '$($baseline.scenario)' vs candidate '$($candidate.scenario)'."
    }

    $scenario = [string]$candidate.scenario
    if ($scenarioIndex.ContainsKey($scenario)) {
        throw "More than one candidate report was supplied for scenario '$scenario'. Supply one explicit pair per scenario."
    }
    $scenarioIndex[$scenario] = $true

    $reasons = @()
    $verdict = "Pass"

    if ($baseline.quality.outcome -ne "Pass" -or $candidate.quality.outcome -ne "Pass") {
        $verdict = "Keep high"
        $reasons += "baseline/candidate quality outcome is not Pass"
        $hardReject = $true
    }
    if ($baseline.quality.review_verdict -ne "Pass" -or $candidate.quality.review_verdict -ne "Pass") {
        $verdict = "Keep high"
        $reasons += "independent reviewer Pass is missing"
        $hardReject = $true
    }
    if ($baseline.quality.test_verdict -ne "Pass" -or $candidate.quality.test_verdict -ne "Pass") {
        $verdict = "Keep high"
        $reasons += "independent tester Pass is missing"
        $hardReject = $true
    }

    foreach ($metric in $qualityCountMetrics) {
        $baselineValue = Get-ObjectProperty $baseline.metrics $metric
        $candidateValue = Get-ObjectProperty $candidate.metrics $metric
        if ($null -eq $baselineValue -or $null -eq $candidateValue) {
            if ($verdict -ne "Keep high") {
                $verdict = "Inconclusive"
            }
            $reasons += "$metric not recorded for both runs"
            $inconclusive = $true
            continue
        }
        if ([double]$candidateValue -gt [double]$baselineValue) {
            $verdict = "Keep high"
            $reasons += "$metric increased ($baselineValue -> $candidateValue)"
            $hardReject = $true
        }
    }

    $bestMetric = $null
    $bestImprovement = $null
    foreach ($metric in $efficiencyMetrics) {
        $baselineValue = Get-ObjectProperty $baseline.metrics $metric
        $candidateValue = Get-ObjectProperty $candidate.metrics $metric
        if ($null -eq $baselineValue -or $null -eq $candidateValue) {
            continue
        }
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
        $reasons += "quality equivalent; best efficiency gain $bestImprovement% on $bestMetric"
    }

    $results += [ordered]@{
        scenario = $scenario
        verdict = $verdict
        reason = ($reasons -join "; ")
        best_metric = $bestMetric
        best_improvement_percent = $bestImprovement
        baseline_report = $baselineRecord.report
        candidate_report = $candidateRecord.report
    }
}

$requiredExact = @("Routine bounded behavior change", "Defect correction")
if ($RequireUserVisibleWorkflow) {
    $requiredExact += "User-visible workflow"
}

$missing = @()
foreach ($scenario in $requiredExact) {
    if (-not $scenarioIndex.ContainsKey($scenario)) {
        $missing += $scenario
    }
}
$hasIntegrity = $scenarioIndex.ContainsKey("Schema and migration change") -or $scenarioIndex.ContainsKey("Authorization/privacy change")
if (-not $hasIntegrity) {
    $missing += "Schema and migration change OR Authorization/privacy change"
}

if ($missing.Count -gt 0) {
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
$baseName = "agent-eval-gate-$runId"
$reportPath = Join-Path $reportRoot "$baseName.md"
$manifestPath = Join-Path $reportRoot "$baseName.json"

$manifest = [ordered]@{
    schema_version = 1
    run_id = $runId
    created_at_utc = (Get-Date).ToUniversalTime().ToString("o")
    baseline_configuration = $BaselineConfiguration
    candidate_configuration = $CandidateConfiguration
    minimum_improvement_percent = $MinimumImprovementPercent
    require_user_visible_workflow = [bool]$RequireUserVisibleWorkflow
    required_scenarios_missing = $missing
    result = $overall
    scenario_results = $results
}
$manifest | ConvertTo-Json -Depth 8 | Set-Content -LiteralPath $manifestPath -Encoding utf8

$lines = @(
    "# Agent Reasoning-Effort Gate — $runId",
    "",
    "**Result: $overall**",
    "",
    "| Field | Value |",
    "| --- | --- |",
    "| Baseline configuration | $(Get-ReportCell $BaselineConfiguration) |",
    "| Candidate configuration | $(Get-ReportCell $CandidateConfiguration) |",
    "| Minimum efficiency gain | $MinimumImprovementPercent% |",
    "| User-visible workflow required | $([bool]$RequireUserVisibleWorkflow) |",
    "",
    "## Scenario results",
    "",
    "| Scenario | Verdict | Best efficiency gain | Reason |",
    "| --- | --- | ---: | --- |"
)
foreach ($result in $results) {
    $gain = if ($null -eq $result.best_improvement_percent) { "Not recorded" } else { "$($result.best_improvement_percent)% ($($result.best_metric))" }
    $lines += "| $(Get-ReportCell $result.scenario) | $(Get-ReportCell $result.verdict) | $(Get-ReportCell $gain) | $(Get-ReportCell $result.reason) |"
}

if ($missing.Count -gt 0) {
    $lines += ""
    $lines += "## Missing required evidence"
    $lines += ""
    foreach ($item in $missing) {
        $lines += "- $item"
    }
}

$lines += ""
$lines += "## Decision rule"
$lines += ""
$lines += "- `Adopt medium`: all required paired scenarios have equivalent independent quality evidence, no extra repair/R1/P0-P3 counts, and at least $MinimumImprovementPercent% measured efficiency gain per scenario."
$lines += "- `Keep high`: any supplied scenario loses quality, adds repair/findings, or fails the minimum efficiency gain."
$lines += "- `Inconclusive`: required scenarios or machine-checkable evidence are missing."
$lines += ""
$lines += "This report does not edit `.codex/agents/backlog-implementer.toml`. Change production reasoning effort only through a separate explicit repository change after reviewing this evidence."
$lines += ""
$lines += ('Structured gate manifest: `{0}`' -f (Split-Path -Leaf $manifestPath))
$lines | Set-Content -LiteralPath $reportPath -Encoding utf8

Write-Output $reportPath
