package config

// PluginConfig stores the type name of the plugin and raw configuration data
// for that plugin. Each plugin is responsible for decoding the raw map into its
// own concrete configuration struct.
type PluginConfig struct {
	Type string         `json:"type"`
	Conf map[string]any `json:"conf"`
}

// ComponentsConfig lists the pluggable components that make up the dispatch
// system. Each component is defined solely by its type and an arbitrary
// configuration map.
type ComponentsConfig struct {
	Dispatcher PluginConfig   `json:"dispatcher"`
	Fallback   PluginConfig   `json:"fallback"`
	Tuner      PluginConfig   `json:"tuner"`
	Prediction PluginConfig   `json:"prediction"`
	Metrics    []PluginConfig `json:"metrics"`
}
