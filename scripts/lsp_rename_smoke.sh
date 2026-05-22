#!/usr/bin/env bash
set -euo pipefail

if [ "$#" -lt 2 ]; then
  echo "usage: scripts/lsp_rename_smoke.sh WORKSPACE_ROOT TARGET_FILE" >&2
  exit 2
fi

root=$1
target=$2

if [ ! -d "$root" ]; then
  echo "workspace root not found: $root" >&2
  exit 2
fi
if [ ! -f "$target" ]; then
  echo "target file not found: $target" >&2
  exit 2
fi
if ! command -v python3 >/dev/null 2>&1; then
  echo "python3 is required for the JSON-RPC smoke driver" >&2
  exit 2
fi

tmpdir=$(mktemp -d)
trap 'rm -rf "$tmpdir"' EXIT

tya_bin=${TYA_BIN:-}
if [ -z "$tya_bin" ]; then
  tya_bin="$tmpdir/tya"
  go build -o "$tya_bin" ./cmd/tya
fi
if [ ! -x "$tya_bin" ]; then
  echo "tya binary is not executable: $tya_bin" >&2
  exit 2
fi

python3 - "$tya_bin" "$root" "$target" <<'PY'
import json
import os
import re
import subprocess
import sys

tya_bin, root, target = sys.argv[1:4]
root = os.path.abspath(root)
target = os.path.abspath(target)

with open(target, "r", encoding="utf-8") as f:
    source = f.read()

lines = source.splitlines()


def file_uri(path):
    return "file://" + os.path.abspath(path)


def new_name_for(name, index):
    if name and name[0].isupper():
        return "RenameSmoke" + name + str(index)
    return "rename_smoke_" + name + "_" + str(index)


def add_case(cases, seen, label, line, char, name):
    key = (line, char, name)
    if key in seen:
        return
    seen.add(key)
    cases.append({
        "label": label,
        "line": line,
        "char": char,
        "name": name,
        "newName": new_name_for(name, len(cases) + 1),
    })


def discover_cases():
    cases = []
    seen = set()
    in_func_indent = None
    for i, line in enumerate(lines):
        stripped = line.strip()
        indent = len(line) - len(line.lstrip(" "))

        m = re.match(r"\s*class\s+([A-Za-z_][A-Za-z0-9_]*)\b", line)
        if m:
            add_case(cases, seen, "class declaration", i, m.start(1), m.group(1))

        m = re.match(r"\s*module\s+([A-Za-z_][A-Za-z0-9_]*)\b", line)
        if m:
            add_case(cases, seen, "module declaration", i, m.start(1), m.group(1))

        m = re.match(r"\s*interface\s+([A-Za-z_][A-Za-z0-9_]*)\b", line)
        if m:
            add_case(cases, seen, "interface declaration", i, m.start(1), m.group(1))

        m = re.match(r"\s*(?:private\s+)?(?:static\s+)?(?:abstract\s+|override\s+)?([A-Za-z_][A-Za-z0-9_]*)\s*=", line)
        if m and indent > 0:
            add_case(cases, seen, "member declaration", i, m.start(1), m.group(1))
            in_func_indent = indent

        m = re.search(r"\bSelf\.([A-Za-z_][A-Za-z0-9_]*)\b", line)
        if m:
            add_case(cases, seen, "Self member reference", i, m.start(1), m.group(1))

        m = re.match(r"\s*([a-z_][A-Za-z0-9_]*)\s*=", line)
        if m and in_func_indent is not None and indent > in_func_indent:
            add_case(cases, seen, "local declaration", i, m.start(1), m.group(1))

        if in_func_indent is not None and stripped and indent <= in_func_indent and not stripped.startswith(("if ", "while ", "for ", "return ")):
            in_func_indent = None
    return cases


def write_message(proc, payload):
    body = json.dumps(payload, separators=(",", ":")).encode("utf-8")
    proc.stdin.write(b"Content-Length: " + str(len(body)).encode("ascii") + b"\r\n\r\n" + body)
    proc.stdin.flush()


def read_message(proc):
    header = b""
    while b"\r\n\r\n" not in header:
        chunk = proc.stdout.read(1)
        if not chunk:
            raise RuntimeError("LSP server closed stdout")
        header += chunk
    length = None
    for line in header.decode("ascii").split("\r\n"):
        if line.lower().startswith("content-length:"):
            length = int(line.split(":", 1)[1].strip())
    if length is None:
        raise RuntimeError("missing Content-Length")
    return json.loads(proc.stdout.read(length))


def request(proc, req_id, method, params):
    write_message(proc, {"jsonrpc": "2.0", "id": req_id, "method": method, "params": params})
    while True:
        msg = read_message(proc)
        if msg.get("id") == req_id:
            return msg


def notify(proc, method, params):
    write_message(proc, {"jsonrpc": "2.0", "method": method, "params": params})


cases = discover_cases()
if not cases:
    print("no rename smoke cases found", file=sys.stderr)
    sys.exit(1)

target_uri = file_uri(target)
proc = subprocess.Popen([tya_bin, "lsp", "--stdio"], stdin=subprocess.PIPE, stdout=subprocess.PIPE)
try:
    request(proc, 1, "initialize", {
        "processId": None,
        "rootUri": file_uri(root),
        "capabilities": {},
    })
    notify(proc, "initialized", {})
    notify(proc, "textDocument/didOpen", {
        "textDocument": {
            "uri": target_uri,
            "languageId": "tya",
            "version": 1,
            "text": source,
        }
    })

    failures = []
    for idx, case in enumerate(cases, start=2):
        pos = {"line": case["line"], "character": case["char"]}
        params = {"textDocument": {"uri": target_uri}, "position": pos}
        prep = request(proc, idx * 2, "textDocument/prepareRename", params)
        if "error" in prep:
            failures.append((case, "prepareRename", prep["error"].get("message", prep["error"])))
            continue
        rename = request(proc, idx * 2 + 1, "textDocument/rename", {
            "textDocument": {"uri": target_uri},
            "position": pos,
            "newName": case["newName"],
        })
        if "error" in rename:
            failures.append((case, "rename", rename["error"].get("message", rename["error"])))
            continue
        changes = rename.get("result", {}).get("changes", {})
        edit_count = sum(len(edits) for edits in changes.values())
        if edit_count == 0:
            failures.append((case, "rename", "returned zero edits"))
            continue
        doc_changes = rename.get("result", {}).get("documentChanges", [])
        expected_file_rename = case["label"] == "class declaration"
        if expected_file_rename:
            old_uri = target_uri
            new_uri = file_uri(os.path.join(os.path.dirname(target), case["newName"] + ".tya"))
            found = any(
                change.get("kind") == "rename"
                and change.get("oldUri") == old_uri
                and change.get("newUri") == new_uri
                for change in doc_changes
                if isinstance(change, dict)
            )
            if not found:
                failures.append((case, "rename", f"missing file rename {old_uri} -> {new_uri}"))
                continue
        suffix = " + file rename" if expected_file_rename else ""
        print(f"ok {case['label']}: {case['name']} at {case['line'] + 1}:{case['char'] + 1} -> {edit_count} edits{suffix}")

    if failures:
        print("", file=sys.stderr)
        for case, method, message in failures:
            print(f"FAIL {method} {case['label']}: {case['name']} at {case['line'] + 1}:{case['char'] + 1}: {message}", file=sys.stderr)
        sys.exit(1)
finally:
    proc.kill()
    proc.wait()
PY
