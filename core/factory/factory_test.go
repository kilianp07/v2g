package factory

import "testing"

type sample struct{ A int }

type sampleConf struct {
	A int `json:"a"`
}

// Test registry registration and instantiation using Decode.
func TestRegistry_Create(t *testing.T) {
	reg := NewRegistry[*sample]()
	if err := reg.Register("s", func(conf map[string]any) (*sample, error) {
		var c sampleConf
		if err := Decode(conf, &c); err != nil {
			return nil, err
		}
		return &sample{A: c.A}, nil
	}); err != nil {
		t.Fatalf("register: %v", err)
	}
	inst, err := reg.Create(ModuleConfig{Type: "s", Conf: map[string]any{"a": 3}})
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if inst.A != 3 {
		t.Fatalf("expected 3 got %d", inst.A)
	}
}

// Test duplicate registration and unknown type errors.
func TestRegistry_Errors(t *testing.T) {
	reg := NewRegistry[int]()
	if err := reg.Register("x", func(map[string]any) (int, error) { return 1, nil }); err != nil {
		t.Fatalf("register: %v", err)
	}
	if err := reg.Register("x", nil); err == nil {
		t.Fatal("expected duplicate error")
	}
	if _, err := reg.Create(ModuleConfig{Type: "y"}); err == nil {
		t.Fatal("expected unknown type error")
	}
}
