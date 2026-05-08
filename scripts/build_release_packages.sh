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

  mkdir -p "$root/bin" "$root/share/tya/runtime" "$root/share/tya/stdlib"
  GOOS="$goos" GOARCH="$goarch" go build -o "$root/bin/tya$ext" ./cmd/tya
  cp runtime/tya_runtime.c runtime/tya_runtime.h "$root/share/tya/runtime/"
  cp stdlib/*.tya "$root/share/tya/stdlib/"
  cp README.md "$root/"

  cat > "$root/install.sh" <<'EOF'
#!/usr/bin/env sh
set -eu

prefix="${PREFIX:-$HOME/.local}"
mkdir -p "$prefix/bin" "$prefix/share/tya/runtime" "$prefix/share/tya/stdlib"
cp bin/tya "$prefix/bin/tya"
cp share/tya/runtime/* "$prefix/share/tya/runtime/"
cp share/tya/stdlib/* "$prefix/share/tya/stdlib/"
echo "installed tya to $prefix/bin/tya"
echo "Add $prefix/bin to PATH if it is not already there."
EOF
  chmod +x "$root/install.sh"

  cat > "$root/install.ps1" <<'EOF'
$ErrorActionPreference = "Stop"
$prefix = if ($env:PREFIX) { $env:PREFIX } else { Join-Path $env:LOCALAPPDATA "Programs\tya" }
New-Item -ItemType Directory -Force -Path (Join-Path $prefix "bin") | Out-Null
New-Item -ItemType Directory -Force -Path (Join-Path $prefix "share\tya\runtime") | Out-Null
New-Item -ItemType Directory -Force -Path (Join-Path $prefix "share\tya\stdlib") | Out-Null
Copy-Item "bin\tya.exe" (Join-Path $prefix "bin\tya.exe") -Force
Copy-Item "share\tya\runtime\*" (Join-Path $prefix "share\tya\runtime") -Force
Copy-Item "share\tya\stdlib\*" (Join-Path $prefix "share\tya\stdlib") -Force
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

ls -1 "$dist_dir"/*.tar.gz
