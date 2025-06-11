# gRPC Plugin System For Golang

**NOTE: This is still in active development. Do NOT use it yet.**

[![Go Reference](https://pkg.go.dev/badge/github.com/trustdsh/grpc-plugin.svg)](https://pkg.go.dev/github.com/trustdsh/grpc-plugin)
[![Go Report Card](https://goreportcard.com/badge/github.com/trustdsh/grpc-plugin)](https://goreportcard.com/report/github.com/trustdsh/grpc-plugin)
[![License](https://img.shields.io/github/license/trustdsh/grpc-plugin.svg)](LICENSE)

A robust and secure system for building dynamically loadable gRPC-based plugins in Go. This library makes it easy to create and manage plugins that communicate over gRPC with mutual TLS authentication.

## Features

- 🔒 Secure by default with mTLS authentication
- 🔌 Dynamic plugin loading
- 📝 Structured logging with [slog](https://pkg.go.dev/log/slog)
- 🛠️ Easy-to-use API for both plugin and runner development
- 🔄 Graceful shutdown handling
- ⚡ Fast plugin startup with build-and-run mode
- 📦 YAML-based plugin configuration
- 🎯 Type-safe plugin interfaces using protocol buffers

## Installation

```bash
go get github.com/trustdsh/grpc-plugin
```

## Quick Start

### 1. Define Your Plugin Interface

Create a protocol buffer file that defines your plugin's interface:

```protobuf
syntax = "proto3";

service Plugin {
    rpc DoSomething(Empty) returns (Empty) {}
    rpc GetSomething(GetSomethingRequest) returns (GetSomethingResponse) {}
}

message GetSomethingRequest {
    string name = 1;
}

message GetSomethingResponse {
    string message = 1;
}

message Empty {}
```

Generate the Go code using protoc:

```bash
protoc --go_out=. --go-grpc_out=. plugin.proto
```

### 2. Implement a Plugin

```go
package main

import (
    "context"
    "log/slog"
    
    "github.com/trustdsh/grpc-plugin/plugin"
    "your/plugin/interface/pkg"
)

type Plugin struct {
    pkg.UnimplementedPluginServer
    logger *slog.Logger
}

func (p *Plugin) Start(options plugin.PluginOptions) {
    p.logger = options.Logger
    pkg.RegisterPluginServer(options.Server, p)
}

func (p *Plugin) DoSomething(ctx context.Context, in *pkg.Empty) (*pkg.Empty, error) {
    p.logger.Info("doing something")
    return &pkg.Empty{}, nil
}

func main() {
    plugin.StartPlugin(&Plugin{})
}
```

### 3. Create a Plugin Runner

First, create a plugin manifest file (`plugins.yml`):

```yaml
plugins:
  - path: ../plugin
    kind: build_and_run
```

Then implement the runner:

```go
package main

import (
    "context"
    "log/slog"
    "os"
    
    "github.com/trustdsh/grpc-plugin/pkgs/config"
    "github.com/trustdsh/grpc-plugin/runner"
    "your/plugin/interface/pkg"
)

func main() {
    cfg := config.Config[pkg.PluginClient]{
        LoggerOptions: &config.LoggerOptions{
            Type: "text",
            Level: slog.LevelInfo,
            Attributes: []slog.Attr{
                slog.String("component", "runner"),
            },
        },
        PluginGenerator: pkg.NewPluginClient,
        Manifest: &config.Manifest{
            Kind: "file",
            Path: "./plugins.yml",
        },
    }

    ctx := context.Background()
    plugins, err := runner.LoadAll(ctx, cfg)
    if err != nil {
        slog.Error("failed to load plugins", "error", err)
        os.Exit(1)
    }
    defer plugins.Close()

    // Use your plugins
    for _, plugin := range plugins.GetAllPlugins() {
        result, err := plugin.DoSomething(ctx, &pkg.Empty{})
        if err != nil {
            slog.Error("plugin call failed", "error", err)
            continue
        }
        slog.Info("plugin call succeeded", "result", result)
    }
}
```

## Configuration

### Plugin Manifest

The plugin manifest supports two formats:

1. File-based configuration (YAML):
```yaml
plugins:
  - path: ./plugin1    # Path to plugin directory
    kind: build_and_run # Plugin loading mode
    name: plugin1      # Optional, defaults to directory name

```

2. Inline configuration:
```go
cfg := config.Config[T]{
    Manifest: &config.Manifest{
        Kind: "inline",
        Config: &config.ManifestConfig{
            Plugins: []config.ManifestPlugin{
                {
                    Path: "./plugin1",
                    Kind: "build_and_run",
                },
            },
        },
    },
}
```

### Logger Configuration

The library uses Go's `slog` package for structured logging. You can configure:

- Log level (Debug, Info, Warn, Error)
- Output format (Text or JSON)
- Custom attributes
- Per-plugin logging configuration

### Security

By default, all plugin communication is secured using mutual TLS (mTLS). The library:

1. Generates a private CA for each runner instance
2. Issues unique certificates for each plugin
3. Validates certificates on both sides
4. Enforces TLS 1.2 minimum version

## Environment Variables

- `GRPC_PLUGINS_ALLOW_RELATIVE_PATHS_DOUBLE_DOT`: Set to "true" to allow plugins with `..` in their path (default: false)

## Advanced Usage

### Plugin Lifecycle Management

The library handles graceful shutdown and cleanup:

1. Plugins receive shutdown signals (SIGTERM/SIGINT)
2. Graceful shutdown period for cleanup
3. Automatic port and resource cleanup
4. Connection termination handling

## Contributing

Contributions are welcome! Please read our [Contributing Guide](CONTRIBUTING.md) for details on:

- Code of Conduct
- Development setup
- Submission guidelines
- Testing requirements

## License

This project is licensed under the Apache 2.0 License - see the [LICENSE](LICENSE) file for details.

## Acknowledgments

- The Go team for gRPC and protocol buffers support
- The community for feedback and contributions

## Support

- 📚 [Documentation](https://pkg.go.dev/github.com/trustdsh/grpc-plugin)
- 🐛 [Issue Tracker](https://github.com/trustdsh/grpc-plugin/issues)
- 💬 [Discussions](https://github.com/trustdsh/grpc-plugin/discussions)

---


 