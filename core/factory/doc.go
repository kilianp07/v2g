// Package factory provides a small generic registry used to instantiate modules
// from configuration. Modules are defined by a type string and a map of raw
// settings. Factories decode the settings into typed structs and return the
// concrete implementation.
//
// Example usage:
//
//	reg := factory.NewRegistry[io.Reader]()
//	reg.Register("file", func(conf map[string]any) (io.Reader, error) {
//	    var c struct{ Path string `json:"path"` }
//	    if err := factory.Decode(conf, &c); err != nil {
//	        return nil, err
//	    }
//	    return os.Open(c.Path)
//	})
//	r, err := reg.Create(factory.ModuleConfig{Type: "file", Conf: map[string]any{"path": "foo"}})
package factory
