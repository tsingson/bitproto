import random
import shutil
import subprocess
from pathlib import Path

try:
    import pytest
except Exception:  # pragma: no cover
    class _MarkFallback:
        @staticmethod
        def skipif(*_args, **_kwargs):
            def _decorator(fn):
                return fn

            return _decorator

    class _PytestFallback:
        mark = _MarkFallback()

    pytest = _PytestFallback()  # type: ignore

ROOT_DIR = Path(__file__).resolve().parents[2]
LIB_C_DIR = ROOT_DIR / "lib" / "c"
EXAMPLE_DIR = ROOT_DIR / "example"
EXAMPLE_PROTO = EXAMPLE_DIR / "example.bitproto"
EXAMPLE_GO_DIR = EXAMPLE_DIR / "Go"
EXAMPLE_PY_DIR = EXAMPLE_DIR / "Python"

_HAVE_TOOLS = all(shutil.which(t) for t in ("go", "cc", "python"))
requires_tools = pytest.mark.skipif(not _HAVE_TOOLS, reason="needs go/cc/python on PATH")

ARG_KEYS = [
    "status",
    "latitude",
    "longitude",
    "altitude",
    "yaw",
    "pitch",
    "roll",
    "vel0",
    "vel1",
    "vel2",
    "acc0",
    "acc1",
    "acc2",
    "battery",
    "power_status",
    "is_charging",
    "signal",
    "heartbeat_at",
    "landing_status",
    "pressure0",
    "pressure1",
    "p0_id",
    "p0_status",
    "p0_direction",
    "p1_id",
    "p1_status",
    "p1_direction",
    "p2_id",
    "p2_status",
    "p2_direction",
    "p3_id",
    "p3_status",
    "p3_direction",
]


def _run(cmd: list[str], cwd: Path | None = None) -> bytes:
    return subprocess.check_output(cmd, cwd=str(cwd) if cwd else None)


def _normalize_wire_output(out: bytes) -> list[int]:
    return [int(p) for p in out.decode("utf-8").strip().split() if p]


def _vector_to_args(v: dict[str, int]) -> list[str]:
    return [str(v[k]) for k in ARG_KEYS]


def _build_go_encoder(tmp_path: Path) -> Path:
    go_main = tmp_path / "ref_go_encoder.go"
    go_main.write_text(
        """package main

import (
    \"fmt\"
    \"os\"
    \"reflect\"
    \"strconv\"
    bp \"github.com/hit9/bitproto/example/Go/gen-bp\"
)

func mustArg(i int) int64 {
    v, err := strconv.ParseInt(os.Args[i], 10, 64)
    if err != nil {
        panic(err)
    }
    return v
}

func main() {
    if len(os.Args) != 34 {
        panic(\"expected 33 numeric args\")
    }

    d := &bp.Drone{}
    i := 1

    d.Status = bp.DroneStatus(uint8(mustArg(i))); i++
    d.Position.Latitude = uint32(mustArg(i)); i++
    d.Position.Longitude = uint32(mustArg(i)); i++
    d.Position.Altitude = uint32(mustArg(i)); i++

    d.Flight.Pose.Yaw = int32(mustArg(i)); i++
    d.Flight.Pose.Pitch = int32(mustArg(i)); i++
    d.Flight.Pose.Roll = int32(mustArg(i)); i++

    d.Flight.Velocity[0] = int32(mustArg(i)); i++
    d.Flight.Velocity[1] = int32(mustArg(i)); i++
    d.Flight.Velocity[2] = int32(mustArg(i)); i++

    d.Flight.Acceleration[0] = int32(mustArg(i)); i++
    d.Flight.Acceleration[1] = int32(mustArg(i)); i++
    d.Flight.Acceleration[2] = int32(mustArg(i)); i++

    d.Power.Battery = uint8(mustArg(i)); i++
    d.Power.Status = bp.PowerStatus(uint8(mustArg(i))); i++
    d.Power.IsCharging = mustArg(i) != 0; i++

    d.Network.Signal = uint8(mustArg(i)); i++
    d.Network.HeartbeatAt = bp.Timestamp(int32(mustArg(i))); i++
    d.LandingGear.Status = bp.LandingGearStatus(uint8(mustArg(i))); i++

    d.PressureSensor.Pressures[0] = int32(mustArg(i)); i++
    d.PressureSensor.Pressures[1] = int32(mustArg(i)); i++

    d.Propellers[0].Id = uint8(mustArg(i)); i++
    d.Propellers[0].Status = bp.PropellerStatus(uint8(mustArg(i))); i++
    d.Propellers[0].Direction = bp.RotatingDirection(uint8(mustArg(i))); i++
    d.Propellers[1].Id = uint8(mustArg(i)); i++
    d.Propellers[1].Status = bp.PropellerStatus(uint8(mustArg(i))); i++
    d.Propellers[1].Direction = bp.RotatingDirection(uint8(mustArg(i))); i++
    d.Propellers[2].Id = uint8(mustArg(i)); i++
    d.Propellers[2].Status = bp.PropellerStatus(uint8(mustArg(i))); i++
    d.Propellers[2].Direction = bp.RotatingDirection(uint8(mustArg(i))); i++
    d.Propellers[3].Id = uint8(mustArg(i)); i++
    d.Propellers[3].Status = bp.PropellerStatus(uint8(mustArg(i))); i++
    d.Propellers[3].Direction = bp.RotatingDirection(uint8(mustArg(i))); i++

    s := d.Encode()
    d2 := &bp.Drone{}
    d2.Decode(s)
    if !reflect.DeepEqual(d, d2) {
        panic(\"go decode roundtrip mismatch\")
    }

    for _, b := range s {
        fmt.Printf(\"%d \", b)
    }
}
""",
        encoding="utf-8",
    )
    go_bin = tmp_path / "ref_go_encoder"
    _run(["go", "build", "-o", str(go_bin), str(go_main)], cwd=EXAMPLE_GO_DIR)
    return go_bin


def _go_output(go_bin: Path, v: dict[str, int]) -> bytes:
    return _run([str(go_bin), *_vector_to_args(v)], cwd=EXAMPLE_GO_DIR)


def _py_output(v: dict[str, int]) -> bytes:
    script = """
import sys
sys.path.insert(0, r'""" + str(EXAMPLE_PY_DIR) + """')
sys.path.insert(0, r'""" + str(ROOT_DIR / "lib" / "py") + """')
import example_bp as bp

vals = list(map(int, sys.argv[1:]))
assert len(vals) == 33

i = 0
d = bp.Drone()
d.status = vals[i]; i += 1
d.position.latitude = vals[i]; i += 1
d.position.longitude = vals[i]; i += 1
d.position.altitude = vals[i]; i += 1

d.flight.pose.yaw = vals[i]; i += 1
d.flight.pose.pitch = vals[i]; i += 1
d.flight.pose.roll = vals[i]; i += 1
d.flight.velocity[0] = vals[i]; i += 1
d.flight.velocity[1] = vals[i]; i += 1
d.flight.velocity[2] = vals[i]; i += 1
d.flight.acceleration[0] = vals[i]; i += 1
d.flight.acceleration[1] = vals[i]; i += 1
d.flight.acceleration[2] = vals[i]; i += 1

d.power.battery = vals[i]; i += 1
d.power.status = vals[i]; i += 1
d.power.is_charging = bool(vals[i]); i += 1

d.network.signal = vals[i]; i += 1
d.network.heartbeat_at = vals[i]; i += 1
d.landing_gear.status = vals[i]; i += 1

d.pressure_sensor.pressures[0] = vals[i]; i += 1
d.pressure_sensor.pressures[1] = vals[i]; i += 1

d.propellers[0].id = vals[i]; i += 1
d.propellers[0].status = vals[i]; i += 1
d.propellers[0].direction = vals[i]; i += 1
d.propellers[1].id = vals[i]; i += 1
d.propellers[1].status = vals[i]; i += 1
d.propellers[1].direction = vals[i]; i += 1
d.propellers[2].id = vals[i]; i += 1
d.propellers[2].status = vals[i]; i += 1
d.propellers[2].direction = vals[i]; i += 1
d.propellers[3].id = vals[i]; i += 1
d.propellers[3].status = vals[i]; i += 1
d.propellers[3].direction = vals[i]; i += 1

s = d.encode()
d2 = bp.Drone()
d2.decode(s)
assert d2.to_json() == d.to_json()
print(' '.join(str(int(x)) for x in s), end=' ')
"""
    return _run(["python", "-c", script, *_vector_to_args(v)])


def _build_native_artifacts(tmp_path: Path) -> Path:
    out_dir = tmp_path / "native"
    out_dir.mkdir(parents=True, exist_ok=True)

    compiler_go_dir = ROOT_DIR / "compiler-go"
    proto_path = ROOT_DIR / "example" / "example.bitproto"
    _run(["go", "run", ".", "native-c", str(proto_path), str(out_dir)], cwd=compiler_go_dir)
    _run(["go", "run", ".", "native-go", str(proto_path), str(out_dir)], cwd=compiler_go_dir)
    _run(["go", "run", ".", "native-py", str(proto_path), str(out_dir)], cwd=compiler_go_dir)

    return out_dir


def _build_native_c_encoder(out_dir: Path) -> Path:
    c_main = out_dir / "interop_c_main.c"
    c_main.write_text(
        """#include <assert.h>
#include <stdbool.h>
#include <stdint.h>
#include <stdio.h>
#include <stdlib.h>
#include <string.h>
#include "drone_bp.h"

static long long arg(int i, char **argv) {
    return strtoll(argv[i], NULL, 10);
}

static void check_canary(const unsigned char *buf, int from, int to, unsigned char c) {
    for (int i = from; i < to; i++) {
        assert(buf[i] == c);
    }
}

int main(int argc, char **argv) {
    if (argc != 34) {
        return 2;
    }

    struct Drone d = {0};
    int i = 1;

    d.status = (DroneStatus)arg(i++, argv);
    d.position.latitude = (uint32_t)arg(i++, argv);
    d.position.longitude = (uint32_t)arg(i++, argv);
    d.position.altitude = (uint32_t)arg(i++, argv);

    d.flight.pose.yaw = (int32_t)arg(i++, argv);
    d.flight.pose.pitch = (int32_t)arg(i++, argv);
    d.flight.pose.roll = (int32_t)arg(i++, argv);

    d.flight.velocity[0] = (int32_t)arg(i++, argv);
    d.flight.velocity[1] = (int32_t)arg(i++, argv);
    d.flight.velocity[2] = (int32_t)arg(i++, argv);
    d.flight.acceleration[0] = (int32_t)arg(i++, argv);
    d.flight.acceleration[1] = (int32_t)arg(i++, argv);
    d.flight.acceleration[2] = (int32_t)arg(i++, argv);

    d.power.battery = (uint8_t)arg(i++, argv);
    d.power.status = (PowerStatus)arg(i++, argv);
    d.power.is_charging = (bool)arg(i++, argv);
    d.network.signal = (uint8_t)arg(i++, argv);
    d.network.heartbeat_at = (int32_t)arg(i++, argv);
    d.landing_gear.status = (LandingGearStatus)arg(i++, argv);

    d.pressure_sensor.pressures[0] = (int32_t)arg(i++, argv);
    d.pressure_sensor.pressures[1] = (int32_t)arg(i++, argv);

    d.propellers[0].id = (uint8_t)arg(i++, argv);
    d.propellers[0].status = (PropellerStatus)arg(i++, argv);
    d.propellers[0].direction = (RotatingDirection)arg(i++, argv);
    d.propellers[1].id = (uint8_t)arg(i++, argv);
    d.propellers[1].status = (PropellerStatus)arg(i++, argv);
    d.propellers[1].direction = (RotatingDirection)arg(i++, argv);
    d.propellers[2].id = (uint8_t)arg(i++, argv);
    d.propellers[2].status = (PropellerStatus)arg(i++, argv);
    d.propellers[2].direction = (RotatingDirection)arg(i++, argv);
    d.propellers[3].id = (uint8_t)arg(i++, argv);
    d.propellers[3].status = (PropellerStatus)arg(i++, argv);
    d.propellers[3].direction = (RotatingDirection)arg(i++, argv);

    unsigned char arena[BYTES_LENGTH_DRONE + 32];
    memset(arena, 0xA5, sizeof(arena));
    unsigned char *payload = arena + 16;
    memset(payload, 0, BYTES_LENGTH_DRONE);

    EncodeDrone(&d, payload);
    check_canary(arena, 0, 16, 0xA5);
    check_canary(arena, 16 + BYTES_LENGTH_DRONE, BYTES_LENGTH_DRONE + 32, 0xA5);

    struct Drone d2 = {0};
    DecodeDrone(&d2, payload);
    check_canary(arena, 0, 16, 0xA5);
    check_canary(arena, 16 + BYTES_LENGTH_DRONE, BYTES_LENGTH_DRONE + 32, 0xA5);

    assert(d2.status == d.status);
    assert(d2.position.latitude == d.position.latitude);
    assert(d2.position.longitude == d.position.longitude);
    assert(d2.position.altitude == d.position.altitude);
    assert(d2.flight.pose.yaw == d.flight.pose.yaw);
    assert(d2.flight.pose.pitch == d.flight.pose.pitch);
    assert(d2.flight.pose.roll == d.flight.pose.roll);
    assert(d2.flight.velocity[0] == d.flight.velocity[0]);
    assert(d2.flight.velocity[1] == d.flight.velocity[1]);
    assert(d2.flight.velocity[2] == d.flight.velocity[2]);
    assert(d2.flight.acceleration[0] == d.flight.acceleration[0]);
    assert(d2.flight.acceleration[1] == d.flight.acceleration[1]);
    assert(d2.flight.acceleration[2] == d.flight.acceleration[2]);
    assert(d2.power.battery == d.power.battery);
    assert(d2.power.status == d.power.status);
    assert(d2.power.is_charging == d.power.is_charging);
    assert(d2.network.signal == d.network.signal);
    assert(d2.network.heartbeat_at == d.network.heartbeat_at);
    assert(d2.landing_gear.status == d.landing_gear.status);
    assert(d2.pressure_sensor.pressures[0] == d.pressure_sensor.pressures[0]);
    assert(d2.pressure_sensor.pressures[1] == d.pressure_sensor.pressures[1]);

    for (int j = 0; j < BYTES_LENGTH_DRONE; j++) {
        printf("%u ", payload[j]);
    }
    return 0;
}
""",
        encoding="utf-8",
    )

    c_bin = out_dir / "interop_native_c"
    _run(
        [
            "cc",
            str(c_main),
            str(out_dir / "drone_bp.c"),
            str(LIB_C_DIR / "bitproto.c"),
            "-I",
            str(out_dir),
            "-I",
            str(LIB_C_DIR),
            "-o",
            str(c_bin),
        ]
    )
    return c_bin


def _native_c_output(c_bin: Path, v: dict[str, int]) -> bytes:
    return _run([str(c_bin), *_vector_to_args(v)])


def _build_native_go_encoder(out_dir: Path) -> Path:
    mod_dir = out_dir / "native_go_mod"
    pkg_dir = mod_dir / "drone"
    pkg_dir.mkdir(parents=True, exist_ok=True)

    (mod_dir / "go.mod").write_text(
        """module nativegocheck

    go 1.25

require github.com/hit9/bitproto/lib/go v0.0.0-00010101000000-000000000000
replace github.com/hit9/bitproto/lib/go => """
        + str(ROOT_DIR / "lib" / "go")
        + "\n",
        encoding="utf-8",
    )

    (pkg_dir / "drone_bp.go").write_text((out_dir / "drone_bp.go").read_text(encoding="utf-8"), encoding="utf-8")

    main_go = mod_dir / "main.go"
    main_go.write_text(
        """package main

import (
    \"fmt\"
    \"os\"
    \"reflect\"
    \"strconv\"

    ng \"nativegocheck/drone\"
)

func mustArg(i int) int64 {
    v, err := strconv.ParseInt(os.Args[i], 10, 64)
    if err != nil {
        panic(err)
    }
    return v
}

func main() {
    if len(os.Args) != 34 {
        panic(\"expected 33 numeric args\")
    }

    d := &ng.Drone{}
    i := 1

    d.Status = ng.DroneStatus(uint8(mustArg(i))); i++
    d.Position.Latitude = uint32(mustArg(i)); i++
    d.Position.Longitude = uint32(mustArg(i)); i++
    d.Position.Altitude = uint32(mustArg(i)); i++

    d.Flight.Pose.Yaw = int32(mustArg(i)); i++
    d.Flight.Pose.Pitch = int32(mustArg(i)); i++
    d.Flight.Pose.Roll = int32(mustArg(i)); i++

    d.Flight.Velocity[0] = int32(mustArg(i)); i++
    d.Flight.Velocity[1] = int32(mustArg(i)); i++
    d.Flight.Velocity[2] = int32(mustArg(i)); i++

    d.Flight.Acceleration[0] = int32(mustArg(i)); i++
    d.Flight.Acceleration[1] = int32(mustArg(i)); i++
    d.Flight.Acceleration[2] = int32(mustArg(i)); i++

    d.Power.Battery = uint8(mustArg(i)); i++
    d.Power.Status = ng.PowerStatus(uint8(mustArg(i))); i++
    d.Power.IsCharging = mustArg(i) != 0; i++

    d.Network.Signal = uint8(mustArg(i)); i++
    d.Network.HeartbeatAt = ng.Timestamp(int32(mustArg(i))); i++
    d.LandingGear.Status = ng.LandingGearStatus(uint8(mustArg(i))); i++

    d.PressureSensor.Pressures[0] = int32(mustArg(i)); i++
    d.PressureSensor.Pressures[1] = int32(mustArg(i)); i++

    d.Propellers[0].Id = uint8(mustArg(i)); i++
    d.Propellers[0].Status = ng.PropellerStatus(uint8(mustArg(i))); i++
    d.Propellers[0].Direction = ng.RotatingDirection(uint8(mustArg(i))); i++
    d.Propellers[1].Id = uint8(mustArg(i)); i++
    d.Propellers[1].Status = ng.PropellerStatus(uint8(mustArg(i))); i++
    d.Propellers[1].Direction = ng.RotatingDirection(uint8(mustArg(i))); i++
    d.Propellers[2].Id = uint8(mustArg(i)); i++
    d.Propellers[2].Status = ng.PropellerStatus(uint8(mustArg(i))); i++
    d.Propellers[2].Direction = ng.RotatingDirection(uint8(mustArg(i))); i++
    d.Propellers[3].Id = uint8(mustArg(i)); i++
    d.Propellers[3].Status = ng.PropellerStatus(uint8(mustArg(i))); i++
    d.Propellers[3].Direction = ng.RotatingDirection(uint8(mustArg(i))); i++

    s := d.Encode()
    d2 := &ng.Drone{}
    d2.Decode(s)
    if !reflect.DeepEqual(d, d2) {
        panic(\"native-go decode roundtrip mismatch\")
    }

    for _, bt := range s {
        fmt.Printf(\"%d \", bt)
    }
}
""",
        encoding="utf-8",
    )

    go_bin = mod_dir / "interop_native_go"
    _run(["go", "build", "-o", str(go_bin), str(main_go)], cwd=mod_dir)
    return go_bin


def _native_go_output(go_bin: Path, v: dict[str, int]) -> bytes:
    return _run([str(go_bin), *_vector_to_args(v)])


def _native_py_output(out_dir: Path, v: dict[str, int]) -> bytes:
    script = """
import sys
sys.path.insert(0, r'""" + str(out_dir) + """')
sys.path.insert(0, r'""" + str(ROOT_DIR / "lib" / "py") + """')
import drone_bp as ng

vals = list(map(int, sys.argv[1:]))
assert len(vals) == 33

i = 0
d = ng.Drone()
d.status = vals[i]; i += 1
d.position.latitude = vals[i]; i += 1
d.position.longitude = vals[i]; i += 1
d.position.altitude = vals[i]; i += 1

d.flight.pose.yaw = vals[i]; i += 1
d.flight.pose.pitch = vals[i]; i += 1
d.flight.pose.roll = vals[i]; i += 1

d.flight.velocity[0] = vals[i]; i += 1
d.flight.velocity[1] = vals[i]; i += 1
d.flight.velocity[2] = vals[i]; i += 1
d.flight.acceleration[0] = vals[i]; i += 1
d.flight.acceleration[1] = vals[i]; i += 1
d.flight.acceleration[2] = vals[i]; i += 1

d.power.battery = vals[i]; i += 1
d.power.status = vals[i]; i += 1
d.power.is_charging = bool(vals[i]); i += 1

d.network.signal = vals[i]; i += 1
d.network.heartbeat_at = vals[i]; i += 1
d.landing_gear.status = vals[i]; i += 1

d.pressure_sensor.pressures[0] = vals[i]; i += 1
d.pressure_sensor.pressures[1] = vals[i]; i += 1

d.propellers[0].id = vals[i]; i += 1
d.propellers[0].status = vals[i]; i += 1
d.propellers[0].direction = vals[i]; i += 1
d.propellers[1].id = vals[i]; i += 1
d.propellers[1].status = vals[i]; i += 1
d.propellers[1].direction = vals[i]; i += 1
d.propellers[2].id = vals[i]; i += 1
d.propellers[2].status = vals[i]; i += 1
d.propellers[2].direction = vals[i]; i += 1
d.propellers[3].id = vals[i]; i += 1
d.propellers[3].status = vals[i]; i += 1
d.propellers[3].direction = vals[i]; i += 1

s = d.encode()
d2 = ng.Drone()
d2.decode(s)
assert bytes(d2.encode()) == bytes(s)
print(' '.join(str(int(x)) for x in s), end=' ')
"""
    return _run(["python", "-c", script, *_vector_to_args(v)])


def _boundary_vectors() -> list[dict[str, int]]:
    i32_min = -(2**31)
    i32_max = (2**31) - 1
    i24_min = -(2**23)
    i24_max = (2**23) - 1
    u32_max = (2**32) - 1

    return [
        {
            "status": 0,
            "latitude": 0,
            "longitude": 0,
            "altitude": 0,
            "yaw": 0,
            "pitch": 0,
            "roll": 0,
            "vel0": 0,
            "vel1": 0,
            "vel2": 0,
            "acc0": 0,
            "acc1": 0,
            "acc2": 0,
            "battery": 0,
            "power_status": 0,
            "is_charging": 0,
            "signal": 0,
            "heartbeat_at": 0,
            "landing_status": 0,
            "pressure0": 0,
            "pressure1": 0,
            "p0_id": 0,
            "p0_status": 0,
            "p0_direction": 0,
            "p1_id": 0,
            "p1_status": 0,
            "p1_direction": 0,
            "p2_id": 0,
            "p2_status": 0,
            "p2_direction": 0,
            "p3_id": 0,
            "p3_status": 0,
            "p3_direction": 0,
        },
        {
            "status": 4,
            "latitude": u32_max,
            "longitude": u32_max,
            "altitude": u32_max,
            "yaw": i32_max,
            "pitch": i32_max,
            "roll": i32_max,
            "vel0": i32_max,
            "vel1": i32_max,
            "vel2": i32_max,
            "acc0": i32_max,
            "acc1": i32_max,
            "acc2": i32_max,
            "battery": 255,
            "power_status": 2,
            "is_charging": 1,
            "signal": 15,
            "heartbeat_at": i32_max,
            "landing_status": 2,
            "pressure0": i24_max,
            "pressure1": i24_max,
            "p0_id": 255,
            "p0_status": 2,
            "p0_direction": 2,
            "p1_id": 255,
            "p1_status": 2,
            "p1_direction": 2,
            "p2_id": 255,
            "p2_status": 2,
            "p2_direction": 2,
            "p3_id": 255,
            "p3_status": 2,
            "p3_direction": 2,
        },
        {
            "status": 1,
            "latitude": 1,
            "longitude": 2,
            "altitude": 3,
            "yaw": i32_min,
            "pitch": i32_min,
            "roll": i32_min,
            "vel0": i32_min,
            "vel1": i32_min,
            "vel2": i32_min,
            "acc0": i32_min,
            "acc1": i32_min,
            "acc2": i32_min,
            "battery": 1,
            "power_status": 1,
            "is_charging": 0,
            "signal": 1,
            "heartbeat_at": i32_min,
            "landing_status": 1,
            "pressure0": i24_min,
            "pressure1": i24_min,
            "p0_id": 1,
            "p0_status": 1,
            "p0_direction": 1,
            "p1_id": 2,
            "p1_status": 1,
            "p1_direction": 1,
            "p2_id": 3,
            "p2_status": 1,
            "p2_direction": 1,
            "p3_id": 4,
            "p3_status": 1,
            "p3_direction": 1,
        },
    ]


def _random_vector(rng: random.Random) -> dict[str, int]:
    return {
        "status": rng.randint(0, 4),
        "latitude": rng.randint(0, (2**32) - 1),
        "longitude": rng.randint(0, (2**32) - 1),
        "altitude": rng.randint(0, (2**32) - 1),
        "yaw": rng.randint(-(2**31), (2**31) - 1),
        "pitch": rng.randint(-(2**31), (2**31) - 1),
        "roll": rng.randint(-(2**31), (2**31) - 1),
        "vel0": rng.randint(-(2**31), (2**31) - 1),
        "vel1": rng.randint(-(2**31), (2**31) - 1),
        "vel2": rng.randint(-(2**31), (2**31) - 1),
        "acc0": rng.randint(-(2**31), (2**31) - 1),
        "acc1": rng.randint(-(2**31), (2**31) - 1),
        "acc2": rng.randint(-(2**31), (2**31) - 1),
        "battery": rng.randint(0, 255),
        "power_status": rng.randint(0, 2),
        "is_charging": rng.randint(0, 1),
        "signal": rng.randint(0, 15),
        "heartbeat_at": rng.randint(-(2**31), (2**31) - 1),
        "landing_status": rng.randint(0, 2),
        "pressure0": rng.randint(-(2**23), (2**23) - 1),
        "pressure1": rng.randint(-(2**23), (2**23) - 1),
        "p0_id": rng.randint(0, 255),
        "p0_status": rng.randint(0, 2),
        "p0_direction": rng.randint(0, 2),
        "p1_id": rng.randint(0, 255),
        "p1_status": rng.randint(0, 2),
        "p1_direction": rng.randint(0, 2),
        "p2_id": rng.randint(0, 255),
        "p2_status": rng.randint(0, 2),
        "p2_direction": rng.randint(0, 2),
        "p3_id": rng.randint(0, 255),
        "p3_status": rng.randint(0, 2),
        "p3_direction": rng.randint(0, 2),
    }


def _run_vectors(tmp_path: Path, vectors: list[dict[str, int]]) -> None:
    ref_go_bin = _build_go_encoder(tmp_path)
    native_out = _build_native_artifacts(tmp_path)
    native_c_bin = _build_native_c_encoder(native_out)
    native_go_bin = _build_native_go_encoder(native_out)

    for idx, v in enumerate(vectors):
        ref_go_wire = _normalize_wire_output(_go_output(ref_go_bin, v))
        ref_py_wire = _normalize_wire_output(_py_output(v))
        native_c_wire = _normalize_wire_output(_native_c_output(native_c_bin, v))
        native_go_wire = _normalize_wire_output(_native_go_output(native_go_bin, v))
        native_py_wire = _normalize_wire_output(_native_py_output(native_out, v))

        assert ref_go_wire == ref_py_wire, f"vector#{idx}: reference go != reference py"
        assert native_c_wire == ref_go_wire, f"vector#{idx}: native-c != reference"
        assert native_go_wire == ref_go_wire, f"vector#{idx}: native-go != reference"
        assert native_py_wire == ref_go_wire, f"vector#{idx}: native-py != reference"


@requires_tools
def test_native_three_end_boundary(tmp_path: Path) -> None:
    _run_vectors(tmp_path, _boundary_vectors())


@requires_tools
def test_native_three_end_random_stress(tmp_path: Path) -> None:
    rng = random.Random(20260726)
    vectors = [_random_vector(rng) for _ in range(80)]
    _run_vectors(tmp_path, vectors)


def _run_standalone() -> None:
    import tempfile

    with tempfile.TemporaryDirectory() as td:
        test_native_three_end_boundary(Path(td))
    with tempfile.TemporaryDirectory() as td:
        test_native_three_end_random_stress(Path(td))


if __name__ == "__main__":
    _run_standalone()
    print("native three-end interop tests passed")
