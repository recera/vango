package config

import (
	"encoding/json"
	"os"
	"path/filepath"
)

// Config represents the vango.json configuration
type Config struct {
	// Styling configuration
	Styling *StylingConfig `json:"styling,omitempty"`

	// Routing configuration
	RoutesDir string `json:"routesDir,omitempty"`

	// Build configuration
	WasmTarget string `json:"wasmTarget,omitempty"`

	// PWA configuration
	PWA *PWAConfig `json:"pwa,omitempty"`

	// Development server configuration
	Dev *DevConfig `json:"dev,omitempty"`
}

// StylingConfig contains styling-related configuration
type StylingConfig struct {
	// Tailwind configuration
	Tailwind *TailwindConfig `json:"tailwind,omitempty"`

	// CSS configuration
	CSS *CSSConfig `json:"css,omitempty"`
}

// TailwindConfig contains Tailwind-specific configuration
type TailwindConfig struct {
	// Whether Tailwind is enabled
	Enabled bool `json:"enabled"`

	// Path to tailwind.config.js
	ConfigPath string `json:"config,omitempty"`

	// Path to input CSS file
	InputPath string `json:"input,omitempty"`

	// Path to output CSS file
	OutputPath string `json:"output,omitempty"`

	// Whether to watch for changes
	Watch bool `json:"watch"`

	// Whether to minify output
	Minify bool `json:"minify,omitempty"`

	// Strategy controls how Tailwind is executed: "auto" | "npm" | "standalone" | "vendor"
	Strategy string `json:"strategy,omitempty"`

	// Version to use for standalone downloads (e.g., "3.4.0")
	Version string `json:"version,omitempty"`

	// Whether to auto-download the standalone binary if missing
	AutoDownload bool `json:"autoDownload,omitempty"`
}

// CSSConfig contains CSS-related configuration
type CSSConfig struct {
	// Path to global styles directory
	StylesDir string `json:"stylesDir,omitempty"`

	// Whether to enable CSS modules
	Modules bool `json:"modules,omitempty"`

	// Whether to enable PostCSS
	PostCSS bool `json:"postCSS,omitempty"`
}

// PWAConfig contains PWA-related configuration
type PWAConfig struct {
	// Whether PWA is enabled
	Enabled bool `json:"enabled"`

	// Path to manifest.json
	ManifestPath string `json:"manifest,omitempty"`

	// Path to service worker
	ServiceWorkerPath string `json:"serviceWorker,omitempty"`
}

// DevConfig contains development server configuration
type DevConfig struct {
	// Server port
	Port int `json:"port,omitempty"`

	// Server host
	Host string `json:"host,omitempty"`

	// Whether to open browser on start
	Open bool `json:"open,omitempty"`

	// Proxy configuration
	Proxy map[string]string `json:"proxy,omitempty"`

	// Whether to enable HTTPS
	HTTPS bool `json:"https,omitempty"`

	// Path to SSL certificate
	CertPath string `json:"cert,omitempty"`

	// Path to SSL key
	KeyPath string `json:"key,omitempty"`
}

// Load loads configuration from vango.json
func Load(projectPath string) (*Config, error) {
	// Try to find vango.json
	configPath := filepath.Join(projectPath, "vango.json")

	// Check if config file exists
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		// Return default config if no file exists
		return DefaultConfig(), nil
	}

	// Read config file
	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, err
	}

	// Parse JSON
	var config Config
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, err
	}

	// Apply defaults for missing values
	applyDefaults(&config)

	return &config, nil
}

// Save saves configuration to vango.json
func Save(config *Config, projectPath string) error {
	configPath := filepath.Join(projectPath, "vango.json")

	// Marshal to JSON with indentation
	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return err
	}

	// Write to file
	return os.WriteFile(configPath, data, 0644)
}

// DefaultConfig returns the default configuration
func DefaultConfig() *Config {
	return &Config{
		RoutesDir:  "app/routes",
		WasmTarget: "wasm",
		Styling: &StylingConfig{
			Tailwind: &TailwindConfig{
				Enabled:    false,
				ConfigPath: "tailwind.config.js",
				InputPath:  "app/styles/input.css",
				OutputPath: "public/styles.css",
				Watch:      true,
				Minify:     false,
			},
			CSS: &CSSConfig{
				StylesDir: "app/styles",
				Modules:   false,
				PostCSS:   false,
			},
		},
		PWA: &PWAConfig{
			Enabled:           false,
			ManifestPath:      "public/manifest.json",
			ServiceWorkerPath: "public/sw.js",
		},
		Dev: &DevConfig{
			Port:  8080,
			Host:  "localhost",
			Open:  false,
			Proxy: make(map[string]string),
			HTTPS: false,
		},
	}
}

// applyDefaults applies default values to missing configuration
func applyDefaults(config *Config) {
	defaults := DefaultConfig()

	// Apply route directory default
	if config.RoutesDir == "" {
		config.RoutesDir = defaults.RoutesDir
	}

	// Apply WASM target default
	if config.WasmTarget == "" {
		config.WasmTarget = defaults.WasmTarget
	}

	// Apply styling defaults
	if config.Styling == nil {
		config.Styling = defaults.Styling
	} else {
		if config.Styling.Tailwind == nil {
			config.Styling.Tailwind = defaults.Styling.Tailwind
		} else {
			// Apply Tailwind defaults
			if config.Styling.Tailwind.ConfigPath == "" {
				config.Styling.Tailwind.ConfigPath = defaults.Styling.Tailwind.ConfigPath
			}
			if config.Styling.Tailwind.InputPath == "" {
				config.Styling.Tailwind.InputPath = defaults.Styling.Tailwind.InputPath
			}
			if config.Styling.Tailwind.OutputPath == "" {
				config.Styling.Tailwind.OutputPath = defaults.Styling.Tailwind.OutputPath
			}
		}

		if config.Styling.CSS == nil {
			config.Styling.CSS = defaults.Styling.CSS
		} else {
			// Apply CSS defaults
			if config.Styling.CSS.StylesDir == "" {
				config.Styling.CSS.StylesDir = defaults.Styling.CSS.StylesDir
			}
		}
	}

	// Apply PWA defaults
	if config.PWA == nil {
		config.PWA = defaults.PWA
	} else {
		if config.PWA.ManifestPath == "" {
			config.PWA.ManifestPath = defaults.PWA.ManifestPath
		}
		if config.PWA.ServiceWorkerPath == "" {
			config.PWA.ServiceWorkerPath = defaults.PWA.ServiceWorkerPath
		}
	}

	// Apply dev server defaults
	if config.Dev == nil {
		config.Dev = defaults.Dev
	} else {
		if config.Dev.Port == 0 {
			config.Dev.Port = defaults.Dev.Port
		}
		if config.Dev.Host == "" {
			config.Dev.Host = defaults.Dev.Host
		}
		if config.Dev.Proxy == nil {
			config.Dev.Proxy = make(map[string]string)
		}
	}
}

// Validate validates the configuration
func (c *Config) Validate() error {
	// Add validation logic here if needed
	return nil
}

// GetTailwindConfig returns Tailwind configuration for use with the runner
func (c *Config) GetTailwindConfig() (configPath, inputPath, outputPath string, watch bool) {
	if c.Styling != nil && c.Styling.Tailwind != nil {
		tw := c.Styling.Tailwind
		return tw.ConfigPath, tw.InputPath, tw.OutputPath, tw.Watch
	}

	// Return defaults
	defaults := DefaultConfig()
	tw := defaults.Styling.Tailwind
	return tw.ConfigPath, tw.InputPath, tw.OutputPath, tw.Watch
}
