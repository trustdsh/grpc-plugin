# Base Example

This example demonstrates the core functionality of the gRPC plugin system with a simple plugin implementation and runner.

## Structure

```
.
├── runner/         # Plugin runner implementation
│   ├── main.go    # Runner code that loads and uses plugins
│   ├── plugins.yml # Plugin configuration
│   └── go.mod     # Runner dependencies
├── shared/        # Shared protocol definitions
│   ├── plugin.proto    # Plugin interface definition
│   ├── plugin.pb.go   # Generated protobuf code
│   └── plugin_grpc.pb.go # Generated gRPC code
└── plugin/        # Example plugin implementation
    ├── main.go    # Plugin implementation
    └── go.mod     # Plugin dependencies
```

## Plugin Interface

The plugin implements a simple gRPC service with two methods:

```protobuf
service Plugin {
    rpc DoSomething(Empty) returns (Empty) {}
    rpc GetSomething(GetSomethingRequest) returns (GetSomethingResponse) {}
}
```

## Quick Start

1. Generate the protocol buffers code (if modified):
   ```bash
   cd shared
   protoc --go_out=. --go-grpc_out=. plugin.proto
   ```

2. Build and run the plugin runner:
   ```bash
   cd runner
   go run main.go
   ```

The runner will:
1. Load the plugin configuration from `plugins.yml`
2. Build and start the plugin
3. Make example RPC calls to demonstrate the functionality

## Configuration

The `plugins.yml` file configures which plugins to load:

```yaml
plugins:
  - path: ../plugin
    kind: build_and_run
```

## Features Demonstrated

- Basic plugin implementation
- Plugin loading and lifecycle management
- gRPC communication between runner and plugin
- Structured logging with `slog`
- Error handling and context propagation

