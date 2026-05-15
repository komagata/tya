$ErrorActionPreference = "Stop"

$repo = "komagata/tya"
$prefix = if ($env:PREFIX) { $env:PREFIX } else { Join-Path $env:LOCALAPPDATA "Programs\tya" }
$tag = $env:TYA_VERSION

if (-not $tag) {
    $latest = Invoke-RestMethod "https://api.github.com/repos/$repo/releases/latest"
    $tag = $latest.tag_name
}

if (-not $tag) {
    throw "tya install: could not determine latest release tag"
}

$arch = switch ($env:PROCESSOR_ARCHITECTURE) {
    "AMD64" { "amd64" }
    "ARM64" { "arm64" }
    default { throw "tya install: unsupported architecture: $env:PROCESSOR_ARCHITECTURE" }
}

$package = "tya-$tag-windows-$arch"
$url = "https://github.com/$repo/releases/download/$tag/$package.tar.gz"
$tmp = Join-Path ([System.IO.Path]::GetTempPath()) ([System.Guid]::NewGuid().ToString())

New-Item -ItemType Directory -Force -Path $tmp | Out-Null

try {
    $archive = Join-Path $tmp "$package.tar.gz"
    Write-Host "Downloading $url"
    Invoke-WebRequest $url -OutFile $archive
    tar.exe -xzf $archive -C $tmp
    Push-Location (Join-Path $tmp $package)
    try {
        $env:PREFIX = $prefix
        .\install.ps1
    } finally {
        Pop-Location
    }
    & (Join-Path $prefix "bin\tya.exe") version

    $compiler = Get-Command cc.exe -ErrorAction SilentlyContinue
    if (-not $compiler) {
        $compiler = Get-Command clang.exe -ErrorAction SilentlyContinue
    }
    if (-not $compiler) {
        Write-Warning "tya was installed, but native tya run and tya build require a C compiler. Install LLVM/Clang or another cc-compatible C toolchain and add it to PATH."
    }

    if (-not (Get-Command zig.exe -ErrorAction SilentlyContinue)) {
        Write-Host "note: Zig was not found in PATH. WebAssembly targets (wasm32-wasi and wasm32-browser) require Zig."
    }
} finally {
    Remove-Item -Recurse -Force $tmp -ErrorAction SilentlyContinue
}
