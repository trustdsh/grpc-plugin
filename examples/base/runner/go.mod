module github.com/trustdsh/grpc-plugin/examples/base/runner

go 1.24.4

replace github.com/trustdsh/grpc-plugin/examples/base/shared => ../shared

replace github.com/trustdsh/grpc-plugin => ../../..

require (
	github.com/trustdsh/grpc-plugin v0.0.0-00010101000000-000000000000
	github.com/trustdsh/grpc-plugin/examples/base/shared v0.0.0-00010101000000-000000000000
)

require (
	github.com/pkg/errors v0.9.1 // indirect
	golang.org/x/net v0.38.0 // indirect
	golang.org/x/sys v0.31.0 // indirect
	golang.org/x/text v0.23.0 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20250324211829-b45e905df463 // indirect
	google.golang.org/grpc v1.73.0 // indirect
	google.golang.org/protobuf v1.36.6 // indirect
	gopkg.in/yaml.v2 v2.4.0 // indirect
)
