# Float/Double Fixed-Point Demo

This demo shows how `float` and `double` are encoded as signed `int32` fixed-point values (scale `1e8`) and verifies byte-level equality across generated native C/Go/Python code.

## Files

- `sensor.bitproto`: demo schema
- `run_demo.py`: one-shot script to generate native code, run C/Go/Python encoders, and compare payload bytes

## Run

From repository root:

```bash
python example/float-fixed-point/run_demo.py
```

Expected output includes:

- `bytes equal across C/Go/Python`
- `PASS`

## What It Verifies

1. `float/double` helper APIs are available and usable in all three generated targets.
2. Fixed-point conversion (`round(v * 1e8)` and divide by `1e8`) is consistent.
3. Wire payload bytes are identical across C, Go, and Python.

## Demo Values

- `temperature = 12.34567891`
- `humidity = -3.21098765`
- `history = [0.1, -0.2, 21.47483647]`
