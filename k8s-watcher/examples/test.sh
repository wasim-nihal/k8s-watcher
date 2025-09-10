#!/bin/bash

# Build the applications
echo "Building k8s-watcher..."
go build -o k8s-watcher ./cmd/k8s-watcher
if [ $? -ne 0 ]; then
    echo "Failed to build k8s-watcher"
    exit 1
fi

# Create output directory
mkdir -p /tmp/k8s-watcher-data

# Create the notification script
cat > /tmp/notify.sh << 'EOF'
#!/bin/sh
echo "Configuration changed at $(date)" >> /tmp/k8s-watcher.log
echo "Affected resource: $1" >> /tmp/k8s-watcher.log
echo "Namespace: $2" >> /tmp/k8s-watcher.log
EOF

# Make the script executable
chmod +x /tmp/notify.sh

# Build and start the webhook server
echo "Building webhook server..."
go build -o webhook-server ./test/tools/webhook-server
./webhook-server &
WEBHOOK_PID=$!

# Apply the Kubernetes resources
echo "Applying Kubernetes resources..."
kubectl apply -f examples/resources.yaml

# Start the watcher
echo "Starting k8s-watcher..."
./k8s-watcher -config examples/config.yaml

# Cleanup function
cleanup() {
    echo "Cleaning up..."
    kill $WEBHOOK_PID
    kubectl delete -f examples/resources.yaml
    rm -f /tmp/notify.sh webhook-server
    rm -rf /tmp/k8s-watcher-data
}

# Set up trap for cleanup
trap cleanup EXIT

# Wait for user input
echo "Press Ctrl+C to stop..."
wait
