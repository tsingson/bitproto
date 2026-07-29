#include <assert.h>
#include <stdio.h>
#include <stddef.h>

#include "example_bp.h"

int main(void) {
    // Encode.
    struct Drone drone = {0};

    drone.status = DRONE_STATUS_RISING;
    drone.position.longitude = 2000;
    drone.position.latitude = 2000;
    drone.position.altitude = 1080;
    drone.flight.acceleration[0] = -1001;
    drone.power.is_charging = false;
    drone.propellers[0].direction = ROTATING_DIRECTION_CLOCK_WISE;
    drone.pressure_sensor.pressures[0] = -11;

    unsigned char s[BYTES_LENGTH_DRONE] = {0};

    size_t payload_len = EncodeDrone(&drone, s);
    assert(payload_len == BYTES_LENGTH_DRONE);

    // Decode.
    struct Drone drone_new = {0};
    size_t payload_len_decoded = DecodeDrone(&drone_new, s);
    assert(payload_len_decoded == BYTES_LENGTH_DRONE);

    assert(drone_new.status == drone.status);

    // Json Formatting.
    char buf[512] = {0};
    JsonDrone(&drone_new, buf);
    printf("%s", buf);

    return 0;
}
