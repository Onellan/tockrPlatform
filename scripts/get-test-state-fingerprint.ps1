[CmdletBinding()]
param(
    [Parameter(Mandatory)]
    [ValidateNotNullOrEmpty()]
    [string]$PlanPath,

    [Parameter(Mandatory)]
    [ValidateNotNullOrEmpty()]
    [string[]]$AuthorityPaths
)

$ErrorActionPreference = "Stop"
$root = (Resolve-Path (Split-Path -Parent $PSScriptRoot)).Path
$agentControlPrefixes = @(".codex/", ".agents/")

function Invoke-GitText([string[]]$Arguments) {
    $value = & git -C $root @Arguments 2>$null
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

function Resolve-RepositoryFile([string]$Path) {
    $candidate = if ([System.IO.Path]::IsPathRooted($Path)) { $Path } else { Join-Path $root $Path }
    $resolved = (Resolve-Path -LiteralPath $candidate).Path
    $rootPrefix = $root.TrimEnd([System.IO.Path]::DirectorySeparatorChar, [System.IO.Path]::AltDirectorySeparatorChar) + [System.IO.Path]::DirectorySeparatorChar
    if (-not $resolved.StartsWith($rootPrefix, [System.StringComparison]::OrdinalIgnoreCase)) {
        throw "Path '$Path' resolves outside the repository."
    }
    if (-not (Test-Path -LiteralPath $resolved -PathType Leaf)) {
        throw "Path '$Path' is not a repository file."
    }
    return $resolved
}

function Get-RepositoryRelativePath([string]$ResolvedPath) {
    return $ResolvedPath.Substring($root.Length + 1).Replace("\\", "/")
}

function Get-FileRecord([string]$Path) {
    $resolved = Resolve-RepositoryFile $Path
    return [ordered]@{
        path = Get-RepositoryRelativePath $resolved
        sha256 = (Get-FileHash -LiteralPath $resolved -Algorithm SHA256).Hash.ToLowerInvariant()
    }
}

function Test-AgentControlPath([string]$RelativePath) {
    $normalized = $RelativePath.Replace("\\", "/")
    foreach ($prefix in $agentControlPrefixes) {
        if ($normalized.StartsWith($prefix, [System.StringComparison]::OrdinalIgnoreCase)) {
            return $true
        }
    }
    return $false
}

$plan = Get-FileRecord $PlanPath
$authority = @(
    foreach ($path in ($AuthorityPaths | Sort-Object -Unique)) {
        Get-FileRecord $path
    }
) | Sort-Object path

$head = (Invoke-GitText @("rev-parse", "HEAD")).Trim()
$branch = (Invoke-GitText @("branch", "--show-current")).Trim()
$trackedDiff = Invoke-GitText @(
    "diff", "--binary", "HEAD", "--", ".",
    ":(exclude).codex/**",
    ":(exclude).agents/**"
)
$untrackedPaths = @(
    ((Invoke-GitText @("ls-files", "--others", "--exclude-standard")) -split "`n") |
        Where-Object { $_ -ne "" -and -not (Test-AgentControlPath $_) } |
        Sort-Object -Unique
)
$untracked = @(
    foreach ($relative in $untrackedPaths) {
        $resolved = Resolve-RepositoryFile $relative
        [ordered]@{
            path = $relative.Replace("\\", "/")
            sha256 = (Get-FileHash -LiteralPath $resolved -Algorithm SHA256).Hash.ToLowerInvariant()
        }
    }
)

$state = [ordered]@{
    head = $head
    branch = $branch
    tracked_diff_sha256 = Get-Sha256Text $trackedDiff
    untracked_files = $untracked
    plan = $plan
    authority = $authority
}
$material = $state | ConvertTo-Json -Depth 8 -Compress
$fingerprint = Get-Sha256Text $material

[ordered]@{
    schema_version = 1
    fingerprint = $fingerprint
    head = $head
    branch = $branch
    tracked_diff_sha256 = $state.tracked_diff_sha256
    untracked_file_count = $untracked.Count
    excluded_agent_control_paths = $agentControlPrefixes
    plan = $plan
    authority = $authority
} | ConvertTo-Json -Depth 8
