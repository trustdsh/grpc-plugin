package pluginsloader

import (
	"log/slog"
	"sync"

	"github.com/pkg/errors"

	"github.com/trustdsh/grpc-plugin/internal/transport"
	"github.com/trustdsh/grpc-plugin/runner/internal/pluginrunner"
	"github.com/trustdsh/grpc-plugin/runner/internal/pluginrunner/portmanager"
)

// LoadedPlugins represents a collection of loaded plugins with their associated resources
type LoadedPlugins[T any] struct {
	pluginsMap         map[string]*pluginrunner.LoadedPlugin[T]
	TransportGenerator *transport.TransportGenerator
	logger             *slog.Logger
	portManager        *portmanager.PortManager
	mu                 sync.RWMutex
}

// Close shuts down all loaded plugins and releases their resources
func (l *LoadedPlugins[T]) Close() error {
	// Snapshot and clear under lock
	l.mu.Lock()
	pluginsCopy := make(map[string]*pluginrunner.LoadedPlugin[T], len(l.pluginsMap))
	for k, v := range l.pluginsMap {
		pluginsCopy[k] = v
	}
	l.mu.Unlock()

	l.logger.Debug("closing all plugins")
	var lastErr error
	for name, plugin := range pluginsCopy {
		pluginLogger := l.logger.With("plugin", name)
		pluginLogger.Debug("closing plugin")

		if err := plugin.Close(); err != nil {
			pluginLogger.Error("failed to close plugin", "error", err)
			lastErr = err
		}

		if plugin.Server != nil && plugin.Server.Port != 0 {
			if err := l.portManager.ReleasePort(plugin.Server.Port); err != nil {
				pluginLogger.Error("failed to release port", "port", plugin.Server.Port, "error", err)
			}
		}

		pluginLogger.Debug("plugin closed successfully")
	}

	if lastErr != nil {
		return errors.Wrap(lastErr, "failed to close one or more plugins")
	}

	l.logger.Info("all plugins closed successfully")
	return nil
}

// GetPlugin retrieves a plugin by name and returns its interface
func (l *LoadedPlugins[T]) GetPlugin(name string) (T, error) {
	l.mu.RLock()
	defer l.mu.RUnlock()

	l.logger.Debug("getting plugin", "plugin", name)
	plugin, ok := l.pluginsMap[name]
	var nilt T
	if !ok {
		l.logger.Error("plugin not found", "plugin", name)
		return nilt, errors.Errorf("plugin %q not found", name)
	}
	l.logger.Debug("plugin found", "plugin", name)
	return plugin.Plugin, nil
}

// GetAllPlugins returns a slice of all loaded plugin interfaces
func (l *LoadedPlugins[T]) GetAllPlugins() []T {
	l.mu.RLock()
	defer l.mu.RUnlock()

	l.logger.Debug("getting all plugins")
	plugins := make([]T, 0, len(l.pluginsMap))
	for name, plugin := range l.pluginsMap {
		l.logger.Debug("adding plugin to list", "plugin", name)
		plugins = append(plugins, plugin.Plugin)
	}
	l.logger.Debug("all plugins retrieved", "count", len(plugins))
	return plugins
}

// GetRawPlugin retrieves a plugin by name and returns its raw LoadedPlugin instance
func (l *LoadedPlugins[T]) GetRawPlugin(name string) (*pluginrunner.LoadedPlugin[T], error) {
	l.mu.RLock()
	defer l.mu.RUnlock()

	l.logger.Debug("getting raw plugin", "plugin", name)
	plugin, ok := l.pluginsMap[name]
	if !ok {
		l.logger.Error("plugin not found", "plugin", name)
		return nil, errors.Errorf("plugin %q not found", name)
	}
	l.logger.Debug("raw plugin found", "plugin", name)
	return plugin, nil
}

// GetAllRawPlugins returns a slice of all loaded plugin instances
func (l *LoadedPlugins[T]) GetAllRawPlugins() []*pluginrunner.LoadedPlugin[T] {
	l.mu.RLock()
	defer l.mu.RUnlock()

	l.logger.Debug("getting all raw plugins")
	plugins := make([]*pluginrunner.LoadedPlugin[T], 0, len(l.pluginsMap))
	for name, plugin := range l.pluginsMap {
		l.logger.Debug("adding raw plugin to list", "plugin", name)
		plugins = append(plugins, plugin)
	}
	l.logger.Debug("all raw plugins retrieved", "count", len(plugins))
	return plugins
}
