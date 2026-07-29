package native

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRenderGoAndPy(t *testing.T) {
	proto, err := ParseFile(filepath.Join("..", "..", "example", "example.bitproto"))
	if err != nil {
		t.Fatalf("ParseFile failed: %v", err)
	}
	dir := t.TempDir()
	goPath, err := RenderGo(proto, dir)
	if err != nil {
		t.Fatalf("RenderGo failed: %v", err)
	}
	pyPath, err := RenderPy(proto, dir)
	if err != nil {
		t.Fatalf("RenderPy failed: %v", err)
	}
	gb, err := os.ReadFile(goPath)
	if err != nil {
		t.Fatalf("read go failed: %v", err)
	}
	pb, err := os.ReadFile(pyPath)
	if err != nil {
		t.Fatalf("read py failed: %v", err)
	}
	if !strings.Contains(string(gb), "package drone") {
		t.Fatalf("unexpected go package")
	}
	if !strings.Contains(string(gb), "type Drone struct") {
		t.Fatalf("missing go Drone struct")
	}
	if !strings.Contains(string(pb), "class Drone(bp.MessageBase):") {
		t.Fatalf("missing py Drone class")
	}
	if !strings.Contains(string(pb), "BYTES_LENGTH_DRONE") {
		t.Fatalf("missing py BYTES_LENGTH_DRONE")
	}
}
