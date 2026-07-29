package native

import (
	"path/filepath"
	"testing"
)

func TestParseExampleProto(t *testing.T) {
	path := filepath.Join("..", "..", "example", "example.bitproto")
	proto, err := ParseFile(path)
	if err != nil {
		t.Fatalf("ParseFile failed: %v", err)
	}
	if proto.Name != "drone" {
		t.Fatalf("unexpected proto name %q", proto.Name)
	}
	if len(proto.Messages) == 0 {
		t.Fatalf("expected messages")
	}
}

func TestParseDroneCaseProto(t *testing.T) {
	path := filepath.Join("..", "..", "tests", "test_encoding", "encoding-cases", "drone", "drone.bitproto")
	proto, err := ParseFile(path)
	if err != nil {
		t.Fatalf("ParseFile failed: %v", err)
	}
	if proto.Name != "drone" {
		t.Fatalf("unexpected proto name %q", proto.Name)
	}
	if len(proto.Enums) == 0 {
		t.Fatalf("expected enums")
	}
}
