package native

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRenderFixedFloatTypes(t *testing.T) {
	proto, err := ParseString(`
proto telem

message Sample {
    float a = 1
    double b = 2
    float[2] c = 3
}
`)
	if err != nil {
		t.Fatalf("ParseString failed: %v", err)
	}

	dir := t.TempDir()
	hPath, cPath, err := RenderC(proto, dir)
	if err != nil {
		t.Fatalf("RenderC failed: %v", err)
	}
	goPath, err := RenderGo(proto, dir)
	if err != nil {
		t.Fatalf("RenderGo failed: %v", err)
	}
	pyPath, err := RenderPy(proto, dir)
	if err != nil {
		t.Fatalf("RenderPy failed: %v", err)
	}

	hb, err := os.ReadFile(hPath)
	if err != nil {
		t.Fatalf("read header failed: %v", err)
	}
	cb, err := os.ReadFile(cPath)
	if err != nil {
		t.Fatalf("read source failed: %v", err)
	}
	gb, err := os.ReadFile(goPath)
	if err != nil {
		t.Fatalf("read go failed: %v", err)
	}
	pb, err := os.ReadFile(pyPath)
	if err != nil {
		t.Fatalf("read py failed: %v", err)
	}

	hs := string(hb)
	cs := string(cb)
	gs := string(gb)
	ps := string(pb)

	libGo, err := os.ReadFile(filepath.Join("..", "..", "lib", "go", "bitproto.go"))
	if err != nil {
		t.Fatalf("read lib/go failed: %v", err)
	}
	libCHeader, err := os.ReadFile(filepath.Join("..", "..", "lib", "c", "bitproto.h"))
	if err != nil {
		t.Fatalf("read lib/c header failed: %v", err)
	}
	libCSource, err := os.ReadFile(filepath.Join("..", "..", "lib", "c", "bitproto.c"))
	if err != nil {
		t.Fatalf("read lib/c source failed: %v", err)
	}

	if !strings.Contains(hs, "int32_t a;") || !strings.Contains(hs, "int32_t b;") {
		t.Fatalf("float/double fields must be int32_t in generated C header")
	}
	if strings.Contains(hs, "#define BP_FLOAT_SCALE 100000000.0") || strings.Contains(cs, "BpFloatToInt32(double v)") {
		t.Fatalf("C fixed-point helpers should live in lib/c")
	}
	if !strings.Contains(string(libCHeader), "#define BP_FLOAT_SCALE 100000000.0") {
		t.Fatalf("missing lib/c fixed-point scale helper")
	}
	if !strings.Contains(string(libCHeader), "int32_t BpFloatToInt32(double v);") || !strings.Contains(string(libCHeader), "double BpInt32ToFloat(int32_t v);") {
		t.Fatalf("missing lib/c fixed-point declarations")
	}
	if !strings.Contains(string(libCSource), "int32_t BpFloatToInt32(double v)") || !strings.Contains(string(libCSource), "double BpInt32ToFloat(int32_t v)") {
		t.Fatalf("missing lib/c fixed-point definitions")
	}
	if !strings.Contains(hs, "SetSampleAFloat") || !strings.Contains(hs, "GetSampleAFloat") {
		t.Fatalf("missing C scalar float helper APIs")
	}
	if !strings.Contains(hs, "SetSampleCFloatAt") || !strings.Contains(hs, "GetSampleCFloatAt") {
		t.Fatalf("missing C array float helper APIs")
	}
	if !strings.Contains(cs, "size_t EncodeSample") {
		t.Fatalf("missing EncodeSample in C source")
	}

	if !strings.Contains(gs, "A int32") || !strings.Contains(gs, "B int32") {
		t.Fatalf("float/double fields must be int32 in generated Go struct")
	}
	if strings.Contains(gs, "func BpFloatToInt32") || strings.Contains(gs, "func BpInt32ToFloat") {
		t.Fatalf("Go fixed-point conversion helpers should live in lib/go")
	}
	if !strings.Contains(gs, "bp.BpFloatToInt32") || !strings.Contains(gs, "bp.BpInt32ToFloat") {
		t.Fatalf("missing Go fixed-point conversion helper calls")
	}
	if !strings.Contains(gs, "func (m *Sample) SetAFloat") || !strings.Contains(gs, "func (m *Sample) GetAFloat") {
		t.Fatalf("missing Go scalar float helper APIs")
	}
	if !strings.Contains(gs, "func (m *Sample) SetCFloatAt") || !strings.Contains(gs, "func (m *Sample) GetCFloatAt") {
		t.Fatalf("missing Go array float helper APIs")
	}

	if !strings.Contains(ps, "self.a = 0") || !strings.Contains(ps, "self.b = 0") {
		t.Fatalf("float/double fields must be int in generated Python class")
	}
	if !strings.Contains(ps, "BP_FLOAT_SCALE = 100000000.0") {
		t.Fatalf("missing Python fixed-point scale helper")
	}
	if !strings.Contains(ps, "def bp_float_to_int32") || !strings.Contains(ps, "def bp_int32_to_float") {
		t.Fatalf("missing Python fixed-point conversion helpers")
	}
	if !strings.Contains(ps, "def set_a_float") || !strings.Contains(ps, "def get_a_float") {
		t.Fatalf("missing Python scalar float helper APIs")
	}
	if !strings.Contains(ps, "def set_c_float_at") || !strings.Contains(ps, "def get_c_float_at") {
		t.Fatalf("missing Python array float helper APIs")
	}
	if !strings.Contains(string(libGo), "func BpFloatToInt32") || !strings.Contains(string(libGo), "func BpInt32ToFloat") {
		t.Fatalf("missing lib/go fixed-point helpers")
	}

	if !strings.Contains(hPath, filepath.Join(dir, "telem_bp.h")) {
		t.Fatalf("unexpected output header path: %s", hPath)
	}
}

func TestRenderConstDefinitions(t *testing.T) {
	proto, err := ParseString(`
proto sample

const MAX_VALUE = 32
const ENABLED = true
const NAME = "bitproto"
`)
	if err != nil {
		t.Fatalf("ParseString failed: %v", err)
	}

	dir := t.TempDir()
	hPath, cPath, err := RenderC(proto, dir)
	if err != nil {
		t.Fatalf("RenderC failed: %v", err)
	}
	goPath, err := RenderGo(proto, dir)
	if err != nil {
		t.Fatalf("RenderGo failed: %v", err)
	}

	hb, err := os.ReadFile(hPath)
	if err != nil {
		t.Fatalf("read header failed: %v", err)
	}
	cb, err := os.ReadFile(cPath)
	if err != nil {
		t.Fatalf("read source failed: %v", err)
	}
	gb, err := os.ReadFile(goPath)
	if err != nil {
		t.Fatalf("read go failed: %v", err)
	}

	hs := string(hb)
	cs := string(cb)
	gs := string(gb)

	if !strings.Contains(hs, "#ifndef __BITPROTO_NATIVE_SAMPLE_H__") {
		t.Fatalf("missing C header guard")
	}
	if !strings.Contains(hs, "#define MAX_VALUE 32") {
		t.Fatalf("missing C const macro")
	}
	if !strings.Contains(gs, "const MAX_VALUE int = 32") {
		t.Fatalf("missing Go const declaration")
	}
	if !strings.Contains(gs, "const ENABLED bool = true") {
		t.Fatalf("missing Go bool const declaration")
	}
	if !strings.Contains(gs, "const NAME string = \"bitproto\"") {
		t.Fatalf("missing Go string const declaration")
	}
	if cs == "" {
		t.Fatalf("generated C source should still be present")
	}
	_ = cs
}
