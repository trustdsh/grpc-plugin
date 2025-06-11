package runner

import (
	"context"

	"github.com/trustdsh/grpc-plugin/pkgs/config"
	"github.com/trustdsh/grpc-plugin/runner/internal/pluginsloader"
)

func LoadAll[T any](ctx context.Context, cfg config.Config[T]) (*pluginsloader.LoadedPlugins[T], error) {
	return pluginsloader.LoadAll(ctx, cfg)
}
