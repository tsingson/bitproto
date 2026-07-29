package native

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRenderC(t *testing.T) {
	proto, err := ParseFile(filepath.Join("..", "..", "example", "example.bitproto"))
	if err != nil {
		t.Fatalf("ParseFile failed: %v", err)
	}
	dir := t.TempDir()
	h, c, err := RenderC(proto, dir)
	if err != nil {
		t.Fatalf("RenderC failed: %v", err)
	}
	hb, err := os.ReadFile(h)
	if err != nil {
		t.Fatalf("read h failed: %v", err)
	}
	cb, err := os.ReadFile(c)
	if err != nil {
		t.Fatalf("read c failed: %v", err)
	}
	hs := string(hb)
	cs := string(cb)
	if !strings.Contains(hs, "struct Drone {") {
		t.Fatalf("missing struct Drone in header")
	}
	if !strings.Contains(hs, "#define BYTES_LENGTH_DRONE") {
		t.Fatalf("missing BYTES_LENGTH_DRONE in header")
	}
	if !strings.Contains(cs, "size_t EncodeDrone") {
		t.Fatalf("missing EncodeDrone stub in source")
	}
}
