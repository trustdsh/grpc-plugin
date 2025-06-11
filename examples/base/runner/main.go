package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	shared "github.com/trustdsh/grpc-plugin/examples/base/shared"
	"github.com/trustdsh/grpc-plugin/pkgs/config"
	runner "github.com/trustdsh/grpc-plugin/runner"
)

func main() {
	// Configure plugin runner
	logLevel := slog.LevelInfo

	// Setup logger
	handlerOpts := &slog.HandlerOptions{
		Level: logLevel,
	}
	logger := slog.New(slog.NewTextHandler(os.Stdout, handlerOpts))
	slog.SetDefault(logger)

	// Create a context that will be cancelled on SIGTERM/SIGINT
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Handle shutdown signals
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGTERM, syscall.SIGINT)
	go func() {
		sig := <-sigChan
		slog.Info("received shutdown signal", "signal", sig)
		cancel()
	}()

	cfg := config.Config[shared.PluginClient]{
		LoggerOptions: &config.LoggerOptions{
			Type:  "text",
			Level: &logLevel,
			Attributes: []slog.Attr{
				slog.String("run by", "runner"),
			},
		},
		PluginGenerator: shared.NewPluginClient,
		Manifest: &config.Manifest{
			Kind: "file",
			Path: "./plugins.yml",
		},
	}

	// Load plugins
	pluginsRaw, err := runner.LoadAll(ctx, cfg)
	if err != nil {
		slog.Error("failed to load plugins", "error", err)
		os.Exit(1)
	}
	defer pluginsRaw.Close()

	plugins := pluginsRaw.GetAllPlugins()
	slog.Info("plugins loaded", "count", len(plugins))

	// Example: Call each plugin with different inputs
	for i, plugin := range plugins {
		// Set a timeout for each operation
		opCtx, opCancel := context.WithTimeout(ctx, 5*time.Second)

		// Try DoSomething
		if _, err := plugin.DoSomething(opCtx, &shared.Empty{}); err != nil {
			slog.Error("failed to do something", "plugin_index", i, "error", err)
		}

		opCancel()
	}
}
