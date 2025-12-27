.PHONY: help build build-mcp docker-build docker-up docker-down docker-logs run-research run-synthesis run-verification run-direct run-orchestration run-orchestration-eino run-all run-all-eino run-direct-verify run-mcp clean install test
.PHONY: k8s-build-images k8s-minikube-setup k8s-minikube-build k8s-minikube-deploy k8s-minikube-delete k8s-eks-deploy k8s-eks-delete helm-lint helm-template
.PHONY: helm-test helm-unittest helm-kubeconform helm-polaris helm-test-all

# Image registry (override for EKS: make k8s-eks-deploy REGISTRY=123456789.dkr.ecr.us-west-2.amazonaws.com)
REGISTRY ?=
IMAGE_TAG ?= latest

help:
	@echo "Statistics Agent - Make targets"
	@echo ""
	@echo "Docker Commands:"
	@echo "  make docker-build            Build Docker image (all-in-one)"
	@echo "  make docker-up               Start all agents with Docker Compose"
	@echo "  make docker-down             Stop all agents"
	@echo "  make docker-logs             View Docker logs"
	@echo ""
	@echo "Kubernetes Commands:"
	@echo "  make k8s-build-images        Build individual agent Docker images"
	@echo "  make k8s-minikube-setup      Setup Minikube with required addons"
	@echo "  make k8s-minikube-build      Build images in Minikube's Docker daemon"
	@echo "  make k8s-minikube-deploy     Deploy to Minikube with Helm"
	@echo "  make k8s-minikube-delete     Delete deployment from Minikube"
	@echo "  make k8s-eks-deploy          Deploy to AWS EKS with Helm"
	@echo "  make k8s-eks-delete          Delete deployment from EKS"
	@echo ""
	@echo "Helm Chart Commands:"
	@echo "  make helm-lint               Lint Helm chart"
	@echo "  make helm-template           Render Helm templates locally"
	@echo "  make helm-unittest           Run helm-unittest tests"
	@echo "  make helm-kubeconform        Validate against K8s schemas"
	@echo "  make helm-polaris            Check security best practices"
	@echo "  make helm-test-all           Run all chart tests"
	@echo ""
	@echo "Build Commands:"
	@echo "  make install                 Install dependencies"
	@echo "  make build                   Build all agents"
	@echo "  make build-mcp               Build MCP server"
	@echo ""
	@echo "Run Commands (Local):"
	@echo "  make run-research            Run research agent"
	@echo "  make run-synthesis           Run synthesis agent"
	@echo "  make run-verification        Run verification agent"
	@echo "  make run-direct              Run direct search agent (with OpenAPI docs)"
	@echo "  make run-direct-verify       Run direct + verification agents (hybrid mode)"
	@echo "  make run-orchestration       Run trpc-agent orchestration"
	@echo "  make run-orchestration-eino  Run Eino orchestration"
	@echo "  make run-all                 Run all agents with trpc-agent orchestrator"
	@echo "  make run-all-eino            Run all agents with Eino orchestrator"
	@echo "  make run-mcp                 Run MCP server (requires agents running)"
	@echo ""
	@echo "Other Commands:"
	@echo "  make test                    Run tests"
	@echo "  make clean                   Clean build artifacts"

install:
	go mod download
	go get github.com/trpc-group/trpc-agent-go
	go get github.com/trpc-group/trpc-a2a-go
	go get github.com/cloudwego/eino

build:
	@echo "Building agents..."
	go build -o bin/research agents/research/main.go
	go build -o bin/synthesis agents/synthesis/main.go
	go build -o bin/verification agents/verification/main.go
	go build -o bin/direct agents/direct/main.go
	go build -o bin/orchestration agents/orchestration/main.go
	go build -o bin/orchestration-eino agents/orchestration-eino/main.go
	go build -o bin/stats-agent main.go
	@echo "Build complete!"

build-mcp:
	@echo "Building MCP server..."
	go build -o bin/mcp-server mcp/server/main.go
	@echo "MCP server build complete!"

docker-build:
	@echo "Building Docker image..."
	docker build -t stats-agent-team:latest .
	@echo "Docker build complete!"

docker-up:
	@echo "Starting all agents with Docker Compose..."
	docker-compose up -d
	@echo "All agents started! Use 'make docker-logs' to view logs"

docker-down:
	@echo "Stopping all agents..."
	docker-compose down
	@echo "All agents stopped"

docker-logs:
	docker-compose logs -f

run-research:
	@echo "Starting Research Agent on :8001 (HTTP) and :9001 (A2A)..."
	go run agents/research/main.go

run-synthesis:
	@echo "Starting Synthesis Agent on :8004..."
	go run agents/synthesis/main.go

run-verification:
	@echo "Starting Verification Agent on :8002 (HTTP) and :9002 (A2A)..."
	go run agents/verification/main.go

run-direct:
	@echo "Starting Direct Agent on :8005 (HTTP)..."
	go run agents/direct/main.go

run-orchestration:
	@echo "Starting Orchestration Agent (trpc-agent) on :8000 (HTTP) and :9000 (A2A)..."
	go run agents/orchestration/main.go

run-orchestration-eino:
	@echo "Starting Orchestration Agent (Eino) on :8000 (HTTP)..."
	go run agents/orchestration-eino/main.go

run-all:
	@echo "Starting all agents with trpc-agent orchestrator..."
	@echo "Research Agent: http://localhost:8001 (A2A: 9001)"
	@echo "Synthesis Agent: http://localhost:8004"
	@echo "Verification Agent: http://localhost:8002 (A2A: 9002)"
	@echo "Orchestration Agent (trpc-agent): http://localhost:8000 (A2A: 9000)"
	@go run agents/research/main.go & \
	go run agents/synthesis/main.go & \
	go run agents/verification/main.go & \
	go run agents/orchestration/main.go & \
	wait

run-all-eino:
	@echo "Starting all agents with Eino orchestrator..."
	@echo "Research Agent: http://localhost:8001 (A2A: 9001)"
	@echo "Synthesis Agent: http://localhost:8004"
	@echo "Verification Agent: http://localhost:8002 (A2A: 9002)"
	@echo "Orchestration Agent (Eino): http://localhost:8000"
	@go run agents/research/main.go & \
	go run agents/synthesis/main.go & \
	go run agents/verification/main.go & \
	go run agents/orchestration-eino/main.go & \
	wait

run-direct-verify:
	@echo "Starting Direct + Verification agents (hybrid mode)..."
	@echo "Direct Agent: http://localhost:8005 (OpenAPI docs at /docs)"
	@echo "Verification Agent: http://localhost:8002"
	@echo ""
	@echo "Usage: ./bin/stats-agent search \"topic\" --direct --direct-verify"
	@go run agents/direct/main.go & \
	go run agents/verification/main.go & \
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

# ============================================
# Kubernetes / Helm Commands
# ============================================

# Build individual agent Docker images
k8s-build-images:
	@echo "Building individual agent Docker images..."
	docker build --build-arg AGENT=research -t stats-agent-research:$(IMAGE_TAG) -f Dockerfile.agent .
	docker build --build-arg AGENT=synthesis -t stats-agent-synthesis:$(IMAGE_TAG) -f Dockerfile.agent .
	docker build --build-arg AGENT=verification -t stats-agent-verification:$(IMAGE_TAG) -f Dockerfile.agent .
	docker build --build-arg AGENT=orchestration-eino -t stats-agent-orchestration-eino:$(IMAGE_TAG) -f Dockerfile.agent .
	docker build --build-arg AGENT=direct -t stats-agent-direct:$(IMAGE_TAG) -f Dockerfile.agent .
	@echo "All agent images built successfully!"

# Setup Minikube with required addons
k8s-minikube-setup:
	@echo "Setting up Minikube..."
	minikube start --cpus=4 --memory=8192
	minikube addons enable ingress
	minikube addons enable metrics-server
	@echo "Minikube is ready!"

# Build images directly in Minikube's Docker daemon
k8s-minikube-build:
	@echo "Building images in Minikube's Docker daemon..."
	@eval $$(minikube docker-env) && \
		docker build --build-arg AGENT=research -t stats-agent-research:$(IMAGE_TAG) -f Dockerfile.agent . && \
		docker build --build-arg AGENT=synthesis -t stats-agent-synthesis:$(IMAGE_TAG) -f Dockerfile.agent . && \
		docker build --build-arg AGENT=verification -t stats-agent-verification:$(IMAGE_TAG) -f Dockerfile.agent . && \
		docker build --build-arg AGENT=orchestration-eino -t stats-agent-orchestration-eino:$(IMAGE_TAG) -f Dockerfile.agent . && \
		docker build --build-arg AGENT=direct -t stats-agent-direct:$(IMAGE_TAG) -f Dockerfile.agent .
	@echo "All images built in Minikube!"

# Deploy to Minikube with Helm
k8s-minikube-deploy:
	@echo "Deploying to Minikube..."
	helm upgrade --install stats-agent ./helm/stats-agent-team \
		-f ./helm/stats-agent-team/values-minikube.yaml \
		--namespace stats-agent \
		--create-namespace
	@echo ""
	@echo "Deployment complete! Access the orchestration agent:"
	@echo "  kubectl port-forward -n stats-agent svc/stats-agent-stats-agent-team-orchestration 8000:8000"
	@echo ""
	@echo "Or get all services:"
	@echo "  kubectl get svc -n stats-agent"

# Delete deployment from Minikube
k8s-minikube-delete:
	@echo "Deleting deployment from Minikube..."
	helm uninstall stats-agent --namespace stats-agent || true
	kubectl delete namespace stats-agent || true
	@echo "Deployment deleted!"

# Push images to ECR (requires AWS CLI configured)
k8s-eks-push:
	@if [ -z "$(REGISTRY)" ]; then echo "Error: REGISTRY not set. Use: make k8s-eks-push REGISTRY=your-ecr-registry"; exit 1; fi
	@echo "Pushing images to ECR..."
	@for agent in research synthesis verification orchestration-eino direct; do \
		docker tag stats-agent-$$agent:$(IMAGE_TAG) $(REGISTRY)/stats-agent-$$agent:$(IMAGE_TAG) && \
		docker push $(REGISTRY)/stats-agent-$$agent:$(IMAGE_TAG); \
	done
	@echo "All images pushed to ECR!"

# Deploy to AWS EKS with Helm
k8s-eks-deploy:
	@if [ -z "$(REGISTRY)" ]; then echo "Error: REGISTRY not set. Use: make k8s-eks-deploy REGISTRY=your-ecr-registry"; exit 1; fi
	@echo "Deploying to AWS EKS..."
	helm upgrade --install stats-agent ./helm/stats-agent-team \
		-f ./helm/stats-agent-team/values-eks.yaml \
		--namespace stats-agent \
		--create-namespace \
		--set global.image.registry=$(REGISTRY) \
		--set global.image.tag=$(IMAGE_TAG)
	@echo ""
	@echo "Deployment complete! Get the ALB address:"
	@echo "  kubectl get ingress -n stats-agent"

# Delete deployment from EKS
k8s-eks-delete:
	@echo "Deleting deployment from EKS..."
	helm uninstall stats-agent --namespace stats-agent || true
	kubectl delete namespace stats-agent || true
	@echo "Deployment deleted!"

# Lint Helm chart
helm-lint:
	@echo "Linting Helm chart..."
	helm lint ./helm/stats-agent-team
	helm lint ./helm/stats-agent-team -f ./helm/stats-agent-team/values-minikube.yaml
	helm lint ./helm/stats-agent-team -f ./helm/stats-agent-team/values-eks.yaml
	@echo "Helm chart is valid!"

# Render Helm templates locally for debugging
helm-template:
	@echo "Rendering Helm templates..."
	helm template stats-agent ./helm/stats-agent-team -f ./helm/stats-agent-team/values-minikube.yaml

# ============================================
# Helm Chart Testing Commands
# ============================================

# Run helm-unittest (requires: helm plugin install https://github.com/helm-unittest/helm-unittest.git)
helm-unittest:
	@echo "Running helm-unittest..."
	@if helm plugin list | grep -q unittest; then \
		helm unittest ./helm/stats-agent-team; \
	else \
		echo "Installing helm-unittest plugin..."; \
		helm plugin install https://github.com/helm-unittest/helm-unittest.git; \
		helm unittest ./helm/stats-agent-team; \
	fi

# Validate rendered templates against Kubernetes schemas (requires: kubeconform)
helm-kubeconform:
	@echo "Validating Kubernetes schemas with kubeconform..."
	@if command -v kubeconform >/dev/null 2>&1; then \
		helm template stats-agent ./helm/stats-agent-team | \
			kubeconform -strict -summary -kubernetes-version 1.29.0; \
	else \
		echo "kubeconform not found. Install: brew install kubeconform (or see https://github.com/yannh/kubeconform)"; \
		exit 1; \
	fi

# Check security best practices with Polaris (requires: polaris)
helm-polaris:
	@echo "Running Polaris security audit..."
	@if command -v polaris >/dev/null 2>&1; then \
		helm template stats-agent ./helm/stats-agent-team | \
			polaris audit --audit-path - --format pretty; \
	else \
		echo "Polaris not found. Install: brew install FairwindsOps/tap/polaris (or see https://github.com/FairwindsOps/polaris)"; \
		exit 1; \
	fi

# Run all chart tests
helm-test-all: helm-lint helm-unittest helm-kubeconform
	@echo ""
	@echo "All Helm chart tests passed!"

# Quick test (lint + unittest only, no external tools required)
helm-test: helm-lint helm-unittest
	@echo ""
	@echo "Helm chart tests passed!"
