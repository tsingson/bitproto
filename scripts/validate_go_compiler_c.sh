#!/usr/bin/env bash

set -euo pipefail

ROOT_DIR="$(cd "$(dirname "$0")/.." && pwd)"
GO_COMPILER_BIN="$ROOT_DIR/dist/bitproto-go"
SCHEMA_FILE="$ROOT_DIR/example/example.bitproto"
MAIN_C_FILE="$ROOT_DIR/example/C/main.c"
MAIN_C_OPT_FILE="$ROOT_DIR/example/C-optimization-mode/main.c"
LIB_C_FILE="$ROOT_DIR/lib/c/bitproto.c"
LIB_INCLUDE_DIR="$ROOT_DIR/lib/c"

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

TMP_DIR="$(mktemp -d "${TMPDIR:-/tmp}/bitproto-verify-XXXXXX")"
cleanup() {
    rm -rf "$TMP_DIR"
}
trap cleanup EXIT

GO_OUT_DIR="$TMP_DIR/out-go"
PY_OUT_DIR="$TMP_DIR/out-py"
GO_OPT_OUT_DIR="$TMP_DIR/out-go-opt"
PY_OPT_OUT_DIR="$TMP_DIR/out-py-opt"
mkdir -p "$GO_OUT_DIR" "$PY_OUT_DIR" "$GO_OPT_OUT_DIR" "$PY_OPT_OUT_DIR"

echo "[step] generate C output with Go compiler"
"$GO_COMPILER_BIN" c "$SCHEMA_FILE" "$GO_OUT_DIR"

echo "[step] generate C output with Python compiler"
PYTHONPATH="$ROOT_DIR/compiler" "$PYTHON_BIN" -m bitproto._main c "$SCHEMA_FILE" "$PY_OUT_DIR"

for f in example_bp.h example_bp.c; do
    if ! diff -u "$PY_OUT_DIR/$f" "$GO_OUT_DIR/$f" >/dev/null; then
        echo "[error] generated file mismatch: $f"
        diff -u "$PY_OUT_DIR/$f" "$GO_OUT_DIR/$f" || true
        exit 1
    fi
done
echo "[ok] generated .c/.h are identical"

for f in example_bp.h example_bp.c; do
    if grep -q '__attribute__((packed' "$GO_OUT_DIR/$f" || grep -q '__attribute__((packed' "$PY_OUT_DIR/$f"; then
        echo "[error] generated C output still contains GNU packed/aligned attribute: $f"
        exit 1
    fi
done
echo "[ok] generated C output is C17-oriented (no GNU packed/aligned attributes)"

echo "[step] generate optimization-mode C output with Go compiler"
"$GO_COMPILER_BIN" c "$SCHEMA_FILE" "$GO_OPT_OUT_DIR" -O -F "Drone"

echo "[step] generate optimization-mode C output with Python compiler"
PYTHONPATH="$ROOT_DIR/compiler" "$PYTHON_BIN" -m bitproto._main c "$SCHEMA_FILE" "$PY_OPT_OUT_DIR" -O -F "Drone"

for f in example_bp.h example_bp.c; do
    if ! diff -u "$PY_OPT_OUT_DIR/$f" "$GO_OPT_OUT_DIR/$f" >/dev/null; then
        echo "[error] generated optimization-mode file mismatch: $f"
        diff -u "$PY_OPT_OUT_DIR/$f" "$GO_OPT_OUT_DIR/$f" || true
        exit 1
    fi
done
echo "[ok] optimization-mode generated .c/.h are identical"

for f in example_bp.h example_bp.c; do
    if grep -q '__attribute__((packed' "$GO_OPT_OUT_DIR/$f" || grep -q '__attribute__((packed' "$PY_OPT_OUT_DIR/$f"; then
        echo "[error] generated optimization-mode C output still contains GNU packed/aligned attribute: $f"
        exit 1
    fi
done
echo "[ok] optimization-mode generated C output is C17-oriented (no GNU packed/aligned attributes)"

echo "[step] compile and run C runtime test (Go-generated output)"
cc -std=c99 -Wall -Wextra -O2 \
    "$MAIN_C_FILE" "$GO_OUT_DIR/example_bp.c" "$LIB_C_FILE" \
    -I"$GO_OUT_DIR" -I"$LIB_INCLUDE_DIR" \
    -o "$TMP_DIR/example-go"
"$TMP_DIR/example-go" >"$TMP_DIR/out-go.txt"

echo "[step] compile and run C runtime test (Python-generated output)"
cc -std=c99 -Wall -Wextra -O2 \
    "$MAIN_C_FILE" "$PY_OUT_DIR/example_bp.c" "$LIB_C_FILE" \
    -I"$PY_OUT_DIR" -I"$LIB_INCLUDE_DIR" \
    -o "$TMP_DIR/example-py"
"$TMP_DIR/example-py" >"$TMP_DIR/out-py.txt"

if ! diff -u "$TMP_DIR/out-py.txt" "$TMP_DIR/out-go.txt" >/dev/null; then
    echo "[error] runtime output mismatch between Go-generated and Python-generated C code"
    diff -u "$TMP_DIR/out-py.txt" "$TMP_DIR/out-go.txt" || true
    exit 1
fi

echo "[ok] C runtime serialization/deserialization behavior matches"

echo "[step] compile and run optimization-mode C runtime test (Go-generated output)"
cc -std=c99 -Wall -Wextra -O2 \
    "$MAIN_C_OPT_FILE" "$GO_OPT_OUT_DIR/example_bp.c" "$LIB_C_FILE" \
    -I"$GO_OPT_OUT_DIR" -I"$LIB_INCLUDE_DIR" \
    -o "$TMP_DIR/example-go-opt"
"$TMP_DIR/example-go-opt" >"$TMP_DIR/out-go-opt.txt"

echo "[step] compile and run optimization-mode C runtime test (Python-generated output)"
cc -std=c99 -Wall -Wextra -O2 \
    "$MAIN_C_OPT_FILE" "$PY_OPT_OUT_DIR/example_bp.c" "$LIB_C_FILE" \
    -I"$PY_OPT_OUT_DIR" -I"$LIB_INCLUDE_DIR" \
    -o "$TMP_DIR/example-py-opt"
"$TMP_DIR/example-py-opt" >"$TMP_DIR/out-py-opt.txt"

if ! diff -u "$TMP_DIR/out-py-opt.txt" "$TMP_DIR/out-go-opt.txt" >/dev/null; then
    echo "[error] optimization-mode runtime output mismatch between Go-generated and Python-generated C code"
    diff -u "$TMP_DIR/out-py-opt.txt" "$TMP_DIR/out-go-opt.txt" || true
    exit 1
fi

echo "[ok] optimization-mode C runtime behavior matches"
echo "[done] validation passed"
