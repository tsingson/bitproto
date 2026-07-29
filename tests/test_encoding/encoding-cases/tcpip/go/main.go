package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"sync"

	bp "github.com/hit9/bitproto/tests/test_encoding/encoding-cases/tcpip/go/bp"
	fh "github.com/hit9/bitproto/tests/test_encoding/encoding-cases/tcpip/go/framebp"
)

const frameMagic = byte(0xB1)

func readFull(conn net.Conn, buf []byte) error {
	_, err := io.ReadFull(conn, buf)
	return err
}

func writeFull(conn net.Conn, buf []byte) error {
	for len(buf) > 0 {
		n, err := conn.Write(buf)
		if err != nil {
			return err
		}
		buf = buf[n:]
	}
	return nil
}

func readFrame(conn net.Conn) ([]byte, error) {
	headerBuf := make([]byte, fh.BYTES_LENGTH_FRAME_HEADER)
	if err := readFull(conn, headerBuf); err != nil {
		return nil, fmt.Errorf("read header: %w", err)
	}
	header := &fh.FrameHeader{}
	header.Decode(headerBuf)
	if header.Magic != frameMagic {
		return nil, fmt.Errorf("bad magic %d", header.Magic)
	}
	if header.PayloadType != fh.PAYLOAD_TYPE_DRONE {
		return nil, fmt.Errorf("bad payload type %d", header.PayloadType)
	}
	payloadLen := int(header.PayloadLength)
	if payloadLen != int(bp.BYTES_LENGTH_DRONE) {
		return nil, fmt.Errorf("invalid payload length %d", payloadLen)
	}
	payload := make([]byte, payloadLen)
	if err := readFull(conn, payload); err != nil {
		return nil, fmt.Errorf("read payload: %w", err)
	}
	return payload, nil
}

func writeFrame(conn net.Conn, payload []byte) error {
	header := &fh.FrameHeader{
		Magic:         frameMagic,
		PayloadType:   fh.PAYLOAD_TYPE_DRONE,
		PayloadLength: uint16(len(payload)),
	}
	headerBuf := header.Encode()
	if len(headerBuf) != int(fh.BYTES_LENGTH_FRAME_HEADER) {
		return fmt.Errorf("unexpected header length %d", len(headerBuf))
	}
	if err := writeFull(conn, headerBuf); err != nil {
		return fmt.Errorf("write header: %w", err)
	}
	if err := writeFull(conn, payload); err != nil {
		return fmt.Errorf("write payload: %w", err)
	}
	return nil
}

func serveConnection(conn net.Conn, rounds int) error {
	defer conn.Close()

	for i := 0; i < rounds; i++ {
		payload, err := readFrame(conn)
		if err != nil {
			return fmt.Errorf("frame %d: %w", i, err)
		}

		drone := &bp.Drone{}
		drone.Decode(payload)
		reply := drone.Encode()
		if len(reply) != int(bp.BYTES_LENGTH_DRONE) {
			return fmt.Errorf("frame %d: unexpected reply length %d", i, len(reply))
		}
		if err := writeFrame(conn, reply); err != nil {
			return fmt.Errorf("frame %d: %w", i, err)
		}
	}

	return nil
}

func main() {
	log.SetFlags(0)
	listenAddr := flag.String("listen", "127.0.0.1:0", "listen address")
	rounds := flag.Int("rounds", 1, "number of frames to process")
	clients := flag.Int("clients", 1, "number of client connections to accept")
	flag.Parse()
	if *rounds <= 0 {
		log.Fatalf("invalid rounds %d", *rounds)
	}
	if *clients <= 0 {
		log.Fatalf("invalid clients %d", *clients)
	}

	ln, err := net.Listen("tcp", *listenAddr)
	if err != nil {
		log.Fatalf("listen: %v", err)
	}

	fmt.Printf("READY %s\n", ln.Addr().String())

	errCh := make(chan error, *clients)
	var wg sync.WaitGroup

	for c := 0; c < *clients; c++ {
		conn, acceptErr := ln.Accept()
		if acceptErr != nil {
			if errors.Is(acceptErr, net.ErrClosed) {
				break
			}
			log.Fatalf("accept: %v", acceptErr)
		}

		wg.Add(1)
		go func(conn net.Conn) {
			defer wg.Done()
			if serveErr := serveConnection(conn, *rounds); serveErr != nil {
				errCh <- serveErr
			}
		}(conn)
	}

	_ = ln.Close()
	wg.Wait()
	close(errCh)

	for serveErr := range errCh {
		if serveErr != nil {
			log.Fatalf("serve: %v", serveErr)
		}
	}

	fmt.Fprintf(os.Stdout, "DONE clients=%d rounds=%d\n", *clients, *rounds)
}
