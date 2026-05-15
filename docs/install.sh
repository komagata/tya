#!/usr/bin/env sh
set -eu

repo="komagata/tya"
prefix="${PREFIX:-$HOME/.local}"

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

cat <<EOF

Tya binary installed:
  $prefix/bin/tya
EOF
"$prefix/bin/tya" version

if ! command -v cc >/dev/null 2>&1; then
  cat >&2 <<'EOF'

Requirement missing:
  cc

Native `tya run` and `tya build` require a C compiler.

macOS:   install Xcode Command Line Tools with `xcode-select --install`
Linux:   install your distribution's build-essential/clang package
EOF
else
  echo "Native build requirement found: cc"
fi

if ! command -v zig >/dev/null 2>&1; then
  cat >&2 <<'EOF'

Optional requirement missing:
  zig

WebAssembly targets (`wasm32-wasi` and `wasm32-browser`) require Zig.
EOF
else
  echo "WebAssembly build requirement found: zig"
fi
