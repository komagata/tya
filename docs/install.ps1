$ErrorActionPreference = "Stop"

$repo = "komagata/tya"
$prefix = if ($env:PREFIX) { $env:PREFIX } else { Join-Path $env:LOCALAPPDATA "Programs\tya" }
$tag = $env:TYA_VERSION
$zigVersion = if ($env:TYA_ZIG_VERSION) { $env:TYA_ZIG_VERSION } else { "0.16.0" }

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

    $zigArch = switch ($arch) {
        "amd64" { "x86_64" }
        "arm64" { "aarch64" }
        default { throw "tya install: unsupported Zig architecture: $arch" }
    }
    $zigPackage = "zig-$zigArch-windows-$zigVersion"
    $zigUrl = "https://ziglang.org/download/$zigVersion/$zigPackage.zip"
    $zigDir = Join-Path (Join-Path $prefix "zig") $zigVersion
    $zigExe = Join-Path $zigDir "zig.exe"
    $hasMatchingZig = $false
    if (Test-Path $zigExe) {
        try {
            $hasMatchingZig = ((& $zigExe version).Trim() -eq $zigVersion)
        } catch {
            $hasMatchingZig = $false
        }
    }
    if ($hasMatchingZig) {
        Write-Host "Managed Zig already installed: $zigExe"
    } else {
        $zigArchive = Join-Path $tmp "$zigPackage.zip"
        Write-Host "Downloading $zigUrl"
        Invoke-WebRequest $zigUrl -OutFile $zigArchive
        Remove-Item -Recurse -Force $zigDir -ErrorAction SilentlyContinue
        Expand-Archive $zigArchive -DestinationPath $tmp -Force
        New-Item -ItemType Directory -Force -Path (Split-Path $zigDir) | Out-Null
        Move-Item (Join-Path $tmp $zigPackage) $zigDir
    }

    Write-Host ""
    Write-Host "Tya binary installed:"
    Write-Host "  $(Join-Path $prefix 'bin\tya.exe')"
    Write-Host "Managed Zig installed:"
    Write-Host "  $zigExe"
    & (Join-Path $prefix "bin\tya.exe") version
    & $zigExe version
} finally {
    Remove-Item -Recurse -Force $tmp -ErrorAction SilentlyContinue
}
