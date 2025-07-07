package factory

import (
	"fmt"
	"sync"

	"github.com/mitchellh/mapstructure"
)

// ModuleConfig contains the type name and raw configuration for a module.
type ModuleConfig struct {
	Type string         `json:"type"`
	Conf map[string]any `json:"conf"`
}

// Factory constructs an implementation of T using the provided raw config.
type Factory[T any] func(map[string]any) (T, error)

// Registry stores factories keyed by module type.
type Registry[T any] struct {
	mu        sync.RWMutex
	factories map[string]Factory[T]
}

// NewRegistry returns an empty factory registry.
func NewRegistry[T any]() *Registry[T] {
	return &Registry[T]{factories: make(map[string]Factory[T])}
}

// Register adds a factory for the given type name.
func (r *Registry[T]) Register(name string, f Factory[T]) error {
	if f == nil {
		return fmt.Errorf("factory nil for %s", name)
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.factories[name]; ok {
		return fmt.Errorf("factory already registered for %s", name)
	}
	r.factories[name] = f
	return nil
}

// Create instantiates a module based on its configuration.
func (r *Registry[T]) Create(cfg ModuleConfig) (T, error) {
	r.mu.RLock()
	f, ok := r.factories[cfg.Type]
	r.mu.RUnlock()
	if !ok {
		var zero T
		return zero, fmt.Errorf("unknown module type %s", cfg.Type)
	}
	return f(cfg.Conf)
}

// Decode fills out the provided struct using json tags.
func Decode(data map[string]any, out any) error {
	dec, err := mapstructure.NewDecoder(&mapstructure.DecoderConfig{TagName: "json", Result: out})
	if err != nil {
		return err
	}
	return dec.Decode(data)
}
