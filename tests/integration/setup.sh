#!/bin/bash
# Quick start script for running k8s-watcher integration tests locally

set -e

echo "🚀 k8s-watcher Integration Test Setup"
echo "======================================"
echo ""

# Check prerequisites
echo "📋 Checking prerequisites..."

# Check Docker
if ! command -v docker &> /dev/null; then
    echo "❌ Docker not found. Please install Docker first."
    exit 1
fi
echo "✓ Docker found"

# Check KinD
if ! command -v kind &> /dev/null; then
    echo "⚠️  KinD not found. Installing..."
    go install sigs.k8s.io/kind@latest
    if ! command -v kind &> /dev/null; then
        echo "❌ KinD installation failed. Please install manually: https://kind.sigs.k8s.io/docs/user/quick-start/"
        exit 1
    fi
fi
echo "✓ KinD found"

# Check kubectl
if ! command -v kubectl &> /dev/null; then
    echo "❌ kubectl not found. Please install kubectl first."
    exit 1
fi
echo "✓ kubectl found"

# Check Python
if ! command -v python3 &> /dev/null; then
    echo "❌ Python 3 not found. Please install Python 3.11 or later."
    exit 1
fi
PYTHON_VERSION=$(python3 --version | cut -d' ' -f2 | cut -d'.' -f1,2)
echo "✓ Python $PYTHON_VERSION found"

# Check pip
if command -v pip &> /dev/null; then
    PIP_CMD="pip"
elif command -v pip3 &> /dev/null; then
    PIP_CMD="pip3"
else
    echo "❌ pip not found. Please install pip first."
    exit 1
fi
echo "✓ pip found"

echo ""
echo "📦 Installing Python dependencies..."

# Detect if we're in tests/integration or project root
if [ -f "requirements.txt" ]; then
    # Already in tests/integration
    $PIP_CMD install -q -r requirements.txt
    PROJECT_ROOT="../.."
elif [ -d "tests/integration" ]; then
    # In project root
    cd tests/integration
    $PIP_CMD install -q -r requirements.txt
    PROJECT_ROOT="../.."
else
    echo "❌ Cannot find tests/integration directory. Please run from project root or tests/integration."
    exit 1
fi
echo "✓ Dependencies installed"

echo ""
echo "🏗️  Building k8s-watcher..."
cd $PROJECT_ROOT
make build
echo "✓ Build complete"

echo ""
echo "🐳 Building Docker image..."
docker build -q -t k8s-watcher:test .
echo "✓ Docker image built"

echo ""
echo "✅ Setup complete!"
echo ""
echo "To run the integration tests:"
echo "  cd tests/integration"
echo "  pytest -v"
echo ""
echo "To run specific test suites:"
echo "  pytest test_configmap_watch.py -v"
echo "  pytest test_secret_watch.py -v"
echo "  pytest test_label_matching.py -v"
echo ""
echo "To run tests with markers:"
echo "  pytest -m configmap -v"
echo "  pytest -m secret -v"
echo ""
