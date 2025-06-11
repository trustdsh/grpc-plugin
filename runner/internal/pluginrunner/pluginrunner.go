package pluginrunner

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"syscall"
	"time"

	"github.com/pkg/errors"
	"github.com/trustdsh/grpc-plugin/internal/transport"
	"github.com/trustdsh/grpc-plugin/pkgs/config"
	"github.com/trustdsh/grpc-plugin/runner/internal/pluginrunner/portmanager"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

const (
	startupTimeout = 10 * time.Second
)

type LoadedPlugin[T any] struct {
	Plugin T
	Server *PluginServerConf
}

type PluginServerConf struct {
	Port    int
	Process *os.Process
}

func buildAndRunPlugin[T any](ctx context.Context, pluginConfig config.ManifestPlugin, cfg *config.Config[T], options *PluginServerOptions) (*PluginServerConf, error) {
	logger := slog.With("component", "plugin_runner", "plugin", pluginConfig.GetName())
	logger.Debug("starting plugin build and run")

	wd, err := os.Getwd()
	if err != nil {
		logger.Error("failed to get working directory", "error", err)
		return nil, errors.Wrap(err, "failed to get working directory")
	}

	cliOptions, err := options.ToCliOptions()
	if err != nil {
		logger.Error("failed to generate CLI options", "error", err)
		return nil, errors.Wrap(err, "failed to generate CLI options")
	}

	pluginPath := filepath.Join(wd, pluginConfig.Path)
	logger.Debug("building and running plugin", "path", pluginPath, "cli_options", cliOptions)

	cmd := exec.CommandContext(ctx, "/usr/bin/env", append([]string{"go", "run", "./..."}, cliOptions...)...)
	cmd.Dir = pluginPath
	cmd.Env = os.Environ()
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.SysProcAttr = &syscall.SysProcAttr{
		Setpgid: true,
	}
	err = cmd.Start()
	if err != nil {
		logger.Error("failed to start plugin process", "error", err)
		return nil, errors.Wrapf(err, "failed to start plugin process at %s", pluginPath)
	}

	logger.Info("plugin process started", "pid", cmd.Process.Pid, "port", options.Port)

	// Wait for the plugin to start
	startCtx, cancel := context.WithTimeout(ctx, startupTimeout)
	defer cancel()

	// Try to connect to the plugin
	for {
		select {
		case <-startCtx.Done():
			if err := syscall.Kill(-cmd.Process.Pid, syscall.SIGTERM); err != nil {
				logger.Error("failed to kill plugin process after timeout", "error", err)
			}
			return nil, errors.Wrapf(startCtx.Err(), "plugin %s failed to start within %v", pluginConfig.GetName(), startupTimeout)
		case <-ctx.Done():
			if err := syscall.Kill(-cmd.Process.Pid, syscall.SIGTERM); err != nil {
				logger.Error("failed to kill plugin process after context cancellation", "error", err)
			}
			return nil, errors.Wrap(ctx.Err(), "context cancelled while waiting for plugin to start")
		default:
			conn, err := net.DialTimeout("tcp", net.JoinHostPort("localhost", strconv.Itoa(options.Port)), time.Second)
			if err == nil {
				conn.Close()
				return &PluginServerConf{
					Port:    options.Port,
					Process: cmd.Process,
				}, nil
			}
			time.Sleep(100 * time.Millisecond)
		}
	}
}

type PluginServerOptions struct {
	Port          int
	KeyAndCert    *transport.KeyAndCert
	LoggerOptions *config.LoggerOptions
	PluginName    string
}

func (options *PluginServerOptions) ToCliOptions() ([]string, error) {
	opts := []string{}
	if options.Port != 0 {
		opts = append(opts, "-port", fmt.Sprintf("%d", options.Port))
	}
	if options.KeyAndCert != nil {
		keyAndCertBytes, err := options.KeyAndCert.Serialize()
		if err != nil {
			return nil, errors.Wrap(err, "failed to serialize key and cert")
		}
		opts = append(opts, "-tls_key_and_cert", string(keyAndCertBytes))
	}
	if options.PluginName != "" {
		opts = append(opts, "-plugin_name", options.PluginName)
	}
	if options.LoggerOptions != nil {
		loggerOptsJSON, err := options.LoggerOptions.MarshalJSON()
		if err != nil {
			return nil, errors.Wrap(err, "failed to marshal logger options")
		}
		opts = append(opts, "-logger_options", string(loggerOptsJSON))
	}
	return opts, nil
}

func startPluginServer[T any](ctx context.Context, pluginConfig config.ManifestPlugin, cfg *config.Config[T], transportGenerator *transport.TransportGenerator, portMgr *portmanager.PortManager) (*PluginServerConf, error) {
	logger := slog.With("component", "plugin_runner", "plugin", pluginConfig.GetName())
	logger.Debug("starting plugin server")

	serverKeyAndCert, err := transportGenerator.GenerateKeyAndCert(pluginConfig.GetName(), "server")
	if err != nil {
		logger.Error("failed to generate server key and cert", "error", err)
		return nil, errors.Wrapf(err, "failed to generate server key and cert for plugin %s", pluginConfig.GetName())
	}

	port, err := portMgr.GetPort()
	if err != nil {
		logger.Error("failed to get port", "error", err)
		return nil, errors.Wrap(err, "failed to get available port")
	}

	options := &PluginServerOptions{
		KeyAndCert:    serverKeyAndCert,
		Port:          port,
		LoggerOptions: cfg.LoggerOptions,
		PluginName:    pluginConfig.GetName(),
	}

	var pluginServer *PluginServerConf
	var startErr error

	if pluginConfig.Kind == "build_and_run" {
		pluginServer, startErr = buildAndRunPlugin(ctx, pluginConfig, cfg, options)
	} else {
		startErr = errors.Errorf("plugin kind %q is not supported", pluginConfig.Kind)
	}

	if startErr != nil {
		if err := portMgr.ReleasePort(port); err != nil {
			logger.Error("failed to release port after error", "error", err)
		}
		return nil, startErr
	}

	return pluginServer, nil
}

func createPluginClient[T any](pluginServer *PluginServerConf, pluginConfig config.ManifestPlugin, cfg *config.Config[T], transportGenerator *transport.TransportGenerator) (T, error) {
	logger := slog.With("component", "plugin_runner", "plugin", pluginConfig.GetName())
	logger.Debug("creating plugin client")

	var nilt T
	keyAndCert, err := transportGenerator.GenerateKeyAndCert(pluginConfig.GetName()+"_client", "client")
	if err != nil {
		logger.Error("failed to generate client key and cert", "error", err)
		return nilt, errors.Wrapf(err, "failed to generate client key and cert for plugin %s", pluginConfig.GetName())
	}

	clientTLSConfig, err := keyAndCert.GetTLSConfig()
	if err != nil {
		logger.Error("failed to get client TLS config", "error", err)
		return nilt, errors.Wrapf(err, "failed to get client TLS config for plugin %s", pluginConfig.GetName())
	}

	addr := net.JoinHostPort("localhost", strconv.Itoa(pluginServer.Port))
	logger.Debug("connecting to plugin server", "address", addr)

	conn, err := grpc.NewClient(addr, grpc.WithTransportCredentials(credentials.NewTLS(clientTLSConfig)))
	if err != nil {
		logger.Error("failed to create gRPC client", "error", err)
		return nilt, errors.Wrapf(err, "failed to create gRPC client for plugin %s at %s", pluginConfig.GetName(), addr)
	}

	logger.Info("plugin client created successfully")
	return cfg.PluginGenerator(conn), nil
}

func (l *LoadedPlugin[T]) Close() error {
	if l.Server.Process != nil {
		err := syscall.Kill(-l.Server.Process.Pid, syscall.SIGTERM)
		if err != nil {
			slog.Error("failed to terminate plugin process", "error", err, "pid", l.Server.Process.Pid)
			return errors.Wrapf(err, "failed to terminate plugin process with PID %d", l.Server.Process.Pid)
		}
		slog.Debug("plugin process terminated", "pid", l.Server.Process.Pid)
	}
	return nil
}

func LoadPlugin[T any](ctx context.Context, pluginConfig config.ManifestPlugin, transportGenerator *transport.TransportGenerator, cfg *config.Config[T], portMgr *portmanager.PortManager) (*LoadedPlugin[T], error) {
	logger := slog.With("component", "plugin_runner", "plugin", pluginConfig.GetName())
	logger.Info("loading plugin")

	pluginServer, err := startPluginServer(ctx, pluginConfig, cfg, transportGenerator, portMgr)
	if err != nil {
		logger.Error("failed to start plugin server", "error", err)
		return nil, errors.Wrapf(err, "failed to start server for plugin %s", pluginConfig.GetName())
	}

	pluginClient, err := createPluginClient(pluginServer, pluginConfig, cfg, transportGenerator)
	if err != nil {
		logger.Error("failed to create plugin client", "error", err)
		if closeErr := syscall.Kill(-pluginServer.Process.Pid, syscall.SIGTERM); closeErr != nil {
			logger.Error("failed to kill plugin process after client creation error", "error", closeErr)
		}
		return nil, errors.Wrapf(err, "failed to create client for plugin %s", pluginConfig.GetName())
	}

	logger.Info("plugin loaded successfully")
	return &LoadedPlugin[T]{
		Plugin: pluginClient,
		Server: pluginServer,
	}, nil
}
