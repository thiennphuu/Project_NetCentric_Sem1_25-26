@echo off
REM Script to generate gRPC code from proto files on Windows
REM Requires protoc and protoc-gen-go to be installed

protoc --go_out=. --go_opt=paths=source_relative --go-grpc_out=. --go-grpc_opt=paths=source_relative api/mangahub.proto

if %ERRORLEVEL% EQU 0 (
    echo gRPC code generated successfully!
) else (
    echo Error generating gRPC code. Make sure protoc and protoc-gen-go are installed.
    echo Install with: go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
    echo                go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest
)

