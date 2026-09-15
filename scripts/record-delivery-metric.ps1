[CmdletBinding()]
param(
    [Parameter(Mandatory)]
    [ValidatePattern("^[A-Za-z0-9][A-Za-z0-9 _.-]*$")]
    [string]$Item,

    [Parameter(Mandatory)]
    [ValidateSet("plan", "implement", "review", "test", "repair-implement", "repair-review", "repair-test", "release-gate", "completion-gate")]
    [string]$Phase,

    [Parameter(Mandatory)]
    [ValidatePattern("^[A-Za-z0-9][A-Za-z0-9_.-]*$")]
    [string]$Agent,

    [Parameter(Mandatory)]
    [ValidateRange(1, 10000)]
    [int]$Attempt,

    [ValidateSet("started", "completed")]
    [string]$Event = "completed",

    [ValidateSet("Pass", "Fail", "Blocked", "Paused", "Interrupted", "Reroute", "Success", "Failure", "unavailable")]
    [string]$Result = "unavailable",

    [ValidateSet("low", "medium", "high", "unavailable")]
    [string]$ReasoningEffort = "unavailable",

    [ValidateSet("true", "false", "unavailable")]
    [string]$RepairRequired = "unavailable",

    [ValidatePattern("^(unavailable|[A-Za-z0-9][A-Za-z0-9_.:-]*)$")]
    [string]$WorkPackage = "unavailable",

    [ValidatePattern("^(unavailable|[A-Za-z0-9][A-Za-z0-9_.:-]*)$")]
    [string]$WorkSlice = "unavailable",

    [ValidateSet("routine", "defect", "migration", "authorization", "ui", "other", "unavailable")]
    [string]$WorkKind = "unavailable",

    [ValidateSet("R", "E", "H", "unavailable")]
    [string]$RiskLevel = "unavailable",

    [string]$RiskCodes,

    [int]$RepairCycle = -1,
    [int]$Turns = -1,
    [int]$ToolCalls = -1,
    [long]$InputTokens = -1,
    [long]$CachedInputTokens = -1,
    [long]$OutputTokens = -1,
    [double]$ElapsedSeconds = -1,
    [double]$EstimatedCostUsd = -1,
    [int]$R1Findings = -1,
    [int]$P0P3Findings = -1,

    [ValidatePattern("^[A-Za-z0-9][A-Za-z0-9_.:-]*$")]
    [string]$MetricSource = "execution-surface",

    [string]$Notes
)

$ErrorActionPreference = "Stop"
$root = Split-Path -Parent $PSScriptRoot
$reportRoot = Join-Path $root ".codex\agent-evals"
$implementationPhases = @("implement", "repair-implement")
$allowedRiskCodes = @("DATA", "AUTH", "FIN", "HIST", "CONC", "OPS", "API", "UI", "PERF", "DEPLOY", "DEP", "GOV", "DOC")

function Invoke-GitText([string[]]$Arguments) {
    $value = & git @Arguments 2>$null
    if ($LASTEXITCODE -ne 0) {
        throw "git $($Arguments -join ' ') failed."
    }
    return [string]($value -join "`n")
}

function Get-NullableInt([int]$Value) {
    if ($Value -lt 0) { return $null }
    return $Value
}

function Get-NullableLong([long]$Value) {
    if ($Value -lt 0) { return $null }
    return $Value
}

function Get-NullableDouble([double]$Value) {
    if ($Value -lt 0) { return $null }
    return $Value
}

function Get-ObjectProperty([object]$Object, [string]$Name) {
    if ($null -eq $Object) { return $null }
    $property = $Object.PSObject.Properties[$Name]
    if ($null -eq $property) { return $null }
    return $property.Value
}

function Get-ExistingWorkValue([object]$Record, [string]$Name) {
    $work = Get-ObjectProperty $Record "work"
    if ($null -eq $work) { return $null }
    return Get-ObjectProperty $work $Name
}

$itemSlug = ($Item.ToLowerInvariant() -replace "[^a-z0-9]+", "-").Trim("-")
if (-not $itemSlug) {
    throw "Item '$Item' does not produce a usable metrics file name."
}

if ($Event -eq "started" -and $Result -ne "unavailable") {
    throw "A started telemetry event cannot have a terminal Result."
}
if ($Event -eq "completed" -and $Result -eq "unavailable") {
    throw "A completed telemetry event requires a Result."
}

$workPackageValue = if ($WorkPackage -eq "unavailable") { $null } else { $WorkPackage }
$workSliceValue = if ($WorkSlice -eq "unavailable") { $null } else { $WorkSlice }
$workKindValue = if ($WorkKind -eq "unavailable") { $null } else { $WorkKind }
if ($null -ne $workSliceValue -and $null -eq $workPackageValue) {
    throw "WorkSlice requires WorkPackage so slice telemetry remains attributable to its planned work package."
}
if (($null -ne $workPackageValue -or $null -ne $workSliceValue -or $null -ne $workKindValue) -and $Phase -notin $implementationPhases) {
    throw "WorkPackage/WorkSlice/WorkKind telemetry is valid only for implement or repair-implement phases."
}
if ($null -ne $workKindValue -and $null -eq $workPackageValue) {
    throw "WorkKind requires WorkPackage so pre-work classification remains tied to a planned package."
}

$riskCodeValues = @()
if (-not [string]::IsNullOrWhiteSpace($RiskCodes)) {
    $riskCodeValues = @(
        $RiskCodes.Split(",") |
            ForEach-Object { $_.Trim().ToUpperInvariant() } |
            Where-Object { -not [string]::IsNullOrWhiteSpace($_) } |
            Select-Object -Unique
    )
    foreach ($code in $riskCodeValues) {
        if ($code -notin $allowedRiskCodes) {
            throw "Unknown RiskCodes value '$code'. Allowed values: $($allowedRiskCodes -join ', ')."
        }
    }
}
if (($RiskLevel -ne "unavailable" -or $riskCodeValues.Count -gt 0) -and $Phase -notin $implementationPhases) {
    throw "RiskLevel/RiskCodes telemetry is valid only for implement or repair-implement phases."
}
if ($riskCodeValues.Count -gt 0 -and $RiskLevel -eq "unavailable") {
    throw "RiskCodes requires an explicit RiskLevel."
}

$riskLevelValue = if ($RiskLevel -eq "unavailable") { $null } else { $RiskLevel }
$measurementScope = if ($null -ne $workSliceValue) { "slice" } elseif ($null -ne $workPackageValue) { "work-package" } else { "phase" }

New-Item -ItemType Directory -Path $reportRoot -Force | Out-Null
$metricsPath = Join-Path $reportRoot "delivery-$itemSlug.jsonl"

# Phase + agent + work unit + attempt is the durable invocation identity.
# Schema-v3 preserves the interruption-safe started/completed model and allows
# implementation usage to be attributed to a planned package/slice.
$existingEvents = @()
if (Test-Path -LiteralPath $metricsPath) {
    foreach ($line in Get-Content -LiteralPath $metricsPath) {
        if ([string]::IsNullOrWhiteSpace($line)) { continue }
        $existing = $line | ConvertFrom-Json
        $existingPackage = Get-ExistingWorkValue $existing "package"
        $existingSlice = Get-ExistingWorkValue $existing "slice"
        if (
            $existing.phase -eq $Phase -and
            $existing.agent -eq $Agent -and
            [int]$existing.attempt -eq $Attempt -and
            $existingPackage -eq $workPackageValue -and
            $existingSlice -eq $workSliceValue
        ) {
            $existingEvents += $existing
        }
    }
}

$hasLegacyCompleted = @($existingEvents | Where-Object { $null -eq (Get-ObjectProperty $_ "event") }).Count -gt 0
$hasStarted = @($existingEvents | Where-Object { (Get-ObjectProperty $_ "event") -eq "started" }).Count -gt 0
$hasCompleted = @($existingEvents | Where-Object { (Get-ObjectProperty $_ "event") -eq "completed" }).Count -gt 0
$identity = "$Phase|$Agent|$(if ($null -eq $workPackageValue) { '-' } else { $workPackageValue })|$(if ($null -eq $workSliceValue) { '-' } else { $workSliceValue })|$Attempt"

if ($Event -eq "started") {
    if ($hasLegacyCompleted -or $hasStarted -or $hasCompleted) {
        throw "Delivery metric invocation '$identity' already exists. Increment Attempt only for a real new invocation of the same work unit."
    }
}
else {
    if ($hasLegacyCompleted -or $hasCompleted) {
        throw "Completed delivery metric already exists for '$identity'."
    }
}

$inputValue = if ($Event -eq "completed") { Get-NullableLong $InputTokens } else { $null }
$outputValue = if ($Event -eq "completed") { Get-NullableLong $OutputTokens } else { $null }
$totalTokens = if ($null -ne $inputValue -and $null -ne $outputValue) {
    [long]$inputValue + [long]$outputValue
}
else {
    $null
}

$repairRequiredValue = if ($Event -eq "completed") {
    switch ($RepairRequired) {
        "true" { $true }
        "false" { $false }
        default { $null }
    }
}
else {
    $null
}

$head = (Invoke-GitText @("rev-parse", "HEAD")).Trim()
$branch = (Invoke-GitText @("branch", "--show-current")).Trim()

$record = [ordered]@{
    schema_version = 3
    event = $Event
    recorded_at_utc = (Get-Date).ToUniversalTime().ToString("o")
    item = $Item
    phase = $Phase
    agent = $Agent
    attempt = $Attempt
    result = if ($Event -eq "completed") { $Result } else { $null }
    reasoning_effort = if ($ReasoningEffort -eq "unavailable") { $null } else { $ReasoningEffort }
    metric_source = if ($Event -eq "completed") { $MetricSource } else { $null }
    work = [ordered]@{
        package = $workPackageValue
        slice = $workSliceValue
        kind = $workKindValue
        scope = $measurementScope
        risk_level = $riskLevelValue
        risk_codes = @($riskCodeValues)
    }
    repository = [ordered]@{
        head = $head
        branch = $branch
    }
    metrics = [ordered]@{
        turns = if ($Event -eq "completed") { Get-NullableInt $Turns } else { $null }
        tool_calls = if ($Event -eq "completed") { Get-NullableInt $ToolCalls } else { $null }
        input_tokens = $inputValue
        cached_input_tokens = if ($Event -eq "completed") { Get-NullableLong $CachedInputTokens } else { $null }
        output_tokens = $outputValue
        total_tokens = $totalTokens
        elapsed_seconds = if ($Event -eq "completed") { Get-NullableDouble $ElapsedSeconds } else { $null }
        estimated_cost_usd = if ($Event -eq "completed") { Get-NullableDouble $EstimatedCostUsd } else { $null }
    }
    quality = [ordered]@{
        repair_required = $repairRequiredValue
        repair_cycle = if ($Event -eq "completed") { Get-NullableInt $RepairCycle } else { $null }
        r1_findings = if ($Event -eq "completed") { Get-NullableInt $R1Findings } else { $null }
        p0_p3_findings = if ($Event -eq "completed") { Get-NullableInt $P0P3Findings } else { $null }
    }
    notes = if ([string]::IsNullOrWhiteSpace($Notes)) { $null } else { $Notes }
}

$line = $record | ConvertTo-Json -Depth 7 -Compress
Add-Content -LiteralPath $metricsPath -Value $line -Encoding utf8

Write-Output $metricsPath
