# gRPC Plugins

**NOTE: This is still in active development. Do NOT use it yet.**

This is a system to make it easy to use gRPC based plugins that are dynamically loaded according to a set gRPC interface in golang.

This can be used in situations where you need to load multiple plugins which use the same interface, or where you may need to swap out plugins based on configuration and such.

## How To Start

### Define a plugin interface

Create a github repo (or directory, etc) which defines your plugin interface as a single protobuf service.
Generate your client and server code from this.
Commit it and release it.
This is what you will use in both your plugins and your runner to use your plugins.

### Write a plugin

1. Create a golang type which conforms to your plugin shape. It should additionally have a `Start()` function which registers the server:
```go
import "github.com/trustdsh/grpc-plugin/plugin"

// these are necessary
func (p *Plugin) Start(options plugin.PluginOptions) {
	p.logger = options.Logger
	shared.RegisterPluginServer(options.Server, p) // shared is the plugin interface defined above
}

func main() {
	plugin.StartPlugin(&Plugin{})
}

// ...other implementations of your protobuf APIs here
```

2. Release your plugin (or commit it)
NOTE: Released (binary-based) plugins are in development.

### Use the plugin in your runner

1. Define the configs of which plugins you want to load:
```yaml
plugins:
  - path: ../plugin
    kind: build_and_run
```

2. Load your plugins and use them:
```go
func main() {
	cfg := runner.Config[shared.PluginClient]{
		Logger:          slog.Default().With("run by", "runner"),
		PluginGenerator: shared.NewPluginClient,
		Manifest: &runner.Manifest{
			Kind: "file",
			Path: "./plugins.yml",
		},
	}

	plugins, err := runner.LoadAll(cfg)
	if err != nil {
		slog.Error("failed to load plugins", "error", err)
		os.Exit(1)
	}

	for _, plugin := range plugins {
		result, err := plugin.DoSomething(context.Background(), &shared.Empty{})
		if err != nil {
			slog.Error("failed to do something", "error", err)
			os.Exit(1)
		}
		slog.Info("result", "result", result.String())
	}
}
```

### Environment Variables

GRPC_PLUGINS_ALLOW_RELATIVE_PATHS_DOUBLE_DOT - set to true to enable plugins with a `..` in their path
 