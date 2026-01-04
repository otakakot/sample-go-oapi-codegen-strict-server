package api

//go:generate go tool github.com/oapi-codegen/oapi-codegen/v2/cmd/oapi-codegen -generate server,strict-server -o server.gen.go -package api ../../openapi.yaml
//go:generate go tool github.com/oapi-codegen/oapi-codegen/v2/cmd/oapi-codegen -generate types -o types.gen.go -package api ../../openapi.yaml
//go:generate go tool github.com/oapi-codegen/oapi-codegen/v2/cmd/oapi-codegen -generate spec -o spec.gen.go -package api ../../openapi.yaml
