package host

//go:generate protoc --go_out=. --go_opt=paths=source_relative --go-grpc_out=. --go-grpc_opt=paths=source_relative -I ../../../proto -I ../../../proto/cline cline/common.proto cline/task.proto cline/ui.proto cline/state.proto cline/account.proto cline/browser.proto cline/checkpoints.proto cline/commands.proto cline/file.proto cline/hooks.proto cline/mcp.proto cline/models.proto cline/oca_account.proto cline/slash.proto cline/web.proto cline/worktree.proto

// This file contains go:generate directives for generating Go code from protobuf definitions.
//
// Prerequisites:
//   - protoc (Protocol Buffers compiler)
//   - protoc-gen-go (Go protobuf plugin)
//   - protoc-gen-go-grpc (Go gRPC plugin)
//
// Installation:
//   go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
//   go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest
//
// To regenerate the Go code from proto files, run:
//   go generate ./...
//
// Generated files will be placed in the same directory as this file:
//   - *.pb.go files contain the protobuf message definitions
//   - *_grpc.pb.go files contain the gRPC service client and server interfaces
