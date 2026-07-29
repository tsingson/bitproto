#include <arpa/inet.h>
#include <assert.h>
#include <netinet/in.h>
#include <signal.h>
#include <stdbool.h>
#include <stdio.h>
#include <stdlib.h>
#include <string.h>
#include <sys/socket.h>
#include <time.h>
#include <unistd.h>

#include "drone_bp.h"
#include "frame_header_bp.h"

enum {
    FRAME_MAGIC = 0xB1,
};

static void die(const char *message) {
    perror(message);
    exit(1);
}

static void send_all(int fd, const unsigned char *buf, size_t len) {
    size_t sent = 0;
    while (sent < len) {
        ssize_t n = send(fd, buf + sent, len - sent, 0);
        if (n < 0) die("send");
        if (n == 0) die("send returned 0");
        sent += (size_t)n;
    }
}

static void recv_all(int fd, unsigned char *buf, size_t len) {
    size_t received = 0;
    while (received < len) {
        ssize_t n = recv(fd, buf + received, len - received, 0);
        if (n < 0) die("recv");
        if (n == 0) die("unexpected eof");
        received += (size_t)n;
    }
}

static int connect_to_server(const char *address) {
    const char *colon = strrchr(address, ':');
    assert(colon != NULL);

    char host[128] = {0};
    size_t host_len = (size_t)(colon - address);
    assert(host_len < sizeof(host));
    memcpy(host, address, host_len);

    int port = atoi(colon + 1);
    assert(port > 0);

    int fd = socket(AF_INET, SOCK_STREAM, 0);
    if (fd < 0) die("socket");

    struct sockaddr_in sockaddr = {0};
    sockaddr.sin_family = AF_INET;
    sockaddr.sin_port = htons((uint16_t)port);
    if (inet_pton(AF_INET, host, &sockaddr.sin_addr) != 1) {
        die("inet_pton");
    }

    if (connect(fd, (struct sockaddr *)&sockaddr, sizeof(sockaddr)) < 0) {
        die("connect");
    }

    return fd;
}

static void fill_drone(struct Drone *drone, int seq) {
    memset(drone, 0, sizeof(*drone));

    drone->status = DRONE_STATUS_RISING;
    drone->position.longitude = 2000 + (uint32_t)seq;
    drone->position.latitude = 3000 + (uint32_t)seq;
    drone->position.altitude = 1080 + (uint32_t)seq;
    drone->flight.pose.yaw = 4321 + seq;
    drone->flight.pose.pitch = 1234 + seq;
    drone->flight.pose.roll = 5678 + seq;
    drone->flight.acceleration[0] = -1001 - seq;
    drone->flight.acceleration[1] = 1002 + seq;
    drone->flight.acceleration[2] = 1003 + seq;
    drone->power.is_charging = (seq % 2) == 0;
    drone->power.battery = 98;
    drone->power.status = POWER_STATUS_ON;
    drone->propellers[0].id = 1;
    drone->propellers[0].direction = ROTATING_DIRECTION_CLOCK_WISE;
    drone->propellers[0].status = PROPELLER_STATUS_ROTATING;
    drone->network.signal = 15;
    drone->network.heartbeat_at = 1611280511628 + seq;
    drone->landing_gear.status = LANDING_GEAR_STATUS_FOLDED;
}

static void assert_drone_equal(const struct Drone *lhs, const struct Drone *rhs) {
    assert(lhs->status == rhs->status);
    assert(lhs->position.longitude == rhs->position.longitude);
    assert(lhs->position.latitude == rhs->position.latitude);
    assert(lhs->position.altitude == rhs->position.altitude);
    assert(lhs->flight.pose.yaw == rhs->flight.pose.yaw);
    assert(lhs->flight.pose.pitch == rhs->flight.pose.pitch);
    assert(lhs->flight.pose.roll == rhs->flight.pose.roll);
    assert(lhs->flight.acceleration[0] == rhs->flight.acceleration[0]);
    assert(lhs->flight.acceleration[1] == rhs->flight.acceleration[1]);
    assert(lhs->flight.acceleration[2] == rhs->flight.acceleration[2]);
    assert(lhs->power.is_charging == rhs->power.is_charging);
    assert(lhs->power.battery == rhs->power.battery);
    assert(lhs->power.status == rhs->power.status);
    assert(lhs->propellers[0].id == rhs->propellers[0].id);
    assert(lhs->propellers[0].direction == rhs->propellers[0].direction);
    assert(lhs->propellers[0].status == rhs->propellers[0].status);
    assert(lhs->network.signal == rhs->network.signal);
    assert(lhs->network.heartbeat_at == rhs->network.heartbeat_at);
    assert(lhs->landing_gear.status == rhs->landing_gear.status);
}

static void write_frame(int fd, const unsigned char *payload, size_t payload_len) {
    struct FrameHeader frame_header = {0};
    unsigned char header_buf[BYTES_LENGTH_FRAME_HEADER] = {0};
    assert(payload_len <= 65535);
    frame_header.magic = FRAME_MAGIC;
    frame_header.payload_type = PAYLOAD_TYPE_DRONE;
    frame_header.payload_length = (uint16_t)payload_len;
    assert(EncodeFrameHeader(&frame_header, header_buf) == BYTES_LENGTH_FRAME_HEADER);
    send_all(fd, header_buf, sizeof(header_buf));
    send_all(fd, payload, payload_len);
}

static void write_frame_fragmented(
    int fd, const unsigned char *payload, size_t payload_len, unsigned int *state
) {
    unsigned char frame[BYTES_LENGTH_FRAME_HEADER + BYTES_LENGTH_DRONE] = {0};
    struct FrameHeader frame_header = {0};
    assert(payload_len <= BYTES_LENGTH_DRONE);
    frame_header.magic = FRAME_MAGIC;
    frame_header.payload_type = PAYLOAD_TYPE_DRONE;
    frame_header.payload_length = (uint16_t)payload_len;
    assert(EncodeFrameHeader(&frame_header, frame) == BYTES_LENGTH_FRAME_HEADER);
    memcpy(frame + BYTES_LENGTH_FRAME_HEADER, payload, payload_len);

    size_t frame_len = BYTES_LENGTH_FRAME_HEADER + payload_len;
    size_t offset = 0;
    while (offset < frame_len) {
        size_t remain = frame_len - offset;
        size_t chunk = (size_t)(rand_r(state) % (unsigned int)remain) + 1;
        send_all(fd, frame + offset, chunk);
        offset += chunk;
    }
}

static void read_frame(int fd, unsigned char *payload, size_t payload_len) {
    unsigned char header_buf[BYTES_LENGTH_FRAME_HEADER] = {0};
    struct FrameHeader frame_header = {0};
    recv_all(fd, header_buf, sizeof(header_buf));
    assert(DecodeFrameHeader(&frame_header, header_buf) == BYTES_LENGTH_FRAME_HEADER);
    assert(frame_header.magic == FRAME_MAGIC);
    assert(frame_header.payload_type == PAYLOAD_TYPE_DRONE);
    size_t received_len = (size_t)frame_header.payload_length;
    assert(received_len == payload_len);
    recv_all(fd, payload, received_len);
}

static void run_roundtrip(const char *address, int rounds) {
    int fd = connect_to_server(address);

    for (int seq = 0; seq < rounds; seq++) {
        struct Drone drone = {0};
        fill_drone(&drone, seq);

        unsigned char payload[BYTES_LENGTH_DRONE] = {0};
        size_t payload_len = EncodeDrone(&drone, payload);
        assert(payload_len == BYTES_LENGTH_DRONE);

        write_frame(fd, payload, payload_len);

        unsigned char reply[BYTES_LENGTH_DRONE] = {0};
        read_frame(fd, reply, payload_len);
        assert(memcmp(payload, reply, payload_len) == 0);

        struct Drone decoded = {0};
        size_t decoded_len = DecodeDrone(&decoded, reply);
        assert(decoded_len == BYTES_LENGTH_DRONE);
        assert_drone_equal(&drone, &decoded);
    }

    close(fd);
}

static void run_fragmented_roundtrip(const char *address, int rounds, unsigned int seed) {
    int fd = connect_to_server(address);
    unsigned int state = seed;

    for (int seq = 0; seq < rounds; seq++) {
        struct Drone drone = {0};
        fill_drone(&drone, seq);

        unsigned char payload[BYTES_LENGTH_DRONE] = {0};
        size_t payload_len = EncodeDrone(&drone, payload);
        assert(payload_len == BYTES_LENGTH_DRONE);

        write_frame_fragmented(fd, payload, payload_len, &state);

        unsigned char reply[BYTES_LENGTH_DRONE] = {0};
        read_frame(fd, reply, payload_len);
        assert(memcmp(payload, reply, payload_len) == 0);

        struct Drone decoded = {0};
        size_t decoded_len = DecodeDrone(&decoded, reply);
        assert(decoded_len == BYTES_LENGTH_DRONE);
        assert_drone_equal(&drone, &decoded);
    }

    close(fd);
}

static void run_oversized_length(const char *address) {
    int fd = connect_to_server(address);
    struct Drone drone = {0};
    fill_drone(&drone, 0);

    unsigned char payload[BYTES_LENGTH_DRONE] = {0};
    size_t payload_len = EncodeDrone(&drone, payload);
    assert(payload_len == BYTES_LENGTH_DRONE);

    struct FrameHeader frame_header = {0};
    unsigned char header_buf[BYTES_LENGTH_FRAME_HEADER] = {0};
    size_t bad_len = BYTES_LENGTH_DRONE + 1;
    frame_header.magic = FRAME_MAGIC;
    frame_header.payload_type = PAYLOAD_TYPE_DRONE;
    frame_header.payload_length = (uint16_t)bad_len;
    assert(EncodeFrameHeader(&frame_header, header_buf) == BYTES_LENGTH_FRAME_HEADER);
    send_all(fd, header_buf, sizeof(header_buf));
    shutdown(fd, SHUT_WR);
    close(fd);
}

static void run_truncated_payload(const char *address) {
    int fd = connect_to_server(address);
    struct Drone drone = {0};
    fill_drone(&drone, 0);

    unsigned char payload[BYTES_LENGTH_DRONE] = {0};
    size_t payload_len = EncodeDrone(&drone, payload);
    assert(payload_len == BYTES_LENGTH_DRONE);

    struct FrameHeader frame_header = {0};
    unsigned char header_buf[BYTES_LENGTH_FRAME_HEADER] = {0};
    frame_header.magic = FRAME_MAGIC;
    frame_header.payload_type = PAYLOAD_TYPE_DRONE;
    frame_header.payload_length = (uint16_t)payload_len;
    assert(EncodeFrameHeader(&frame_header, header_buf) == BYTES_LENGTH_FRAME_HEADER);
    send_all(fd, header_buf, sizeof(header_buf));
    send_all(fd, payload, payload_len / 2);
    shutdown(fd, SHUT_WR);
    close(fd);
}

int main(int argc, char **argv) {
    signal(SIGPIPE, SIG_IGN);

    if (argc < 3) {
        fprintf(stderr, "usage: %s <host:port> <mode> [rounds] [seed]\n", argv[0]);
        return 2;
    }

    const char *address = argv[1];
    const char *mode = argv[2];
    int rounds = argc > 3 ? atoi(argv[3]) : 1;
    unsigned int seed = argc > 4 ? (unsigned int)strtoul(argv[4], NULL, 10)
                                 : (unsigned int)time(NULL);
    if (rounds <= 0) rounds = 1;

    if (strcmp(mode, "roundtrip") == 0) {
        run_roundtrip(address, rounds);
        printf("tcpip client roundtrip ok (%d rounds)\n", rounds);
        return 0;
    }

    if (strcmp(mode, "fragmented-roundtrip") == 0) {
        run_fragmented_roundtrip(address, rounds, seed);
        printf("tcpip client fragmented roundtrip ok (%d rounds, seed=%u)\n", rounds,
               seed);
        return 0;
    }

    if (strcmp(mode, "oversized-length") == 0) {
        run_oversized_length(address);
        printf("tcpip client sent oversized length\n");
        return 0;
    }

    if (strcmp(mode, "truncated-payload") == 0) {
        run_truncated_payload(address);
        printf("tcpip client sent truncated payload\n");
        return 0;
    }

    fprintf(stderr, "unknown mode: %s\n", mode);
    return 2;
}
