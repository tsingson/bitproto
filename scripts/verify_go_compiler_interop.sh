#!/usr/bin/env bash

set -euo pipefail

ROOT_DIR="$(cd "$(dirname "$0")/.." && pwd)"
GO_COMPILER_BIN="$ROOT_DIR/dist/bitproto-go"
SCHEMA_FILE="$ROOT_DIR/example/example.bitproto"
LIB_C_FILE="$ROOT_DIR/lib/c/bitproto.c"
LIB_C_INCLUDE_DIR="$ROOT_DIR/lib/c"
LIB_PY_DIR="$ROOT_DIR/lib/py"

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

TMP_DIR="$(mktemp -d "${TMPDIR:-/tmp}/bitproto-interop-XXXXXX")"
cleanup() {
    rm -rf "$TMP_DIR"
}
trap cleanup EXIT

GEN_C_DIR="$TMP_DIR/gen-c"
GEN_GO_DIR="$TMP_DIR/gen-go"
GEN_PY_DIR="$TMP_DIR/gen-py"
BIN_DIR="$TMP_DIR/bin"
WORK_C_DIR="$TMP_DIR/work-c"
WORK_GO_DIR="$TMP_DIR/work-go"
WORK_PY_DIR="$TMP_DIR/work-py"

mkdir -p "$GEN_C_DIR" "$GEN_GO_DIR" "$GEN_PY_DIR" "$BIN_DIR" "$WORK_C_DIR" "$WORK_GO_DIR" "$WORK_PY_DIR"

echo "[step] generate C/Go/Python from same .bitproto by go-compiler"
"$GO_COMPILER_BIN" c "$SCHEMA_FILE" "$GEN_C_DIR"
"$GO_COMPILER_BIN" go "$SCHEMA_FILE" "$GEN_GO_DIR"
"$GO_COMPILER_BIN" py "$SCHEMA_FILE" "$GEN_PY_DIR"

cat >"$WORK_C_DIR/main.c" <<'EOF'
#include <assert.h>
#include <stdbool.h>
#include <stdint.h>
#include <stdio.h>
#include <stdlib.h>

#include "example_bp.h"

static void fill_drone(struct Drone *drone) {
    *drone = (struct Drone){0};
    drone->status = DRONE_STATUS_RISING;
    drone->position.longitude = 2000;
    drone->position.latitude = 2000;
    drone->position.altitude = 1080;
    drone->flight.acceleration[0] = -1001;
    drone->power.is_charging = true;
    drone->propellers[0].direction = ROTATING_DIRECTION_CLOCK_WISE;
    drone->pressure_sensor.pressures[0] = -11;
    drone->flight.pose.yaw = -10;
}

static void assert_drone(const struct Drone *drone) {
    assert(drone->status == DRONE_STATUS_RISING);
    assert(drone->position.longitude == 2000);
    assert(drone->position.latitude == 2000);
    assert(drone->position.altitude == 1080);
    assert(drone->flight.acceleration[0] == -1001);
    assert(drone->power.is_charging == true);
    assert(drone->propellers[0].direction == ROTATING_DIRECTION_CLOCK_WISE);
    assert(drone->pressure_sensor.pressures[0] == -11);
    assert(drone->flight.pose.yaw == -10);
}

int main(int argc, char **argv) {
    if (argc != 3) {
        fprintf(stderr, "usage: %s <encode|decode> <binfile>\n", argv[0]);
        return 2;
    }

    const char *mode = argv[1];
    const char *path = argv[2];

    if (mode[0] == 'e') {
        struct Drone drone;
        unsigned char s[BYTES_LENGTH_DRONE] = {0};
        fill_drone(&drone);
        EncodeDrone(&drone, s);

        FILE *fp = fopen(path, "wb");
        if (!fp) {
            perror("fopen");
            return 1;
        }
        size_t n = fwrite(s, 1, BYTES_LENGTH_DRONE, fp);
        fclose(fp);
        if (n != BYTES_LENGTH_DRONE) {
            fprintf(stderr, "short write\n");
            return 1;
        }
        return 0;
    }

    struct Drone drone = {0};
    unsigned char s[BYTES_LENGTH_DRONE] = {0};
    FILE *fp = fopen(path, "rb");
    if (!fp) {
        perror("fopen");
        return 1;
    }
    size_t n = fread(s, 1, BYTES_LENGTH_DRONE, fp);
    fclose(fp);
    if (n != BYTES_LENGTH_DRONE) {
        fprintf(stderr, "short read\n");
        return 1;
    }
    DecodeDrone(&drone, s);
    assert_drone(&drone);
    return 0;
}
EOF

echo "[step] build C interop runner"
cc -std=c99 -Wall -Wextra -O2 \
    "$WORK_C_DIR/main.c" "$GEN_C_DIR/example_bp.c" "$LIB_C_FILE" \
    -I"$GEN_C_DIR" -I"$LIB_C_INCLUDE_DIR" \
    -o "$WORK_C_DIR/interop_c"

mkdir -p "$WORK_GO_DIR/genbp"
cp "$GEN_GO_DIR/example_bp.go" "$WORK_GO_DIR/genbp/example_bp.go"

cat >"$WORK_GO_DIR/go.mod" <<EOF
module interop

go 1.22

require github.com/hit9/bitproto/lib/go v0.0.0-00010101000000-000000000000

replace github.com/hit9/bitproto/lib/go => $ROOT_DIR/lib/go
EOF

cat >"$WORK_GO_DIR/main.go" <<'EOF'
package main

import (
    "fmt"
    "os"

    bp "interop/genbp"
)

func fillDrone() *bp.Drone {
    drone := &bp.Drone{}
    drone.Status = bp.DRONE_STATUS_RISING
    drone.Position.Longitude = 2000
    drone.Position.Latitude = 2000
    drone.Position.Altitude = 1080
    drone.Flight.Acceleration[0] = -1001
    drone.Power.IsCharging = true
    drone.Propellers[0].Direction = bp.ROTATING_DIRECTION_CLOCK_WISE
    drone.PressureSensor.Pressures[0] = -11
    drone.Flight.Pose.Yaw = -10
    return drone
}

func assertDrone(drone *bp.Drone) {
    if drone.Status != bp.DRONE_STATUS_RISING {
        panic("status mismatch")
    }
    if drone.Position.Longitude != 2000 || drone.Position.Latitude != 2000 || drone.Position.Altitude != 1080 {
        panic("position mismatch")
    }
    if drone.Flight.Acceleration[0] != -1001 {
        panic("acceleration mismatch")
    }
    if !drone.Power.IsCharging {
        panic("is_charging mismatch")
    }
    if drone.Propellers[0].Direction != bp.ROTATING_DIRECTION_CLOCK_WISE {
        panic("direction mismatch")
    }
    if drone.PressureSensor.Pressures[0] != -11 {
        panic("pressure mismatch")
    }
    if drone.Flight.Pose.Yaw != -10 {
        panic("yaw mismatch")
    }
}

func main() {
    if len(os.Args) != 3 {
        fmt.Fprintf(os.Stderr, "usage: %s <encode|decode> <binfile>\n", os.Args[0])
        os.Exit(2)
    }

    mode := os.Args[1]
    path := os.Args[2]

    switch mode {
    case "encode":
        s := fillDrone().Encode()
        if err := os.WriteFile(path, s, 0644); err != nil {
            panic(err)
        }
    case "decode":
        s, err := os.ReadFile(path)
        if err != nil {
            panic(err)
        }
        drone := &bp.Drone{}
        drone.Decode(s)
        assertDrone(drone)
    default:
        panic("invalid mode")
    }
}
EOF

echo "[step] build Go interop runner"
(cd "$WORK_GO_DIR" && go build -o interop_go .)

cat >"$WORK_PY_DIR/main.py" <<'EOF'
import sys

import example_bp as bp


def fill_drone() -> bp.Drone:
    drone = bp.Drone()
    drone.status = bp.DRONE_STATUS_RISING
    drone.position.longitude = 2000
    drone.position.latitude = 2000
    drone.position.altitude = 1080
    drone.flight.acceleration[0] = -1001
    drone.power.is_charging = True
    drone.propellers[0].direction = bp.ROTATING_DIRECTION_CLOCK_WISE
    drone.pressure_sensor.pressures[0] = -11
    drone.flight.pose.yaw = -10
    return drone


def assert_drone(drone: bp.Drone) -> None:
    assert drone.status == bp.DRONE_STATUS_RISING
    assert drone.position.longitude == 2000
    assert drone.position.latitude == 2000
    assert drone.position.altitude == 1080
    assert drone.flight.acceleration[0] == -1001
    assert drone.power.is_charging is True
    assert drone.propellers[0].direction == bp.ROTATING_DIRECTION_CLOCK_WISE
    assert drone.pressure_sensor.pressures[0] == -11
    assert drone.flight.pose.yaw == -10


def main() -> None:
    if len(sys.argv) != 3:
        raise SystemExit(f"usage: {sys.argv[0]} <encode|decode> <binfile>")
    mode = sys.argv[1]
    path = sys.argv[2]
    if mode == "encode":
        s = fill_drone().encode()
        with open(path, "wb") as f:
            f.write(s)
        return
    if mode == "decode":
        with open(path, "rb") as f:
            s = bytearray(f.read())
        drone = bp.Drone()
        drone.decode(s)
        assert_drone(drone)
        return
    raise SystemExit("invalid mode")


if __name__ == "__main__":
    main()
EOF

echo "[step] serialize into .bin via Python/C/Go"
PYTHONPATH="$GEN_PY_DIR:$LIB_PY_DIR" "$PYTHON_BIN" "$WORK_PY_DIR/main.py" encode "$BIN_DIR/py.bin"
"$WORK_C_DIR/interop_c" encode "$BIN_DIR/c.bin"
"$WORK_GO_DIR/interop_go" encode "$BIN_DIR/go.bin"

echo "[step] compare .bin bytes"
cmp "$BIN_DIR/py.bin" "$BIN_DIR/c.bin"
cmp "$BIN_DIR/py.bin" "$BIN_DIR/go.bin"

echo "[step] cross-deserialize all bins in Python"
PYTHONPATH="$GEN_PY_DIR:$LIB_PY_DIR" "$PYTHON_BIN" "$WORK_PY_DIR/main.py" decode "$BIN_DIR/py.bin"
PYTHONPATH="$GEN_PY_DIR:$LIB_PY_DIR" "$PYTHON_BIN" "$WORK_PY_DIR/main.py" decode "$BIN_DIR/c.bin"
PYTHONPATH="$GEN_PY_DIR:$LIB_PY_DIR" "$PYTHON_BIN" "$WORK_PY_DIR/main.py" decode "$BIN_DIR/go.bin"

echo "[step] cross-deserialize all bins in C"
"$WORK_C_DIR/interop_c" decode "$BIN_DIR/py.bin"
"$WORK_C_DIR/interop_c" decode "$BIN_DIR/c.bin"
"$WORK_C_DIR/interop_c" decode "$BIN_DIR/go.bin"

echo "[step] cross-deserialize all bins in Go"
"$WORK_GO_DIR/interop_go" decode "$BIN_DIR/py.bin"
"$WORK_GO_DIR/interop_go" decode "$BIN_DIR/c.bin"
"$WORK_GO_DIR/interop_go" decode "$BIN_DIR/go.bin"

echo "[done] cross-language serialization/deserialization interop verified"
