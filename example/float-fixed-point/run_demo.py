#!/usr/bin/env python3
from __future__ import annotations

import subprocess
import sys
import tempfile
from pathlib import Path

ROOT = Path(__file__).resolve().parents[2]
PROTO = Path(__file__).resolve().parent / "sensor.bitproto"

VALUES = {
    "temperature": 12.34567891,
    "humidity": -3.21098765,
    "history": [0.1, -0.2, 21.47483647],
}


def run(cmd: list[str], cwd: Path | None = None) -> bytes:
    return subprocess.check_output(cmd, cwd=str(cwd) if cwd else None)


def parse_bytes(out: bytes) -> list[int]:
    return [int(x) for x in out.decode("utf-8").strip().split() if x]


def generate_native(out_dir: Path) -> None:
    compiler_go = ROOT / "compiler-go"
    run(["go", "run", ".", "native-c", str(PROTO), str(out_dir)], cwd=compiler_go)
    run(["go", "run", ".", "native-go", str(PROTO), str(out_dir)], cwd=compiler_go)
    run(["go", "run", ".", "native-py", str(PROTO), str(out_dir)], cwd=compiler_go)


def run_c(out_dir: Path) -> list[int]:
    main_c = out_dir / "c_demo.c"
    main_c.write_text(
        """#include <assert.h>
#include <math.h>
#include <stdio.h>
#include <string.h>

#include \"telemfp_bp.h\"

int main(void) {
    struct Sensor s = {0};
    SetSensorTemperatureFloat(&s, 12.34567891);
    SetSensorHumidityDouble(&s, -3.21098765);
    SetSensorHistoryFloatAt(&s, 0, 0.1);
    SetSensorHistoryFloatAt(&s, 1, -0.2);
    SetSensorHistoryFloatAt(&s, 2, 21.47483647);

    const double eps = 0.5 / 100000000.0 + 1e-12;
    assert(fabs(GetSensorTemperatureFloat(&s) - 12.34567891) <= eps);
    assert(fabs(GetSensorHumidityDouble(&s) + 3.21098765) <= eps);
    assert(fabs(GetSensorHistoryFloatAt(&s, 2) - 21.47483647) <= eps);

    unsigned char buf[BYTES_LENGTH_SENSOR] = {0};
    EncodeSensor(&s, buf);

    struct Sensor d = {0};
    DecodeSensor(&d, buf);
    assert(fabs(GetSensorTemperatureFloat(&d) - 12.34567891) <= eps);

    for (int i = 0; i < BYTES_LENGTH_SENSOR; i++) {
        printf(\"%u \", buf[i]);
    }
    return 0;
}
""",
        encoding="utf-8",
    )

    c_bin = out_dir / "c_demo"
    run(
        [
            "cc",
            str(main_c),
            str(out_dir / "telemfp_bp.c"),
            str(ROOT / "lib" / "c" / "bitproto.c"),
            "-I",
            str(out_dir),
            "-I",
            str(ROOT / "lib" / "c"),
            "-o",
            str(c_bin),
        ]
    )
    return parse_bytes(run([str(c_bin)]))


def run_go(out_dir: Path) -> list[int]:
    mod_dir = out_dir / "go_demo"
    pkg_dir = mod_dir / "telemfp"
    pkg_dir.mkdir(parents=True, exist_ok=True)

    (pkg_dir / "telemfp_bp.go").write_text((out_dir / "telemfp_bp.go").read_text(encoding="utf-8"), encoding="utf-8")

    (mod_dir / "go.mod").write_text(
        """module telemfpdemo

go 1.25

require github.com/hit9/bitproto/lib/go v0.0.0-00010101000000-000000000000
replace github.com/hit9/bitproto/lib/go => """
        + str(ROOT / "lib" / "go")
        + "\n",
        encoding="utf-8",
    )

    (mod_dir / "main.go").write_text(
        """package main

import (
    \"fmt\"
    \"math\"

    t \"telemfpdemo/telemfp\"
)

func main() {
    s := &t.Sensor{}
    s.SetTemperatureFloat(12.34567891)
    s.SetHumidityDouble(-3.21098765)
    _ = s.SetHistoryFloatAt(0, 0.1)
    _ = s.SetHistoryFloatAt(1, -0.2)
    _ = s.SetHistoryFloatAt(2, 21.47483647)

    const eps = 0.5/100000000.0 + 1e-12
    if math.Abs(s.GetTemperatureFloat()-12.34567891) > eps {
        panic(\"temperature helper mismatch\")
    }
    if math.Abs(s.GetHumidityDouble()+3.21098765) > eps {
        panic(\"humidity helper mismatch\")
    }

    buf := s.Encode()
    d := &t.Sensor{}
    d.Decode(buf)
    if math.Abs(d.GetTemperatureFloat()-12.34567891) > eps {
        panic(\"decode helper mismatch\")
    }

    for _, b := range buf {
        fmt.Printf(\"%d \", b)
    }
}
""",
        encoding="utf-8",
    )

    go_bin = mod_dir / "go_demo"
    run(["go", "build", "-o", str(go_bin), "main.go"], cwd=mod_dir)
    return parse_bytes(run([str(go_bin)], cwd=mod_dir))


def run_py(out_dir: Path) -> list[int]:
    script = """
import math
import sys

sys.path.insert(0, r'""" + str(out_dir) + """')
sys.path.insert(0, r'""" + str(ROOT / "lib" / "py") + """')

import telemfp_bp as t

s = t.Sensor()
s.set_temperature_float(12.34567891)
s.set_humidity_double(-3.21098765)
assert s.set_history_float_at(0, 0.1)
assert s.set_history_float_at(1, -0.2)
assert s.set_history_float_at(2, 21.47483647)

eps = 0.5 / 100000000.0 + 1e-12
assert math.fabs(s.get_temperature_float() - 12.34567891) <= eps
assert math.fabs(s.get_humidity_double() + 3.21098765) <= eps

buf = s.encode()
d = t.Sensor()
d.decode(buf)
assert math.fabs(d.get_temperature_float() - 12.34567891) <= eps

print(' '.join(str(int(x)) for x in buf), end=' ')
"""
    return parse_bytes(run([sys.executable, "-c", script]))


def main() -> int:
    print("[demo] schema:", PROTO)
    print("[demo] values:", VALUES)
    with tempfile.TemporaryDirectory() as td:
        out_dir = Path(td)
        print("[demo] generating native c/go/py...")
        generate_native(out_dir)

        print("[demo] running C encoder...")
        c_bytes = run_c(out_dir)
        print("[demo] running Go encoder...")
        go_bytes = run_go(out_dir)
        print("[demo] running Python encoder...")
        py_bytes = run_py(out_dir)

        if c_bytes != go_bytes or c_bytes != py_bytes:
            print("[demo] byte mismatch detected")
            print("  c :", c_bytes)
            print("  go:", go_bytes)
            print("  py:", py_bytes)
            return 1

        print("[demo] bytes equal across C/Go/Python")
        print("[demo] payload bytes:", " ".join(map(str, c_bytes)))
        print("[demo] PASS")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
