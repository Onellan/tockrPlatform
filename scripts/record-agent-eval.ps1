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

    [string]$BaselineReport,
    [string]$PhaseMetricsPath,
    [int]$Turns = -1,
    [int]$ToolCalls = -1,
    [double]$InputTokens = -1,
    [double]$CachedInputTokens = -1,
    [double]$OutputTokens = -1,
    [double]$ElapsedSeconds = -1,
    [double]$EstimatedCostUsd = -1,
    [int]$RepairCycles = -1,
    [int]$R1Findings = -1,
    [int]$P0P3Findings = -1,
    [string]$ReviewVerdict = "Not run",
    [string]$TestVerdict = "Not run"
)

$ErrorActionPreference = "Stop"
$root = Split-Path -Parent $PSScriptRoot
$reportRoot = Join-Path $root ".codex\agent-evals"
$allowedPhases = @("baseline_context", "implementation", "debugging", "validation", "handoff")
$allowedPhaseMetrics = @("input_tokens", "cached_input_tokens", "output_tokens", "tool_calls", "elapsed_seconds", "estimated_cost_usd")

function Invoke-GitText([string[]]$Arguments) {
    $value = & git @Arguments 2>$null
    if ($LASTEXITCODE -ne 0) {
        throw "git $($Arguments -join ' ') failed."
    }
    return [string]($value -join "`n")
}

function Get-Sha256Text([string]$Value) {
    $bytes = [System.Text.Encoding]::UTF8.GetBytes($Value)
    $hash = [System.Security.Cryptography.SHA256]::HashData($bytes)
    return [System.Convert]::ToHexString($hash).ToLowerInvariant()
}

function Get-ReportCell([object]$Value) {
    if ($null -eq $Value -or $Value -eq "") {
        return "Not recorded"
    }
    return ([string]$Value).Replace("|", "\\|").Replace("`r", " ").Replace("`n", " ")
}

function Get-Metric([double]$Value) {
    if ($Value -lt 0) {
        return $null
    }
    return $Value
}

function Get-CountMetric([int]$Value) {
    if ($Value -lt 0) {
        return $null
    }
    return $Value
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

function Read-PhaseMetrics([string]$Path) {
    if (-not $Path) {
        return [ordered]@{}
    }

    $resolved = Resolve-Path -LiteralPath $Path
    $raw = Get-Content -Raw -LiteralPath $resolved | ConvertFrom-Json
    $result = [ordered]@{}

    foreach ($phaseProperty in $raw.PSObject.Properties) {
        $phase = $phaseProperty.Name
        if ($phase -notin $allowedPhases) {
            throw "Unknown phase '$phase' in '$Path'. Allowed phases: $($allowedPhases -join ', ')."
        }

        $phaseValue = $phaseProperty.Value
        $phaseResult = [ordered]@{}
        foreach ($metricProperty in $phaseValue.PSObject.Properties) {
            $metric = $metricProperty.Name
            if ($metric -notin $allowedPhaseMetrics) {
                throw "Unknown phase metric '$metric' for '$phase'. Allowed metrics: $($allowedPhaseMetrics -join ', ')."
            }
            if ($null -eq $metricProperty.Value) {
                $phaseResult[$metric] = $null
                continue
            }
            $number = [double]$metricProperty.Value
            if ($number -lt 0) {
                throw "Phase metric '$phase.$metric' must be non-negative."
            }
            $phaseResult[$metric] = $number
        }
        $result[$phase] = $phaseResult
    }

    return $result
}

$head = Invoke-GitText @("rev-parse", "HEAD")
$branch = Invoke-GitText @("branch", "--show-current")
$status = Invoke-GitText @("status", "--short", "--untracked-files=all")
$changedPaths = Invoke-GitText @("diff", "--name-only", "HEAD")
$diff = Invoke-GitText @("diff", "--binary", "HEAD")

$configFiles = @(
    Get-ChildItem -Path (Join-Path $root ".codex\agents") -File -Filter "*.toml"
    Get-ChildItem -Path (Join-Path $root ".agents\skills") -Recurse -File |
        Where-Object { $_.Name -in @("SKILL.md", "openai.yaml") }
) | Sort-Object FullName
$configurationInventory = @(
    foreach ($file in $configFiles) {
        [ordered]@{
            path = $file.FullName.Substring($root.Length + 1).Replace("\\", "/")
            sha256 = (Get-FileHash -LiteralPath $file.FullName -Algorithm SHA256).Hash.ToLowerInvariant()
        }
    }
)

$scenarioSlug = ($Scenario.ToLowerInvariant() -replace "[^a-z0-9]+", "-").Trim("-")
$configurationSlug = ($Configuration.ToLowerInvariant() -replace "[^a-z0-9]+", "-").Trim("-")
$runId = "{0}-{1}" -f (Get-Date).ToUniversalTime().ToString("yyyyMMddTHHmmssZ"), ([guid]::NewGuid().ToString("N").Substring(0, 8))
$baseName = "agent-eval-$runId-$scenarioSlug-$configurationSlug"
New-Item -ItemType Directory -Path $reportRoot -Force | Out-Null
$reportPath = Join-Path $reportRoot "$baseName.md"
$manifestPath = Join-Path $reportRoot "$baseName.json"

$baseline = $null
if ($BaselineReport) {
    $baselineReportPath = (Resolve-Path -LiteralPath $BaselineReport).Path
    $baselineManifest = [System.IO.Path]::ChangeExtension($baselineReportPath, ".json")
    if (-not (Test-Path -LiteralPath $baselineManifest)) {
        throw "Baseline manifest was not found beside '$BaselineReport'."
    }
    $baseline = Get-Content -Raw -LiteralPath $baselineManifest | ConvertFrom-Json
    if ($baseline.scenario -ne $Scenario) {
        throw "Baseline scenario '$($baseline.scenario)' does not match '$Scenario'."
    }
}

$phaseMetrics = Read-PhaseMetrics $PhaseMetricsPath
$metrics = [ordered]@{
    turns = Get-CountMetric $Turns
    tool_calls = Get-CountMetric $ToolCalls
    input_tokens = Get-Metric $InputTokens
    cached_input_tokens = Get-Metric $CachedInputTokens
    output_tokens = Get-Metric $OutputTokens
    elapsed_seconds = Get-Metric $ElapsedSeconds
    estimated_cost_usd = Get-Metric $EstimatedCostUsd
    repair_cycles = Get-CountMetric $RepairCycles
    r1_findings = Get-CountMetric $R1Findings
    p0_p3_findings = Get-CountMetric $P0P3Findings
}
$manifest = [ordered]@{
    schema_version = 2
    run_id = $runId
    created_at_utc = (Get-Date).ToUniversalTime().ToString("o")
    scenario = $Scenario
    configuration = $Configuration
    quality = [ordered]@{
        outcome = $QualityOutcome
        evidence = $QualityEvidence
        review_verdict = $ReviewVerdict
        test_verdict = $TestVerdict
    }
    implementation = [ordered]@{
        summary = $ImplementationSummary
        head = $head.Trim()
        branch = $branch.Trim()
        status = $status
        changed_paths = $changedPaths
        diff_sha256 = Get-Sha256Text $diff
    }
    configuration_inventory = $configurationInventory
    metrics = $metrics
    phase_metrics = $phaseMetrics
    baseline_report = if ($BaselineReport) { (Resolve-Path -LiteralPath $BaselineReport).Path } else { $null }
}
$manifest | ConvertTo-Json -Depth 8 | Set-Content -LiteralPath $manifestPath -Encoding utf8

$decision = if ($QualityOutcome -ne "Pass") {
    "Reject: the evaluated run did not meet the quality gate."
}
elseif ($null -eq $baseline) {
    "Baseline recorded: compare a candidate against this same scenario before changing configuration."
}
elseif ($baseline.quality.outcome -ne "Pass") {
    "Inconclusive: the selected baseline did not pass its quality gate."
}
elseif ($ReviewVerdict -ne "Pass" -or $TestVerdict -ne "Pass" -or $baseline.quality.review_verdict -ne "Pass" -or $baseline.quality.test_verdict -ne "Pass") {
    "Inconclusive: paired reviewer/tester Pass evidence is incomplete."
}
else {
    $baselineRepair = Get-ObjectProperty $baseline.metrics "repair_cycles"
    $baselineR1 = Get-ObjectProperty $baseline.metrics "r1_findings"
    $baselineP0P3 = Get-ObjectProperty $baseline.metrics "p0_p3_findings"
    if ($null -ne $baselineRepair -and $null -ne $metrics.repair_cycles -and $metrics.repair_cycles -gt [int]$baselineRepair) {
        "Reject: the candidate required more repair/re-review/retest cycles than the baseline."
    }
    elseif ($null -ne $baselineR1 -and $null -ne $metrics.r1_findings -and $metrics.r1_findings -gt [int]$baselineR1) {
        "Reject: the candidate introduced additional R1 findings."
    }
    elseif ($null -ne $baselineP0P3 -and $null -ne $metrics.p0_p3_findings -and $metrics.p0_p3_findings -gt [int]$baselineP0P3) {
        "Reject: the candidate introduced additional P0-P3 findings."
    }
    else {
        "Paired scenario eligible: include this result in the multi-scenario reasoning-effort gate; do not change production configuration from one scenario."
    }
}

$comparisonRows = @()
if ($null -ne $baseline) {
    foreach ($name in $metrics.Keys) {
        $baselineValue = Get-ObjectProperty $baseline.metrics $name
        $candidateValue = $metrics[$name]
        if ($null -ne $baselineValue -and $null -ne $candidateValue) {
            $delta = [math]::Round(([double]$candidateValue - [double]$baselineValue), 2)
            $comparisonRows += "| $name | $(Get-ReportCell $baselineValue) | $(Get-ReportCell $candidateValue) | $(Get-ReportCell $delta) |"
        }
    }
}

$phaseRows = @()
foreach ($phase in $allowedPhases) {
    if (-not $phaseMetrics.Contains($phase)) {
        continue
    }
    foreach ($metric in $allowedPhaseMetrics) {
        $candidateValue = if ($phaseMetrics[$phase].Contains($metric)) { $phaseMetrics[$phase][$metric] } else { $null }
        if ($null -eq $candidateValue) {
            continue
        }
        $phaseRows += "| $phase | $metric | $(Get-ReportCell $candidateValue) |"
    }
}

$phaseComparisonRows = @()
if ($null -ne $baseline -and $null -ne (Get-ObjectProperty $baseline "phase_metrics")) {
    foreach ($phase in $allowedPhases) {
        $baselinePhase = Get-ObjectProperty $baseline.phase_metrics $phase
        $candidatePhase = if ($phaseMetrics.Contains($phase)) { $phaseMetrics[$phase] } else { $null }
        if ($null -eq $baselinePhase -or $null -eq $candidatePhase) {
            continue
        }
        foreach ($metric in $allowedPhaseMetrics) {
            $baselineValue = Get-ObjectProperty $baselinePhase $metric
            $candidateValue = if ($candidatePhase.Contains($metric)) { $candidatePhase[$metric] } else { $null }
            if ($null -ne $baselineValue -and $null -ne $candidateValue) {
                $delta = [math]::Round(([double]$candidateValue - [double]$baselineValue), 2)
                $phaseComparisonRows += "| $phase | $metric | $(Get-ReportCell $baselineValue) | $(Get-ReportCell $candidateValue) | $(Get-ReportCell $delta) |"
            }
        }
    }
}

$lines = @(
    "# Agent Efficiency Evaluation — $runId",
    "",
    "| Field | Value |",
    "| --- | --- |",
    "| Scenario | $(Get-ReportCell $Scenario) |",
    "| Configuration | $(Get-ReportCell $Configuration) |",
    "| Quality outcome | $(Get-ReportCell $QualityOutcome) |",
    "| Review verdict | $(Get-ReportCell $ReviewVerdict) |",
    "| Test verdict | $(Get-ReportCell $TestVerdict) |",
    "| Implementation commit | $(Get-ReportCell $manifest.implementation.head) |",
    "| Implementation diff SHA-256 | $(Get-ReportCell $manifest.implementation.diff_sha256) |",
    "| Configuration files fingerprinted | $($configurationInventory.Count) |",
    "| Report manifest | $(Split-Path -Leaf $manifestPath) |",
    "",
    "## Implementation under evaluation",
    "",
    $ImplementationSummary,
    "",
    "## Quality evidence",
    "",
    $QualityEvidence,
    "",
    "## Repository change set",
    "",
    ('- Commit: `{0}`' -f (Get-ReportCell $manifest.implementation.head)),
    ('- Diff SHA-256: `{0}`' -f (Get-ReportCell $manifest.implementation.diff_sha256)),
    "- Changed tracked paths at recording:",
    "",
    '```text',
    $(if ($changedPaths) { $changedPaths } else { "No tracked diff from HEAD." }),
    '```',
    "",
    "## Resource metrics",
    "",
    "| Metric | Value |",
    "| --- | --- |"
)
foreach ($name in $metrics.Keys) {
    $lines += "| $name | $(Get-ReportCell $metrics[$name]) |"
}
if ($phaseRows.Count -gt 0) {
    $lines += ""
    $lines += "## Phase metrics"
    $lines += ""
    $lines += "| Phase | Metric | Value |"
    $lines += "| --- | --- | ---: |"
    $lines += $phaseRows
}
$lines += ""
$lines += "## Decision"
$lines += ""
$lines += $decision
if ($comparisonRows.Count -gt 0) {
    $lines += ""
    $lines += "## Baseline comparison"
    $lines += ""
    $lines += "| Metric | Baseline | Candidate | Delta |"
    $lines += "| --- | ---: | ---: | ---: |"
    $lines += $comparisonRows
}
if ($phaseComparisonRows.Count -gt 0) {
    $lines += ""
    $lines += "## Phase comparison"
    $lines += ""
    $lines += "| Phase | Metric | Baseline | Candidate | Delta |"
    $lines += "| --- | --- | ---: | ---: | ---: |"
    $lines += $phaseComparisonRows
}
$lines += ""
$lines += "## Reproducibility"
$lines += ""
$lines += "- Branch: $(Get-ReportCell $manifest.implementation.branch)"
$lines += ('- Worktree status at recording: `{0}`' -f (Get-ReportCell $manifest.implementation.status))
$lines += ('- Configuration inventory and full structured evidence: `{0}`' -f (Split-Path -Leaf $manifestPath))
$lines | Set-Content -LiteralPath $reportPath -Encoding utf8

Write-Output $reportPath
