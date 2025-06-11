package config

import (
	"crypto/rand"
	"encoding/base64"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"github.com/pkg/errors"
	"gopkg.in/yaml.v2"
)

type ManifestPlugin struct {
	Name string `yaml:"name"`
	Path string `yaml:"path"`
	Kind string `yaml:"kind"`
}

func generateRandomName() string {
	b := make([]byte, 6)
	if _, err := rand.Read(b); err != nil {
		// If crypto/rand fails, use a simple fallback
		return "unnamed_plugin"
	}
	return "plugin_" + base64.RawURLEncoding.EncodeToString(b)
}

func (p *ManifestPlugin) GetName() string {
	if p.Name != "" {
		return p.Name
	}
	if p.Path != "" {
		name := filepath.Base(p.Path)
		p.Name = name
		return name
	}

	name := generateRandomName()
	p.Name = name
	return name
}

func (p *ManifestPlugin) Validate() error {
	if p.Path == "" {
		return errors.New("plugin path cannot be empty")
	}

	if !filepath.IsAbs(p.Path) {
		if strings.Contains(p.Path, "..") && os.Getenv("GRPC_PLUGINS_ALLOW_RELATIVE_PATHS_DOUBLE_DOT") != "true" {
			return errors.New("plugin path cannot contain '..'")
		}
	}

	switch p.Kind {
	case "build_and_run":
		// Currently the only supported kind
		return nil
	case "":
		return errors.New("plugin kind cannot be empty")
	default:
		return errors.Errorf("unsupported plugin kind: %q", p.Kind)
	}
}

type TLSConfig struct {
	UseCustomTLS bool `yaml:"use_custom_tls"`
}

func (c *TLSConfig) Validate() error {
	// Currently no validation needed for TLS config
	return nil
}

type ManifestConfig struct {
	Plugins []ManifestPlugin `yaml:"plugins"`
	TLS     TLSConfig        `yaml:"tls"`
}

func (c *ManifestConfig) Validate() error {
	if len(c.Plugins) == 0 {
		return errors.New("manifest must contain at least one plugin")
	}

	seenNames := make(map[string]struct{})
	seenPaths := make(map[string]struct{})

	for i, plugin := range c.Plugins {
		if err := plugin.Validate(); err != nil {
			return errors.Wrapf(err, "invalid plugin at index %d", i)
		}

		name := plugin.GetName()
		if _, exists := seenNames[name]; exists {
			return errors.Errorf("duplicate plugin name %q", name)
		}
		seenNames[name] = struct{}{}

		absPath, err := filepath.Abs(plugin.Path)
		if err != nil {
			return errors.Wrapf(err, "failed to get absolute path for plugin %q", name)
		}
		if _, exists := seenPaths[absPath]; exists {
			return errors.Errorf("duplicate plugin path %q", absPath)
		}
		seenPaths[absPath] = struct{}{}
	}

	if err := c.TLS.Validate(); err != nil {
		return errors.Wrap(err, "invalid TLS configuration")
	}

	return nil
}

type Manifest struct {
	Kind   string          // can be "file" or "inline"
	Path   string          // if kind is "file", this is the path to the file
	Config *ManifestConfig // if kind is "inline", this is the config for the plugin
}

func (m *Manifest) Validate() error {
	switch m.Kind {
	case "file":
		if m.Path == "" {
			return errors.New("manifest path cannot be empty for kind 'file'")
		}
		if !filepath.IsAbs(m.Path) {
			if strings.Contains(m.Path, "..") {
				return errors.New("manifest path cannot contain '..'")
			}
		}
		return nil
	case "inline":
		if m.Config == nil {
			return errors.New("manifest config cannot be nil for kind 'inline'")
		}
		return nil
	case "":
		return errors.New("manifest kind cannot be empty")
	default:
		return errors.Errorf("unsupported manifest kind: %q", m.Kind)
	}
}

func LoadManifestFile[T any](cfg *Config[T]) (*ManifestConfig, error) {
	logger := slog.With("component", "config", "manifest_kind", "file", "path", cfg.Manifest.Path)
	logger.Debug("loading manifest from file")

	if err := cfg.Manifest.Validate(); err != nil {
		logger.Error("invalid manifest configuration", "error", err)
		return nil, errors.Wrap(err, "invalid manifest configuration")
	}

	configFile, err := os.ReadFile(cfg.Manifest.Path)
	if err != nil {
		logger.Error("failed to read manifest file", "error", err)
		return nil, errors.Wrapf(err, "failed to read manifest file at %s", cfg.Manifest.Path)
	}

	var pluginConfig ManifestConfig
	if err := yaml.Unmarshal(configFile, &pluginConfig); err != nil {
		logger.Error("failed to unmarshal manifest file", "error", err)
		return nil, errors.Wrapf(err, "failed to unmarshal manifest file at %s", cfg.Manifest.Path)
	}

	if err := pluginConfig.Validate(); err != nil {
		logger.Error("invalid manifest configuration", "error", err)
		return nil, errors.Wrap(err, "invalid manifest configuration")
	}

	logger.Info("manifest file loaded successfully",
		"plugin_count", len(pluginConfig.Plugins),
		"use_custom_tls", pluginConfig.TLS.UseCustomTLS)
	return &pluginConfig, nil
}

func LoadManifestInline[T any](cfg *Config[T]) (*ManifestConfig, error) {
	logger := slog.With("component", "config", "manifest_kind", "inline")
	logger.Debug("loading inline manifest")

	if err := cfg.Manifest.Validate(); err != nil {
		logger.Error("invalid manifest configuration", "error", err)
		return nil, errors.Wrap(err, "invalid manifest configuration")
	}

	if err := cfg.Manifest.Config.Validate(); err != nil {
		logger.Error("invalid manifest configuration", "error", err)
		return nil, errors.Wrap(err, "invalid manifest configuration")
	}

	logger.Info("inline manifest loaded successfully",
		"plugin_count", len(cfg.Manifest.Config.Plugins),
		"use_custom_tls", cfg.Manifest.Config.TLS.UseCustomTLS)
	return cfg.Manifest.Config, nil
}

func LoadManifest[T any](cfg *Config[T]) (*ManifestConfig, error) {
	logger := slog.With("component", "config")
	logger.Debug("loading manifest", "kind", cfg.Manifest.Kind)

	if err := cfg.Manifest.Validate(); err != nil {
		logger.Error("invalid manifest configuration", "error", err)
		return nil, errors.Wrap(err, "invalid manifest configuration")
	}

	var result *ManifestConfig
	var err error

	switch cfg.Manifest.Kind {
	case "inline":
		result, err = LoadManifestInline(cfg)
	case "file":
		result, err = LoadManifestFile(cfg)
	default:
		// This should never happen due to Validate() check above
		logger.Error("unsupported manifest kind", "kind", cfg.Manifest.Kind)
		return nil, errors.Errorf("manifest kind %q is not supported", cfg.Manifest.Kind)
	}

	if err != nil {
		logger.Error("failed to load manifest", "error", err)
		return nil, errors.Wrapf(err, "failed to load manifest of kind %s", cfg.Manifest.Kind)
	}

	return result, nil
}
