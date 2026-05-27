#!/usr/bin/env sh
set -eu

version="${1:-}"
if [ -z "$version" ]; then
  version="$(go run ./cmd/tya version)"
fi

case "$version" in
  v*) tag="$version"; version="${version#v}" ;;
  *) tag="v$version" ;;
esac

dist_dir="${DIST_DIR:-dist}"
rm -rf "$dist_dir"
mkdir -p "$dist_dir"

build_package() {
  goos="$1"
  goarch="$2"
  ext="$3"
  package="tya-$tag-$goos-$goarch"
  root="$dist_dir/$package"

  mkdir -p "$root/bin" "$root/share/tya/runtime" "$root/share/tya/lib"
  GOOS="$goos" GOARCH="$goarch" go build -o "$root/bin/tya$ext" ./cmd/tya
  cp runtime/tya_runtime.c runtime/tya_runtime.h "$root/share/tya/runtime/"
  if [ -f runtime/tya_cover.c ]; then
    cp runtime/tya_cover.c "$root/share/tya/runtime/"
  fi
  if [ -f runtime/tya_http_server.c ]; then
    cp runtime/tya_http_server.c runtime/tya_http_server.h "$root/share/tya/runtime/"
  fi
  cp -R lib/. "$root/share/tya/lib/"
  cp README.md "$root/"

  cat > "$root/install.sh" <<'EOF'
#!/usr/bin/env sh
set -eu

prefix="${PREFIX:-$HOME/.local}"
zig_version="${TYA_ZIG_VERSION:-0.16.0}"
zig_bin="$prefix/zig/$zig_version/zig"

mkdir -p "$prefix/bin" "$prefix/share/tya/runtime" "$prefix/share/tya/lib"
cp bin/tya "$prefix/bin/tya"
cp share/tya/runtime/* "$prefix/share/tya/runtime/"
cp -R share/tya/lib/. "$prefix/share/tya/lib/"

install_zig() {
  for cmd in curl tar mktemp; do
    if ! command -v "$cmd" >/dev/null 2>&1; then
      echo "tya install: missing required command: $cmd" >&2
      exit 1
    fi
  done

  os="$(uname -s)"
  arch="$(uname -m)"
  case "$os" in
    Darwin) zig_os="macos" ;;
    Linux) zig_os="linux" ;;
    *) echo "unsupported OS for managed Zig: $os" >&2; exit 1 ;;
  esac
  case "$arch" in
    x86_64|amd64) zig_arch="x86_64" ;;
    arm64|aarch64) zig_arch="aarch64" ;;
    *) echo "unsupported architecture for managed Zig: $arch" >&2; exit 1 ;;
  esac

  tmp="$(mktemp -d)"
  trap 'rm -rf "$tmp"' EXIT HUP INT TERM
  zig_name="zig-$zig_arch-$zig_os-$zig_version"
  zig_url="https://ziglang.org/download/$zig_version/$zig_name.tar.xz"
  curl -fsSL "$zig_url" -o "$tmp/zig.tar.xz"
  tar -C "$tmp" -xJf "$tmp/zig.tar.xz"
  rm -rf "$prefix/zig/$zig_version"
  mkdir -p "$prefix/zig"
  mv "$tmp/$zig_name" "$prefix/zig/$zig_version"
}

if [ ! -x "$zig_bin" ] || [ "$("$zig_bin" version 2>/dev/null || true)" != "$zig_version" ]; then
  install_zig
  echo "Managed Zig installed: $zig_bin"
else
  echo "Managed Zig already installed: $zig_bin"
fi

echo "installed tya to $prefix/bin/tya"
echo "Add $prefix/bin to PATH if it is not already there."
EOF
  chmod +x "$root/install.sh"

  cat > "$root/install.ps1" <<'EOF'
$ErrorActionPreference = "Stop"
$prefix = if ($env:PREFIX) { $env:PREFIX } else { Join-Path $env:LOCALAPPDATA "Programs\tya" }
$zigVersion = if ($env:TYA_ZIG_VERSION) { $env:TYA_ZIG_VERSION } else { "0.16.0" }
$zigBin = Join-Path $prefix "zig\$zigVersion\zig.exe"

New-Item -ItemType Directory -Force -Path (Join-Path $prefix "bin") | Out-Null
New-Item -ItemType Directory -Force -Path (Join-Path $prefix "share\tya\runtime") | Out-Null
New-Item -ItemType Directory -Force -Path (Join-Path $prefix "share\tya\lib") | Out-Null
Copy-Item "bin\tya.exe" (Join-Path $prefix "bin\tya.exe") -Force
Copy-Item "share\tya\runtime\*" (Join-Path $prefix "share\tya\runtime") -Force
Copy-Item "share\tya\lib\*" (Join-Path $prefix "share\tya\lib") -Recurse -Force

function Install-ManagedZig {
  $arch = if ([System.Runtime.InteropServices.RuntimeInformation]::OSArchitecture -eq [System.Runtime.InteropServices.Architecture]::Arm64) { "aarch64" } else { "x86_64" }
  $zigName = "zig-$arch-windows-$zigVersion"
  $zigUrl = "https://ziglang.org/download/$zigVersion/$zigName.zip"
  $tmp = Join-Path ([System.IO.Path]::GetTempPath()) ("tya-zig-" + [Guid]::NewGuid().ToString("N"))
  New-Item -ItemType Directory -Force -Path $tmp | Out-Null
  try {
    $zip = Join-Path $tmp "zig.zip"
    Invoke-WebRequest -Uri $zigUrl -OutFile $zip
    Expand-Archive -Path $zip -DestinationPath $tmp -Force
    $zigRoot = Join-Path $prefix "zig"
    $zigDest = Join-Path $zigRoot $zigVersion
    New-Item -ItemType Directory -Force -Path $zigRoot | Out-Null
    if (Test-Path $zigDest) { Remove-Item $zigDest -Recurse -Force }
    Move-Item (Join-Path $tmp $zigName) $zigDest
  } finally {
    if (Test-Path $tmp) { Remove-Item $tmp -Recurse -Force }
  }
}

$hasMatchingZig = $false
if (Test-Path $zigBin) {
  try {
    $hasMatchingZig = ((& $zigBin version).Trim() -eq $zigVersion)
  } catch {
    $hasMatchingZig = $false
  }
}

if (-not $hasMatchingZig) {
  Install-ManagedZig
  Write-Host "Managed Zig installed: $zigBin"
} else {
  Write-Host "Managed Zig already installed: $zigBin"
}

Write-Host "installed tya to $(Join-Path $prefix 'bin\tya.exe')"
Write-Host "Add $(Join-Path $prefix 'bin') to PATH if it is not already there."
EOF

  tar -C "$dist_dir" -czf "$dist_dir/$package.tar.gz" "$package"
  shasum -a 256 "$dist_dir/$package.tar.gz" > "$dist_dir/$package.tar.gz.sha256"
}

build_package darwin amd64 ""
build_package darwin arm64 ""
build_package linux amd64 ""
build_package linux arm64 ""
build_package windows amd64 ".exe"
build_package windows arm64 ".exe"

ls -1 "$dist_dir"/*.tar.gz
