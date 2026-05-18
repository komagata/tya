#!/bin/sh
set -eu

repo_root=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)

fail() {
  echo "bootstrap_no_go: error: $*" >&2
  exit 1
}

run_step() {
  step=$1
  shift
  echo "bootstrap_no_go: ${step}: $*" >&2
  if ! "$@"; then
    fail "${step} failed: $*"
  fi
}

run_step_sh() {
  step=$1
  shift
  echo "bootstrap_no_go: ${step}: $*" >&2
  if ! sh -c "$*"; then
    fail "${step} failed: $*"
  fi
}

case ${TYA_BOOTSTRAP_TYA+x} in
  x) ;;
  *) fail "TYA_BOOTSTRAP_TYA must point to an executable tya binary" ;;
esac

case $TYA_BOOTSTRAP_TYA in
  /*) bootstrap_tya=$TYA_BOOTSTRAP_TYA ;;
  *) bootstrap_tya=$(command -v "$TYA_BOOTSTRAP_TYA" 2>/dev/null || true) ;;
esac

if [ -z "$bootstrap_tya" ] || [ ! -f "$bootstrap_tya" ] || [ ! -x "$bootstrap_tya" ]; then
  fail "TYA_BOOTSTRAP_TYA is not an executable file: $TYA_BOOTSTRAP_TYA"
fi

work_dir=$(mktemp -d "${TMPDIR:-/tmp}/tya-bootstrap-no-go.XXXXXX") || fail "mktemp failed"
shim_dir="$work_dir/no-go-shim"
mkdir -p "$shim_dir"
cat > "$shim_dir/go" <<'SH'
#!/bin/sh
echo "bootstrap_no_go: no-Go violation: attempted to execute go $*" >&2
exit 127
SH
chmod +x "$shim_dir/go"

cleanup() {
  status=$?
  if [ "$status" -eq 0 ]; then
    rm -rf "$work_dir"
  elif [ "${TYA_KEEP_BOOTSTRAP_TMP:-}" = "1" ]; then
    echo "bootstrap_no_go: retaining work directory: $work_dir" >&2
  else
    rm -rf "$work_dir"
  fi
}
trap cleanup EXIT HUP INT TERM

export PATH="$shim_dir:$PATH"
export TYA_LEGACY_MODULES=1

stage2_c="$work_dir/stage-2.c"
stage2_bin="$work_dir/stage-2"
stage3_c="$work_dir/stage-3.c"
stage3_bin="$work_dir/stage-3"

echo "bootstrap_no_go: bootstrap binary: $bootstrap_tya" >&2
echo "bootstrap_no_go: work directory: $work_dir" >&2

cd "$repo_root"

run_step_sh "stage-2 emit" "\"$bootstrap_tya\" run selfhost/v02/compiler.tya selfhost/v02/compiler.tya > \"$stage2_c\""
run_step "stage-2 compile" cc "$stage2_c" runtime/tya_runtime.c -I runtime -o "$stage2_bin" -lpthread -lm -lz
run_step_sh "stage-3 emit" "\"$stage2_bin\" selfhost/v02/compiler.tya > \"$stage3_c\""
run_step "stage-3 compile" cc "$stage3_c" runtime/tya_runtime.c -I runtime -o "$stage3_bin" -lpthread -lm -lz
run_step "fixed-point compare" diff -u "$stage2_c" "$stage3_c"

echo "bootstrap_no_go: fixed-point compare passed" >&2
