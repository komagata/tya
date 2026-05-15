#!/usr/bin/env sh
set -eu

repo="komagata/tya"
prefix="${PREFIX:-$HOME/.local}"
zig_version="${TYA_ZIG_VERSION:-0.16.0}"

need() {
  if ! command -v "$1" >/dev/null 2>&1; then
    echo "tya install: missing required command: $1" >&2
    exit 1
  fi
}

need curl
need tar
need mktemp

os="$(uname -s | tr '[:upper:]' '[:lower:]')"
arch="$(uname -m)"

case "$os" in
  darwin) os="darwin" ;;
  linux) os="linux" ;;
  *)
    echo "tya install: unsupported OS: $os" >&2
    exit 1
    ;;
esac

case "$arch" in
  x86_64 | amd64) arch="amd64" ;;
  arm64 | aarch64) arch="arm64" ;;
  *)
    echo "tya install: unsupported architecture: $arch" >&2
    exit 1
    ;;
esac

tag="${TYA_VERSION:-}"
if [ -z "$tag" ]; then
  tag="$(curl -fsSL "https://api.github.com/repos/$repo/releases/latest" | sed -n 's/.*"tag_name":[[:space:]]*"\([^"]*\)".*/\1/p' | head -n 1)"
fi

if [ -z "$tag" ]; then
  echo "tya install: could not determine latest release tag" >&2
  exit 1
fi

package="tya-$tag-$os-$arch"
url="https://github.com/$repo/releases/download/$tag/$package.tar.gz"
tmp="$(mktemp -d)"

cleanup() {
  rm -rf "$tmp"
}
trap cleanup EXIT INT TERM

echo "Downloading $url"
curl -fsSL "$url" -o "$tmp/$package.tar.gz"
tar -xzf "$tmp/$package.tar.gz" -C "$tmp"

(
  cd "$tmp/$package"
  PREFIX="$prefix" sh ./install.sh
)

zig_os="$os"
if [ "$zig_os" = "darwin" ]; then
  zig_os="macos"
fi
zig_arch="$arch"
if [ "$zig_arch" = "amd64" ]; then
  zig_arch="x86_64"
else
  zig_arch="aarch64"
fi
zig_package="zig-$zig_arch-$zig_os-$zig_version"
zig_url="https://ziglang.org/download/$zig_version/$zig_package.tar.xz"
zig_dir="$prefix/zig/$zig_version"
zig_bin="$zig_dir/zig"

if [ -x "$zig_bin" ] && [ "$("$zig_bin" version 2>/dev/null || true)" = "$zig_version" ]; then
  echo "Managed Zig already installed: $zig_bin"
else
  echo "Downloading $zig_url"
  rm -rf "$tmp/$zig_package" "$zig_dir"
  curl -fsSL "$zig_url" -o "$tmp/$zig_package.tar.xz"
  tar -xJf "$tmp/$zig_package.tar.xz" -C "$tmp"
  mkdir -p "$(dirname "$zig_dir")"
  mv "$tmp/$zig_package" "$zig_dir"
fi

cat <<EOF

Tya binary installed:
  $prefix/bin/tya
Managed Zig installed:
  $zig_bin
EOF
"$prefix/bin/tya" version
"$zig_bin" version
