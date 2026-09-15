[CmdletBinding()]
param(
    [Parameter(Mandatory)]
    [ValidatePattern("^[A-Za-z0-9][A-Za-z0-9 _.-]*$")]
    [string]$Item,

    [string]$MetricsPath,
    [string]$OutputPath
)

$ErrorActionPreference = "Stop"
$root = Split-Path -Parent $PSScriptRoot
$reportRoot = Join-Path $root ".codex\agent-evals"
$phaseOrder = @(
    "plan",
    "implement",
    "review",
    "test",
    "repair-implement",
    "repair-review",
    "repair-test",
    "release-gate",
    "completion-gate"
)
$requiredReleasePhases = @("plan", "implement", "review", "test", "release-gate", "completion-gate")
$implementationPhases = @("implement", "repair-implement")
$metricNames = @(
    "input_tokens",
    "cached_input_tokens",
    "output_tokens",
    "total_tokens",
    "tool_calls",
    "elapsed_seconds",
    "estimated_cost_usd"
)

function Get-ObjectProperty([object]$Object, [string]$Name) {
    if ($null -eq $Object) { return $null }
    $property = $Object.PSObject.Properties[$Name]
    if ($null -eq $property) { return $null }
    return $property.Value
}

function Get-WorkValue([object]$Record, [string]$Name) {
    $work = Get-ObjectProperty $Record "work"
    if ($null -eq $work) { return $null }
    return Get-ObjectProperty $work $Name
}

function Get-InvocationKey([object]$Record) {
    $package = Get-WorkValue $Record "package"
    $slice = Get-WorkValue $Record "slice"
    $packageKey = if ($null -eq $package) { "-" } else { [string]$package }
    $sliceKey = if ($null -eq $slice) { "-" } else { [string]$slice }
    return "$($Record.phase)|$($Record.agent)|$packageKey|$sliceKey|$($Record.attempt)"
}

function Resolve-InvocationRecords([object[]]$RawRecords) {
    $groups = [ordered]@{}
    $order = @()

    foreach ($record in $RawRecords) {
        $key = Get-InvocationKey $record
        if (-not $groups.Contains($key)) {
            $groups[$key] = [ordered]@{
                started = $null
                completed = $null
                legacy = $false
            }
            $order += $key
        }

        $group = $groups[$key]
        $event = Get-ObjectProperty $record "event"
        if ($null -eq $event) {
            if ($null -ne $group.started -or $null -ne $group.completed -or $group.legacy) {
                throw "Conflicting legacy/event delivery telemetry for invocation '$key'."
            }
            $group.completed = $record
            $group.legacy = $true
            continue
        }

        switch ($event) {
            "started" {
                if ($null -ne $group.started) {
                    throw "Duplicate started telemetry event for invocation '$key'."
                }
                $group.started = $record
            }
            "completed" {
                if ($null -ne $group.completed) {
                    throw "Duplicate completed telemetry event for invocation '$key'."
                }
                $group.completed = $record
            }
            default {
                throw "Unknown delivery telemetry event '$event' for invocation '$key'."
            }
        }
    }

    $resolved = @()
    foreach ($key in $order) {
        $group = $groups[$key]
        if ($null -ne $group.completed) {
            $record = $group.completed
            $state = if ($null -ne $group.started) { "completed" } elseif ($group.legacy) { "legacy-completed" } else { "completed-without-start" }
            $durableStart = ($null -ne $group.started)
        }
        elseif ($null -ne $group.started) {
            $record = $group.started
            $state = "incomplete"
            $durableStart = $true
            $record | Add-Member -NotePropertyName result -NotePropertyValue "Interrupted" -Force
        }
        else {
            throw "Delivery telemetry invocation '$key' has no usable event."
        }

        $record | Add-Member -NotePropertyName invocation_state -NotePropertyValue $state -Force
        $record | Add-Member -NotePropertyName durable_start -NotePropertyValue $durableStart -Force
        $record | Add-Member -NotePropertyName work_package -NotePropertyValue (Get-WorkValue $record "package") -Force
        $record | Add-Member -NotePropertyName work_slice -NotePropertyValue (Get-WorkValue $record "slice") -Force
        $record | Add-Member -NotePropertyName measurement_scope -NotePropertyValue (Get-WorkValue $record "scope") -Force
        $record | Add-Member -NotePropertyName risk_level -NotePropertyValue (Get-WorkValue $record "risk_level") -Force
        $record | Add-Member -NotePropertyName risk_codes -NotePropertyValue (Get-WorkValue $record "risk_codes") -Force
        $resolved += $record
    }

    return $resolved
}

function Get-MetricValues([object[]]$Records, [string]$Metric) {
    $values = @()
    foreach ($record in $Records) {
        $metrics = Get-ObjectProperty $record "metrics"
        $value = Get-ObjectProperty $metrics $Metric
        if ($null -ne $value) {
            $values += [double]$value
        }
    }
    return $values
}

function Get-Aggregate([object[]]$Records, [string]$Metric) {
    $values = @(Get-MetricValues $Records $Metric)
    $count = $values.Count
    $totalRecords = $Records.Count
    if ($count -eq 0) {
        return [ordered]@{
            value = $null
            coverage = "0/$totalRecords"
            complete = $false
        }
    }

    $sum = ($values | Measure-Object -Sum).Sum
    if ($Metric -in @("input_tokens", "cached_input_tokens", "output_tokens", "total_tokens", "tool_calls")) {
        $sum = [long][math]::Round([double]$sum, 0)
    }
    else {
        $sum = [math]::Round([double]$sum, 2)
    }

    return [ordered]@{
        value = $sum
        coverage = "$count/$totalRecords"
        complete = ($count -eq $totalRecords)
    }
}

function Format-Aggregate([object]$Aggregate) {
    if ($null -eq $Aggregate.value) {
        return "unavailable ($($Aggregate.coverage))"
    }
    return "$($Aggregate.value) ($($Aggregate.coverage))"
}

function Format-Risk([object]$Record) {
    $level = $Record.risk_level
    $codes = @($Record.risk_codes)
    if ($null -eq $level) { return "unavailable" }
    if ($codes.Count -eq 0) { return [string]$level }
    return "$level[$($codes -join ',')]"
}

function Get-DistinctReasoning([object[]]$Records) {
    $values = @(
        $Records |
            ForEach-Object { if ($null -eq $_.reasoning_effort) { "unavailable" } else { [string]$_.reasoning_effort } } |
            Select-Object -Unique
    )
    return ($values -join ",")
}

function Get-DistinctRisk([object[]]$Records) {
    $values = @($Records | ForEach-Object { Format-Risk $_ } | Select-Object -Unique)
    return ($values -join ",")
}

function Get-GroupedRecords([object[]]$Records, [scriptblock]$KeySelector) {
    $groups = [ordered]@{}
    foreach ($record in $Records) {
        $key = & $KeySelector $record
        if (-not $groups.Contains($key)) {
            $groups[$key] = @()
        }
        $groups[$key] += $record
    }
    return $groups
}

$itemSlug = ($Item.ToLowerInvariant() -replace "[^a-z0-9]+", "-").Trim("-")
if (-not $itemSlug) {
    throw "Item '$Item' does not produce a usable metrics file name."
}

if (-not $MetricsPath) {
    $MetricsPath = Join-Path $reportRoot "delivery-$itemSlug.jsonl"
}
$resolvedMetrics = (Resolve-Path -LiteralPath $MetricsPath).Path

$rawRecords = @()
foreach ($line in Get-Content -LiteralPath $resolvedMetrics) {
    if ([string]::IsNullOrWhiteSpace($line)) { continue }
    $record = $line | ConvertFrom-Json
    if ($record.item -ne $Item) {
        throw "Metrics file contains item '$($record.item)' while '$Item' was requested."
    }
    $rawRecords += $record
}

if ($rawRecords.Count -eq 0) {
    throw "No delivery metrics were found in '$resolvedMetrics'."
}

# Resolve started/completed event pairs into exactly one logical invocation.
# A started event without a matching completed event remains an interrupted
# invocation with unavailable counters. Work-package/slice views are rollups of
# the same logical invocations, never additional cost records.
$records = @(Resolve-InvocationRecords $rawRecords)

$phaseRows = @()
foreach ($phase in $phaseOrder) {
    $phaseRecords = @($records | Where-Object { $_.phase -eq $phase })
    if ($phaseRecords.Count -eq 0) { continue }

    $input = Get-Aggregate $phaseRecords "input_tokens"
    $cached = Get-Aggregate $phaseRecords "cached_input_tokens"
    $output = Get-Aggregate $phaseRecords "output_tokens"
    $total = Get-Aggregate $phaseRecords "total_tokens"
    $tools = Get-Aggregate $phaseRecords "tool_calls"
    $elapsed = Get-Aggregate $phaseRecords "elapsed_seconds"

    $phaseRows += "| $phase | $($phaseRecords.Count) | $(Format-Aggregate $input) | $(Format-Aggregate $cached) | $(Format-Aggregate $output) | $(Format-Aggregate $total) | $(Format-Aggregate $tools) | $(Format-Aggregate $elapsed) |"
}

$allTotals = [ordered]@{}
foreach ($metric in $metricNames) {
    $allTotals[$metric] = Get-Aggregate $records $metric
}

$implementationRecords = @($records | Where-Object { $_.phase -in $implementationPhases })
$attributedImplementationRecords = @($implementationRecords | Where-Object { $null -ne $_.work_package })
$sliceImplementationRecords = @($attributedImplementationRecords | Where-Object { $null -ne $_.work_slice })
$unattributedImplementationRecords = @($implementationRecords | Where-Object { $null -eq $_.work_package })

$workPackageRows = @()
if ($attributedImplementationRecords.Count -gt 0) {
    $packageGroups = Get-GroupedRecords $attributedImplementationRecords { param($record) [string]$record.work_package }
    foreach ($package in $packageGroups.Keys) {
        $packageRecords = @($packageGroups[$package])
        $repairCount = @($packageRecords | Where-Object { $_.phase -eq "repair-implement" }).Count
        $total = Get-Aggregate $packageRecords "total_tokens"
        $tools = Get-Aggregate $packageRecords "tool_calls"
        $elapsed = Get-Aggregate $packageRecords "elapsed_seconds"
        $workPackageRows += "| $package | $($packageRecords.Count) | $repairCount | $(Get-DistinctRisk $packageRecords) | $(Get-DistinctReasoning $packageRecords) | $(Format-Aggregate $total) | $(Format-Aggregate $tools) | $(Format-Aggregate $elapsed) |"
    }
}

$sliceRows = @()
if ($sliceImplementationRecords.Count -gt 0) {
    $sliceGroups = Get-GroupedRecords $sliceImplementationRecords {
        param($record)
        return "$($record.work_package)|$($record.work_slice)"
    }
    foreach ($key in $sliceGroups.Keys) {
        $sliceRecords = @($sliceGroups[$key])
        $first = $sliceRecords[0]
        $repairCount = @($sliceRecords | Where-Object { $_.phase -eq "repair-implement" }).Count
        $total = Get-Aggregate $sliceRecords "total_tokens"
        $tools = Get-Aggregate $sliceRecords "tool_calls"
        $elapsed = Get-Aggregate $sliceRecords "elapsed_seconds"
        $sliceRows += "| $($first.work_package) | $($first.work_slice) | $($sliceRecords.Count) | $repairCount | $(Get-DistinctRisk $sliceRecords) | $(Get-DistinctReasoning $sliceRecords) | $(Format-Aggregate $total) | $(Format-Aggregate $tools) | $(Format-Aggregate $elapsed) |"
    }
}

$repairRecords = @($records | Where-Object { $_.phase -like "repair-*" })
$initialRecords = @($records | Where-Object { $_.phase -notlike "repair-*" })
$repairTokenAggregate = Get-Aggregate $repairRecords "total_tokens"
$initialTokenAggregate = Get-Aggregate $initialRecords "total_tokens"

$repairCycles = @(
    foreach ($record in $repairRecords) {
        $quality = Get-ObjectProperty $record "quality"
        $cycle = Get-ObjectProperty $quality "repair_cycle"
        if ($null -ne $cycle) { [int]$cycle }
    }
)
$maxRepairCycle = if ($repairCycles.Count -gt 0) { ($repairCycles | Measure-Object -Maximum).Maximum } else { 0 }

$incompleteRecords = @($records | Where-Object { $_.invocation_state -eq "incomplete" })
$completedWithoutDurableStart = @($records | Where-Object { -not [bool]$_.durable_start })
$allInvocationsDurablyStarted = ($completedWithoutDurableStart.Count -eq 0)
$allInvocationsCompleted = ($incompleteRecords.Count -eq 0)

$completionRecords = @($records | Where-Object { $_.phase -eq "completion-gate" -and $_.invocation_state -ne "incomplete" })
$released = $false
if ($completionRecords.Count -gt 0) {
    $lastCompletion = $completionRecords | Sort-Object attempt | Select-Object -Last 1
    $released = $lastCompletion.result -in @("Pass", "Success")
}

$firstRecord = $records[0]
$streamStartsAtPlan = (
    $firstRecord.phase -eq "plan" -and
    [int]$firstRecord.attempt -eq 1 -and
    [bool]$firstRecord.durable_start
)
$missingRequiredPhases = @(
    foreach ($phase in $requiredReleasePhases) {
        if (@($records | Where-Object { $_.phase -eq $phase }).Count -eq 0) {
            $phase
        }
    }
)
$requiredLifecycleRecorded = ($missingRequiredPhases.Count -eq 0)
$invocationCoverageCompleteForRelease = (
    $streamStartsAtPlan -and
    $requiredLifecycleRecorded -and
    $allInvocationsDurablyStarted -and
    $allInvocationsCompleted
)

$tokenPerRelease = $null
if (
    $released -and
    $invocationCoverageCompleteForRelease -and
    $allTotals.total_tokens.complete -and
    $null -ne $allTotals.total_tokens.value
) {
    $tokenPerRelease = $allTotals.total_tokens.value
}

$tokenCoverage = $allTotals.total_tokens.coverage
$implementationAttributionCoverage = "$($attributedImplementationRecords.Count)/$($implementationRecords.Count)"
$implementationTokenAggregate = Get-Aggregate $implementationRecords "total_tokens"
$attributedTokenAggregate = Get-Aggregate $attributedImplementationRecords "total_tokens"
$createdAt = (Get-Date).ToUniversalTime().ToString("o")
if (-not $OutputPath) {
    New-Item -ItemType Directory -Path $reportRoot -Force | Out-Null
    $OutputPath = Join-Path $reportRoot "delivery-$itemSlug-summary.md"
}

$lines = @(
    "# Delivery Metrics — $Item",
    "",
    "Generated: $createdAt",
    "",
    "Metrics are summed only where the execution surface supplied an exact value. Coverage is shown as `measured invocations / invocations`; unavailable counters are never treated as zero or estimated from text length. A durable `started` event with no `completed` event is retained as an interrupted invocation. Work-package/slice tables are attribution views over the same implementation invocations and therefore do not add to delivery totals.",
    "",
    "## Phase totals",
    "",
    "| Phase | Invocations | Input tokens | Cached input | Output tokens | Total tokens | Tool calls | Elapsed seconds |",
    "| --- | ---: | ---: | ---: | ---: | ---: | ---: | ---: |"
)
$lines += $phaseRows
$lines += ""
$lines += "## Implementation work-unit coverage"
$lines += ""
$lines += "| Field | Value |"
$lines += "| --- | --- |"
$lines += "| Implementation invocations | $($implementationRecords.Count) |"
$lines += "| Attributed to planned work package | $($attributedImplementationRecords.Count) |"
$lines += "| Attributed to explicit slice | $($sliceImplementationRecords.Count) |"
$lines += "| Unattributed implementation invocations | $($unattributedImplementationRecords.Count) |"
$lines += "| Work-package attribution coverage | $implementationAttributionCoverage |"
$lines += "| All implementation token total | $(Format-Aggregate $implementationTokenAggregate) |"
$lines += "| Work-package-attributed token total | $(Format-Aggregate $attributedTokenAggregate) |"
$lines += ""

if ($workPackageRows.Count -gt 0) {
    $lines += "## Implementation work-package totals"
    $lines += ""
    $lines += "| Work package | Invocations | Repair invocations | Risk | Reasoning | Total tokens | Tool calls | Elapsed seconds |"
    $lines += "| --- | ---: | ---: | --- | --- | ---: | ---: | ---: |"
    $lines += $workPackageRows
    $lines += ""
}

if ($sliceRows.Count -gt 0) {
    $lines += "## Implementation slice totals"
    $lines += ""
    $lines += "| Work package | Slice | Invocations | Repair invocations | Risk | Reasoning | Total tokens | Tool calls | Elapsed seconds |"
    $lines += "| --- | --- | ---: | ---: | --- | --- | ---: | ---: | ---: |"
    $lines += $sliceRows
    $lines += ""
}

$lines += "## Delivery totals"
$lines += ""
$lines += "| Metric | Measured total (coverage) |"
$lines += "| --- | ---: |"
foreach ($metric in $metricNames) {
    $lines += "| $metric | $(Format-Aggregate $allTotals[$metric]) |"
}
$lines += ""
$lines += "## Interruption coverage"
$lines += ""
$lines += "| Field | Value |"
$lines += "| --- | --- |"
$lines += "| Logical invocations | $($records.Count) |"
$lines += "| Raw telemetry events | $($rawRecords.Count) |"
$lines += "| Incomplete/orphaned started invocations | $($incompleteRecords.Count) |"
$lines += "| Invocations lacking durable started event | $($completedWithoutDurableStart.Count) |"
$lines += "| All invocations durably started | $allInvocationsDurablyStarted |"
$lines += "| All invocations completed | $allInvocationsCompleted |"
$lines += ""
$lines += "## Repair overhead"
$lines += ""
$lines += "| Field | Value |"
$lines += "| --- | --- |"
$lines += "| Repair phase invocations | $($repairRecords.Count) |"
$lines += "| Highest recorded repair cycle | $maxRepairCycle |"
$lines += "| Initial/non-repair token total | $(Format-Aggregate $initialTokenAggregate) |"
$lines += "| Repair token total | $(Format-Aggregate $repairTokenAggregate) |"
$lines += ""
$lines += "## Release efficiency"
$lines += ""
$lines += "| Field | Value |"
$lines += "| --- | --- |"
$lines += "| Successful completion gate recorded | $released |"
$lines += "| Telemetry stream begins at durably started plan attempt 1 | $streamStartsAtPlan |"
$lines += "| Required lifecycle phases recorded | $requiredLifecycleRecorded |"
$lines += "| Missing required lifecycle phases | $(if ($missingRequiredPhases.Count -eq 0) { 'none' } else { $missingRequiredPhases -join ', ' }) |"
$lines += "| Complete invocation coverage | $invocationCoverageCompleteForRelease |"
$lines += "| Total-token telemetry coverage | $tokenCoverage |"
$lines += "| Tokens per successfully released backlog item | $(if ($null -eq $tokenPerRelease) { 'unavailable until release succeeds with complete interruption-safe invocation and total-token coverage' } else { $tokenPerRelease }) |"
$lines += ""
$lines += "## Invocation detail"
$lines += ""
$lines += "| Phase | Work package | Slice | Agent | Attempt | State | Result | Risk | Reasoning | Total tokens | Tools | Elapsed seconds | Repair cycle |"
$lines += "| --- | --- | --- | --- | ---: | --- | --- | --- | --- | ---: | ---: | ---: | ---: |"
foreach ($record in $records) {
    $metrics = Get-ObjectProperty $record "metrics"
    $quality = Get-ObjectProperty $record "quality"
    $total = Get-ObjectProperty $metrics "total_tokens"
    $tools = Get-ObjectProperty $metrics "tool_calls"
    $elapsed = Get-ObjectProperty $metrics "elapsed_seconds"
    $repairCycle = Get-ObjectProperty $quality "repair_cycle"
    $reasoning = if ($null -eq $record.reasoning_effort) { "unavailable" } else { $record.reasoning_effort }
    $package = if ($null -eq $record.work_package) { "-" } else { $record.work_package }
    $slice = if ($null -eq $record.work_slice) { "-" } else { $record.work_slice }
    $lines += "| $($record.phase) | $package | $slice | $($record.agent) | $($record.attempt) | $($record.invocation_state) | $($record.result) | $(Format-Risk $record) | $reasoning | $(if ($null -eq $total) { 'unavailable' } else { $total }) | $(if ($null -eq $tools) { 'unavailable' } else { $tools }) | $(if ($null -eq $elapsed) { 'unavailable' } else { $elapsed }) | $(if ($null -eq $repairCycle) { '-' } else { $repairCycle }) |"
}

$lines | Set-Content -LiteralPath $OutputPath -Encoding utf8
Write-Output $OutputPath
