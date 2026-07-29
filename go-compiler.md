# Go Compiler Development And Verification Record

## Overview

This note records how the Go compiler executable was introduced, how it is verified, and how it is wired into CI.

Current state:
- `dist/bitproto-go` is buildable from this repository.
- The Go executable currently wraps the existing Python compiler backend.
- Generated `c`, `go`, and `py` outputs have been compared against the Python compiler and validated.
- Cross-language serialization interoperability has been verified with `.bin` files.
- Generated C output is kept C17-oriented in the validation flow, without GNU packed/aligned attributes.
- `float` / `double` schema types are supported with fixed-point transport semantics:
   both map to signed `int32` on wire using scale `1e8` (8 decimal places).
- Native generated C/Go/Python code includes helper APIs to set/get float-like values
   while preserving fixed-point wire compatibility.

## Implementation Summary

### Go compiler executable

Added a new Go module:
- `compiler-go/go.mod`
- `compiler-go/main.go`

Produced binary:
- `dist/bitproto-go`

Build command:
- `make -C compiler build-go-compiler`

### Build integration

Added Makefile targets in [compiler/Makefile](compiler/Makefile):
- `build-go-compiler`
- `verify-go-compiler-c`
- `verify-go-compiler-go`
- `verify-go-compiler-interop`

These are thin wrappers around the verification scripts under `scripts/`, so they can be reused locally and in CI.

## Architecture

The current implementation is a Go executable frontend that invokes the existing Python compiler backend.

Execution flow:
1. Resolve the compiler root directory.
2. Prefer a local virtualenv python if available.
3. Fall back to system `python3`.
4. Invoke `compiler/bitproto/_main.py`.
5. Inject `PYTHONPATH` so backend imports resolve correctly.

Behavior goals:
- Keep CLI usage aligned with the current compiler.
- Preserve relative schema path handling from the caller working directory.

Practical meaning:
- The Go executable is the user-facing compiler entry point.
- Parsing and code generation still come from the Python implementation.
- This gives us a usable bridge while leaving room for a future native Go backend.

## Verification

### C output equivalence and runtime check

Script:
- [scripts/validate_go_compiler_c.sh](scripts/validate_go_compiler_c.sh)

Verified:
1. Generate C output with go-compiler and Python compiler.
2. Diff generated files:
   - `example_bp.h`
   - `example_bp.c`
3. Compile and run the C runtime encode/decode test using both generated outputs.
4. Compare runtime outputs.
5. Repeat in optimization mode with `-O -F "Drone"`.

Command:
- `make -C compiler verify-go-compiler-c`

Result:
- Normal mode `.c/.h` match exactly.
- Optimization mode `.c/.h` match exactly.
- Runtime behavior matches for Go-generated and Python-generated outputs.
- C output validation also checks that the generated files do not emit GNU `packed/aligned` attributes.

### Go output equivalence and runtime check

Script:
- [scripts/validate_go_compiler_go.sh](scripts/validate_go_compiler_go.sh)

Verified:
1. Generate Go output with go-compiler and Python compiler.
2. Diff generated file `example_bp.go`.
3. Build and run temporary Go runtime tests for both outputs.
4. Compare runtime outputs.
5. Repeat in optimization mode with `-O -F "Drone"`.

Command:
- `make -C compiler verify-go-compiler-go`

Result:
- Normal mode `example_bp.go` matches exactly.
- Optimization mode `example_bp.go` matches exactly.
- Runtime behavior matches for Go-generated and Python-generated outputs.

### Cross-language interoperability check

Script:
- [scripts/verify_go_compiler_interop.sh](scripts/verify_go_compiler_interop.sh)

Verified for the same `.bitproto` input:
1. Serialize in Python, C, and Go, producing:
   - `py.bin`
   - `c.bin`
   - `go.bin`
2. Confirm all binary bytes are identical.
3. Cross-deserialize all binaries in each language:
   - Python decodes `py.bin/c.bin/go.bin`
   - C decodes `py.bin/c.bin/go.bin`
   - Go decodes `py.bin/c.bin/go.bin`
4. Assert the decoded fields match the original payload.

Command:
- `make -C compiler verify-go-compiler-interop`

Result:
- The three `.bin` files are identical.
- Each language successfully decodes all three binaries.
- Decoded field values match the original values.

Coverage notes:
- The interoperability check covers scalar, signed, enum, bool, nested, and array fields.
- The test uses the same `Drone` payload in all three languages, so byte-level equivalence is meaningful.

Fixed-point float notes:
- `float` and `double` are intentionally stored as `int32` for deterministic wire format.
- Conversion is `round(v * 1e8)` on write, `stored / 1e8` on read, with int32 saturation.
- Effective value range is about `[-21.47483648, 21.47483647]`.

## CI Integration

Workflow:
- [.github/workflows/ci.yml](.github/workflows/ci.yml)

Added CI step:
- Verify go-compiler outputs and interop

Gating condition:
- `matrix.python == '3.11'`

Commands run in CI:
- `make -C compiler verify-go-compiler-c --no-print-directory -s`
- `make -C compiler verify-go-compiler-go --no-print-directory -s`
- `make -C compiler verify-go-compiler-interop --no-print-directory -s`

Why only one matrix axis:
- These checks are relatively heavy.
- Running them once keeps the CI signal strong without multiplying runtime across the whole Python matrix.

## Reproduction

From the repository root:

1. Build the compiler binary:
- `make -C compiler build-go-compiler`

2. Verify C generation and runtime behavior:
- `make -C compiler verify-go-compiler-c`

3. Verify Go generation and runtime behavior:
- `make -C compiler verify-go-compiler-go`

4. Verify cross-language interoperability:
- `make -C compiler verify-go-compiler-interop`

5. Run all go-compiler checks in sequence:
- `make -C compiler verify-go-compiler-c --no-print-directory -s && make -C compiler verify-go-compiler-go --no-print-directory -s && make -C compiler verify-go-compiler-interop --no-print-directory -s`

Useful debugging artifacts:
- `dist/bitproto-go`
- `scripts/validate_go_compiler_c.sh`
- `scripts/validate_go_compiler_go.sh`
- `scripts/verify_go_compiler_interop.sh`

## Environment Notes

Backend Python requirements:
- `compiler/requirements.txt`

Expected Python modules:
- `ply`
- `typing_extensions`

The scripts work with either:
- `compiler/.venv/bin/python`
- system `python3`

If the required Python modules are missing, install them into the interpreter selected by the script.

## Known Warnings

C compilation may show pre-existing warnings from `lib/c/bitproto.c` about unused parameters.
These warnings do not affect pass/fail results.

## Scope And Limitations

What is guaranteed now:
- Generated outputs are byte-compatible with Python compiler outputs for the covered flows.
- Generated `py/c/go` code is runtime-validated.
- Cross-language data interchange is verified end-to-end.

What is not yet guaranteed:
- A pure-Go parser and renderer implementation.
- Full replacement of Python compiler internals.
- Schema features or target behaviors that are not exercised by the current regression suite.

## Next Phase

If we want a fully native Go backend:
1. Implement lexer/parser/AST in Go.
2. Reproduce renderer behavior in Go for `c/go/py` targets.
3. Keep the current equivalence and interoperability scripts as regression gates.
4. Switch the backend from Python invocation to native Go behind a feature flag.

## Verified In This Workspace

The following checks completed successfully with exit code `0`:

- `make -C compiler verify-go-compiler-c --no-print-directory -s`
- `make -C compiler verify-go-compiler-go --no-print-directory -s`
- `make -C compiler verify-go-compiler-interop --no-print-directory -s`
