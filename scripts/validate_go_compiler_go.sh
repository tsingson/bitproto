#!/usr/bin/env bash

set -euo pipefail

ROOT_DIR="$(cd "$(dirname "$0")/.." && pwd)"
GO_COMPILER_BIN="$ROOT_DIR/dist/bitproto-go"
SCHEMA_FILE="$ROOT_DIR/example/example.bitproto"

if [[ -x "$ROOT_DIR/compiler/.venv/bin/python" ]]; then
    PYTHON_BIN="$ROOT_DIR/compiler/.venv/bin/python"
elif command -v python3 >/dev/null 2>&1; then
    PYTHON_BIN="$(command -v python3)"
else
    echo "[error] python3 not found"
    exit 1
fi

if [[ ! -x "$GO_COMPILER_BIN" ]]; then
    echo "[info] building Go compiler binary..."
    make -C "$ROOT_DIR/compiler" build-go-compiler
fi

if ! "$PYTHON_BIN" -c "import ply, typing_extensions" >/dev/null 2>&1; then
    echo "[error] required python modules missing (ply, typing_extensions)"
    echo "[hint] install with: $PYTHON_BIN -m pip install -r $ROOT_DIR/compiler/requirements.txt"
    exit 1
fi

TMP_DIR="$(mktemp -d "${TMPDIR:-/tmp}/bitproto-verify-go-XXXXXX")"
cleanup() {
    rm -rf "$TMP_DIR"
}
trap cleanup EXIT

GO_OUT_DIR="$TMP_DIR/out-go"
PY_OUT_DIR="$TMP_DIR/out-py"
GO_OPT_OUT_DIR="$TMP_DIR/out-go-opt"
PY_OPT_OUT_DIR="$TMP_DIR/out-py-opt"
mkdir -p "$GO_OUT_DIR" "$PY_OUT_DIR" "$GO_OPT_OUT_DIR" "$PY_OPT_OUT_DIR"

echo "[step] generate Go output with Go compiler"
"$GO_COMPILER_BIN" go "$SCHEMA_FILE" "$GO_OUT_DIR"

echo "[step] generate Go output with Python compiler"
PYTHONPATH="$ROOT_DIR/compiler" "$PYTHON_BIN" -m bitproto._main go "$SCHEMA_FILE" "$PY_OUT_DIR"

if ! diff -u "$PY_OUT_DIR/example_bp.go" "$GO_OUT_DIR/example_bp.go" >/dev/null; then
    echo "[error] generated Go file mismatch: example_bp.go"
    diff -u "$PY_OUT_DIR/example_bp.go" "$GO_OUT_DIR/example_bp.go" || true
    exit 1
fi
echo "[ok] generated Go file is identical"

echo "[step] generate optimization-mode Go output with Go compiler"
"$GO_COMPILER_BIN" go "$SCHEMA_FILE" "$GO_OPT_OUT_DIR" -O -F "Drone"

echo "[step] generate optimization-mode Go output with Python compiler"
PYTHONPATH="$ROOT_DIR/compiler" "$PYTHON_BIN" -m bitproto._main go "$SCHEMA_FILE" "$PY_OPT_OUT_DIR" -O -F "Drone"

if ! diff -u "$PY_OPT_OUT_DIR/example_bp.go" "$GO_OPT_OUT_DIR/example_bp.go" >/dev/null; then
    echo "[error] optimization-mode generated Go file mismatch: example_bp.go"
    diff -u "$PY_OPT_OUT_DIR/example_bp.go" "$GO_OPT_OUT_DIR/example_bp.go" || true
    exit 1
fi
echo "[ok] optimization-mode generated Go file is identical"

make_runtime_case() {
    local case_dir="$1"
    local generated_file="$2"

    mkdir -p "$case_dir/gen-bp"
    cp "$generated_file" "$case_dir/gen-bp/example_bp.go"

    cat >"$case_dir/go.mod" <<'EOF'
module bitproto_verify

go 1.22

require github.com/hit9/bitproto/lib/go v0.0.0-00010101000000-000000000000

replace github.com/hit9/bitproto/lib/go => REPLACE_ROOT/lib/go
EOF

    cat >"$case_dir/main.go" <<'EOF'
package main

import (
    "fmt"

    bp "bitproto_verify/gen-bp"
)

func main() {
    drone := &bp.Drone{}
    drone.Status = bp.DRONE_STATUS_RISING
    drone.Position.Longitude = 2000
    drone.Position.Latitude = 2000
    drone.Position.Altitude = 1080
    drone.Flight.Acceleration[0] = -1001
    drone.Power.IsCharging = true
    drone.PressureSensor.Pressures[0] = -11
    s := drone.Encode()

    droneNew := &bp.Drone{}
    droneNew.Decode(s)

    if droneNew.Status != drone.Status {
        panic("status mismatch after decode")
    }
    if droneNew.Position.Longitude != drone.Position.Longitude {
        panic("longitude mismatch after decode")
    }
    if droneNew.Flight.Acceleration[0] != drone.Flight.Acceleration[0] {
        panic("acceleration mismatch after decode")
    }
    if droneNew.PressureSensor.Pressures[0] != drone.PressureSensor.Pressures[0] {
        panic("pressure mismatch after decode")
    }

    fmt.Printf("%v", droneNew)
}
EOF

    sed -i.bak "s#REPLACE_ROOT#$ROOT_DIR#g" "$case_dir/go.mod"
    rm -f "$case_dir/go.mod.bak"
}

echo "[step] runtime test for normal-mode generated Go code"
make_runtime_case "$TMP_DIR/runtime-go" "$GO_OUT_DIR/example_bp.go"
make_runtime_case "$TMP_DIR/runtime-py" "$PY_OUT_DIR/example_bp.go"
(cd "$TMP_DIR/runtime-go" && go run . >"$TMP_DIR/out-go.txt")
(cd "$TMP_DIR/runtime-py" && go run . >"$TMP_DIR/out-py.txt")
if ! diff -u "$TMP_DIR/out-py.txt" "$TMP_DIR/out-go.txt" >/dev/null; then
    echo "[error] normal-mode runtime output mismatch"
    diff -u "$TMP_DIR/out-py.txt" "$TMP_DIR/out-go.txt" || true
    exit 1
fi
echo "[ok] normal-mode runtime behavior matches"

echo "[step] runtime test for optimization-mode generated Go code"
make_runtime_case "$TMP_DIR/runtime-go-opt" "$GO_OPT_OUT_DIR/example_bp.go"
make_runtime_case "$TMP_DIR/runtime-py-opt" "$PY_OPT_OUT_DIR/example_bp.go"
(cd "$TMP_DIR/runtime-go-opt" && go run . >"$TMP_DIR/out-go-opt.txt")
(cd "$TMP_DIR/runtime-py-opt" && go run . >"$TMP_DIR/out-py-opt.txt")
if ! diff -u "$TMP_DIR/out-py-opt.txt" "$TMP_DIR/out-go-opt.txt" >/dev/null; then
    echo "[error] optimization-mode runtime output mismatch"
    diff -u "$TMP_DIR/out-py-opt.txt" "$TMP_DIR/out-go-opt.txt" || true
    exit 1
fi
echo "[ok] optimization-mode runtime behavior matches"

echo "[done] Go generation validation passed"
