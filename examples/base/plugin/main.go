package main

import (
	"context"
	"log/slog"

	"github.com/trustdsh/grpc-plugin/examples/base/shared"
	"github.com/trustdsh/grpc-plugin/plugin"
)

type Plugin struct {
	shared.UnimplementedPluginServer
	logger *slog.Logger
}

func (p *Plugin) Start(options plugin.PluginOptions) {
	p.logger = options.Logger
	p.logger.Info("registering server")
	shared.RegisterPluginServer(options.Server, p)
}

func main() {
	plugin.StartPlugin(&Plugin{})
}

func (p *Plugin) DoSomething(ctx context.Context, in *shared.Empty) (*shared.Empty, error) {
	p.logger.Info("doing something")
	return &shared.Empty{}, nil
}

func (p *Plugin) GetSomething(ctx context.Context, in *shared.GetSomethingRequest) (*shared.GetSomethingResponse, error) {
	logger := p.logger.With("name", in.Name)
	logger.Info("getting something")

	if in.Name == "" {
		return &shared.GetSomethingResponse{
			Message: "nothing",
		}, nil
	}
	return &shared.GetSomethingResponse{
		Message: "something",
	}, nil
}
