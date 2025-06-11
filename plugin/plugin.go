package plugin

import (
	"context"
	"flag"
	"log/slog"
	"net"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/trustdsh/grpc-plugin/internal/transport"
	"github.com/trustdsh/grpc-plugin/pkgs/config"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

const (
	shutdownTimeout = 5 * time.Second
)

type PluginOptions struct {
	Logger *slog.Logger
	Server *grpc.Server
}

type Plugin interface {
	Start(PluginOptions)
}

func parseAndSetLoggerOptions(rawLoggerOptions string) {
	if rawLoggerOptions == "" {
		return
	}
	loggerOptions := &config.LoggerOptions{}
	if err := loggerOptions.UnmarshalJSON([]byte(rawLoggerOptions)); err != nil {
		slog.Error("failed to unmarshal logger options", "error", err)
		return
	}

	handlerOptions := &slog.HandlerOptions{}
	if loggerOptions.Level != nil {
		handlerOptions.Level = loggerOptions.Level
	}

	switch loggerOptions.Type {
	case "text":
		slog.SetDefault(slog.New(slog.NewTextHandler(os.Stdout, handlerOptions)))
	case "json":
		slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, handlerOptions)))
	default:
		slog.SetDefault(slog.New(slog.NewTextHandler(os.Stdout, handlerOptions)))
		slog.Warn("no logger type specified, using text")
	}

	for _, attr := range loggerOptions.Attributes {
		slog.SetDefault(slog.Default().With(attr))
	}
}

func parseAndSetLoggerOptionsAndPluginName(pluginName string, rawLoggerOptions string) {
	parseAndSetLoggerOptions(rawLoggerOptions)

	if pluginName != "" {
		slog.SetDefault(slog.Default().With("plugin", pluginName))
	}
}

func StartPlugin(plugin Plugin) {
	var (
		port          = flag.Int("port", 50051, "The server port")
		tlsKeyAndCert = flag.String("tls_key_and_cert", "{}", "The server tls key and cert")
		pluginName    = flag.String("plugin_name", "", "The name of the plugin")
		loggerOptions = flag.String("logger_options", "", "The logger options")
	)

	flag.Parse()

	parseAndSetLoggerOptionsAndPluginName(*pluginName, *loggerOptions)

	logger := slog.Default().With("component", "plugin")
	logger.Debug("starting plugin initialization")

	// Create a context that will be canceled on SIGTERM/SIGINT
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Handle shutdown signals
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGTERM, syscall.SIGINT)
	go func() {
		sig := <-sigChan
		logger.Info("received shutdown signal", "signal", sig)
		cancel()
	}()

	keyAndCert, err := transport.DeserializeKeyAndCert([]byte(*tlsKeyAndCert))
	if err != nil {
		logger.Error("failed to deserialize tls key and cert", "error", err)
		return
	}
	logger.Debug("tls key and cert deserialized successfully")

	lis, err := net.Listen("tcp", net.JoinHostPort("", strconv.Itoa(*port)))
	if err != nil {
		logger.Error("failed to listen", "error", err, "port", *port)
		return
	}
	logger.Info("server listening", "port", *port)

	tlsConfig, err := keyAndCert.GetTLSConfig()
	if err != nil {
		logger.Error("failed to get tls config", "error", err)
		return
	}
	logger.Debug("tls config created successfully")

	s := grpc.NewServer(grpc.Creds(credentials.NewTLS(tlsConfig)))

	plugin.Start(PluginOptions{
		Logger: logger,
		Server: s,
	})

	// Start server in a goroutine
	go func() {
		logger.Info("starting grpc server")
		if err := s.Serve(lis); err != nil {
			logger.Error("failed to serve", "error", err)
			cancel()
		}
	}()

	// Wait for shutdown signal
	<-ctx.Done()

	// Initiate graceful shutdown
	logger.Info("initiating graceful shutdown")
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), shutdownTimeout)
	defer shutdownCancel()

	// Create a channel to signal completion of graceful stop
	stopped := make(chan struct{})
	go func() {
		s.GracefulStop()
		close(stopped)
	}()

	// Wait for graceful stop or timeout
	select {
	case <-shutdownCtx.Done():
		logger.Warn("graceful shutdown timed out, forcing stop")
		s.Stop()
	case <-stopped:
		logger.Info("graceful shutdown completed")
	}
}
