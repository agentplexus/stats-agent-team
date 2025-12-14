.PHONY: help build build-mcp run-research run-verification run-orchestration run-orchestration-eino run-all run-all-eino run-mcp clean install

help:
	@echo "Statistics Agent - Make targets"
	@echo ""
	@echo "  make install                 Install dependencies"
	@echo "  make build                   Build all agents"
	@echo "  make build-mcp               Build MCP server"
	@echo "  make run-research            Run research agent"
	@echo "  make run-verification        Run verification agent"
	@echo "  make run-orchestration       Run trpc-agent orchestration"
	@echo "  make run-orchestration-eino  Run Eino orchestration"
	@echo "  make run-all                 Run all agents with trpc-agent orchestrator"
	@echo "  make run-all-eino            Run all agents with Eino orchestrator"
	@echo "  make run-mcp                 Run MCP server (requires agents running)"
	@echo "  make clean                   Clean build artifacts"
	@echo "  make test                    Run tests"

install:
	go mod download
	go get github.com/trpc-group/trpc-agent-go
	go get github.com/trpc-group/trpc-a2a-go
	go get github.com/cloudwego/eino

build:
	@echo "Building agents..."
	go build -o bin/research agents/research/main.go
	go build -o bin/verification agents/verification/main.go
	go build -o bin/orchestration agents/orchestration/main.go
	go build -o bin/orchestration-eino agents/orchestration-eino/main.go
	go build -o bin/stats-agent main.go
	@echo "Build complete!"

build-mcp:
	@echo "Building MCP server..."
	go build -o bin/mcp-server mcp/server/main.go
	@echo "MCP server build complete!"

run-research:
	@echo "Starting Research Agent on :8001 (HTTP) and :9001 (A2A)..."
	go run agents/research/main.go

run-verification:
	@echo "Starting Verification Agent on :8002 (HTTP) and :9002 (A2A)..."
	go run agents/verification/main.go

run-orchestration:
	@echo "Starting Orchestration Agent (trpc-agent) on :8000 (HTTP) and :9000 (A2A)..."
	go run agents/orchestration/main.go

run-orchestration-eino:
	@echo "Starting Orchestration Agent (Eino) on :8003 (HTTP) and :9003 (A2A)..."
	go run agents/orchestration-eino/main.go

run-all:
	@echo "Starting all agents with trpc-agent orchestrator..."
	@echo "Research Agent: http://localhost:8001 (A2A: 9001)"
	@echo "Verification Agent: http://localhost:8002 (A2A: 9002)"
	@echo "Orchestration Agent (trpc-agent): http://localhost:8000 (A2A: 9000)"
	@go run agents/research/main.go & \
	go run agents/verification/main.go & \
	go run agents/orchestration/main.go & \
	wait

run-all-eino:
	@echo "Starting all agents with Eino orchestrator..."
	@echo "Research Agent: http://localhost:8001 (A2A: 9001)"
	@echo "Verification Agent: http://localhost:8002 (A2A: 9002)"
	@echo "Orchestration Agent (Eino): http://localhost:8003 (A2A: 9003)"
	@go run agents/research/main.go & \
	go run agents/verification/main.go & \
	go run agents/orchestration-eino/main.go & \
	wait

run-mcp:
	@echo "Starting MCP server (stdio)..."
	@echo "Note: Ensure research and verification agents are running first!"
	@echo "  Terminal 1: make run-research"
	@echo "  Terminal 2: make run-verification"
	@go run mcp/server/main.go

clean:
	rm -rf bin/
	go clean

test:
	go test ./...
