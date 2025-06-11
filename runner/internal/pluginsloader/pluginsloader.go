package pluginsloader

import (
	"context"
	"log/slog"

	"github.com/pkg/errors"
	"github.com/trustdsh/grpc-plugin/internal/transport"
	"github.com/trustdsh/grpc-plugin/pkgs/config"
	"github.com/trustdsh/grpc-plugin/runner/internal/pluginrunner"
	"github.com/trustdsh/grpc-plugin/runner/internal/pluginrunner/portmanager"
)

// LoadAll loads all plugins defined in the configuration
func LoadAll[T any](ctx context.Context, cfg config.Config[T]) (*LoadedPlugins[T], error) {
	logger := slog.With("component", "plugins_loader")
	logger.Debug("starting plugins loading")

	pluginConfig, err := config.LoadManifest(&cfg)
	if err != nil {
		logger.Error("failed to load manifest", "error", err)
		return nil, errors.Wrap(err, "failed to load plugin manifest")
	}
	logger.Debug("manifest loaded successfully", "plugin_count", len(pluginConfig.Plugins))

	transportGenerator, err := transport.NewTransportGenerator(&pluginConfig.TLS)
	if err != nil {
		logger.Error("failed to create transport generator", "error", err)
		return nil, errors.Wrap(err, "failed to create transport generator")
	}
	logger.Debug("transport generator created successfully")

	portMgr := portmanager.New()

	plugins := &LoadedPlugins[T]{
		pluginsMap:         make(map[string]*pluginrunner.LoadedPlugin[T]),
		TransportGenerator: transportGenerator,
		logger:             logger,
		portManager:        portMgr,
	}

	// If loading fails, ensure we clean up any loaded plugins
	var loadErr error
	defer func() {
		if loadErr != nil {
			if err := plugins.Close(); err != nil {
				logger.Error("failed to clean up plugins after load error", "error", err)
			}
		}
	}()

	for _, pluginConfig := range pluginConfig.Plugins {
		select {
		case <-ctx.Done():
			loadErr = errors.Wrap(ctx.Err(), "plugin loading cancelled")
			return nil, loadErr
		default:
			pluginLogger := logger.With("plugin", pluginConfig.GetName())
			pluginLogger.Debug("loading plugin", "path", pluginConfig.Path, "kind", pluginConfig.Kind)

			plugin, err := pluginrunner.LoadPlugin(ctx, pluginConfig, transportGenerator, &cfg, portMgr)
			if err != nil {
				pluginLogger.Error("failed to load plugin", "error", err)
				loadErr = errors.Wrapf(err, "failed to load plugin %s", pluginConfig.GetName())
				return nil, loadErr
			}

			plugins.pluginsMap[pluginConfig.GetName()] = plugin
			pluginLogger.Info("plugin loaded successfully")
		}
	}

	logger.Info("all plugins loaded successfully", "plugin_count", len(plugins.pluginsMap))
	return plugins, nil
}
