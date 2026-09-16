[CmdletBinding()]
param(
    [ValidateSet('generate', 'generate-runtime', 'check')]
    [string]$Command = 'check'
)

$ErrorActionPreference = 'Stop'
$Root = (Resolve-Path (Join-Path $PSScriptRoot '..')).Path
$TailwindVersion = 'v4.3.3'
$TailwindWindowsSha256 = 'e0e260ce048014e9268f6237ff18f8ccf02cef521cbd0ae04e82c2cdf7aa3955'
$CacheRoot = if ($env:TOCKR_PLATFORM_FRONTEND_TOOL_CACHE) { $env:TOCKR_PLATFORM_FRONTEND_TOOL_CACHE } else { Join-Path $env:LOCALAPPDATA 'tockrplatform\frontend-tools' }

function Get-TailwindPath {
    New-Item -ItemType Directory -Force -Path $CacheRoot | Out-Null
    $tool = Join-Path $CacheRoot "$TailwindVersion-tailwindcss-windows-x64.exe"
    if (-not (Test-Path -LiteralPath $tool -PathType Leaf)) {
        $temporary = "$tool.download"
        Remove-Item -LiteralPath $temporary -Force -ErrorAction SilentlyContinue
        Invoke-WebRequest -Uri "https://github.com/tailwindlabs/tailwindcss/releases/download/$TailwindVersion/tailwindcss-windows-x64.exe" -OutFile $temporary
        Move-Item -LiteralPath $temporary -Destination $tool -Force
    }
    $actual = (Get-FileHash -LiteralPath $tool -Algorithm SHA256).Hash.ToLowerInvariant()
    if ($actual -ne $TailwindWindowsSha256) { throw "Tailwind checksum mismatch: expected $TailwindWindowsSha256, got $actual" }
    return $tool
}

function Invoke-Tailwind([string]$OutputPath) {
    New-Item -ItemType Directory -Force -Path (Split-Path -Parent $OutputPath) | Out-Null
    $tailwind = Get-TailwindPath
    & $tailwind -i web/static/tailwind.css -o $OutputPath --minify
    if ($LASTEXITCODE -ne 0 -or (Get-Item -LiteralPath $OutputPath).Length -eq 0) { throw 'Tailwind did not produce a non-empty stylesheet' }
}

Push-Location $Root
try {
    if ($Command -eq 'check') {
        & go tool templ generate -check -include-version=false
        if ($LASTEXITCODE -ne 0) { throw 'templ check failed' }
        $output = Join-Path $Root ".presentation-output-$PID\presentation.css"
        try { Invoke-Tailwind $output } finally { Remove-Item -LiteralPath (Split-Path -Parent $output) -Recurse -Force -ErrorAction SilentlyContinue }
    } else {
        & go tool templ generate -include-version=false
        if ($LASTEXITCODE -ne 0) { throw 'templ generation failed' }
        $output = Join-Path $Root ".presentation-runtime-$PID\presentation.css"
        try {
            Invoke-Tailwind $output
            if ($Command -eq 'generate-runtime') { Move-Item -LiteralPath $output -Destination (Join-Path $Root 'web/static/presentation.css') -Force }
        } finally { Remove-Item -LiteralPath (Split-Path -Parent $output) -Recurse -Force -ErrorAction SilentlyContinue }
    }
}
finally { Pop-Location }
