#!/bin/bash
# Script to generate gRPC code from proto files

# Make sure protoc is installed
# On Windows, you may need to install it separately or use WSL

protoc --go_out=. --go_opt=paths=source_relative \
       --go-grpc_out=. --go-grpc_opt=paths=source_relative \
       api/mangahub.proto

echo "gRPC code generated successfully!"

