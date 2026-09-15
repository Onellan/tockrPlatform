[CmdletBinding()]
param(
    [Parameter(Mandatory)]
    [ValidatePattern("^[A-Za-z0-9][A-Za-z0-9_.:-]*$")]
    [string]$WorkPackage,

    [Parameter(Mandatory)]
    [ValidateSet("routine", "defect", "migration", "authorization", "ui", "other")]
    [string]$WorkKind,

    [Parameter(Mandatory)]
    [ValidateSet("R", "E", "H")]
    [string]$RiskLevel,

    [string]$RiskCodes,
    [Parameter(Mandatory)]
    [ValidateNotNullOrEmpty()]
    [string]$PlanPath,
    [string]$PolicyPath,
    [string]$ReportRoot,

    [ValidateRange(1, 1000)]
    [int]$MinimumHistorySamples = 3
)

$ErrorActionPreference = "Stop"
$root = Split-Path -Parent $PSScriptRoot
$python = Get-Command python -ErrorAction SilentlyContinue
if ($null -eq $python) { $python = Get-Command python3 -ErrorAction SilentlyContinue }
if ($null -eq $python) { throw "Python 3 is required for the canonical delivery contract." }
$riskValue = if ($RiskCodes) { $RiskCodes } else { "" }

$arguments = @(
    (Join-Path $PSScriptRoot "recommend_implementer_reasoning.py"),
    "--work-package", $WorkPackage,
    "--work-kind", $WorkKind,
    "--risk-level", $RiskLevel,
    "--risk-codes", $riskValue,
    "--minimum-history-samples", $MinimumHistorySamples
)
$arguments += @("--plan", $PlanPath)
if ($PolicyPath) { $arguments += @("--policy-path", $PolicyPath) }
if ($ReportRoot) { $arguments += @("--report-root", $ReportRoot) }

& $python.Source @arguments
exit $LASTEXITCODE
