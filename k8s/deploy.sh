#!/bin/bash
set -e

echo "🚀 Deploying Personal Wiki to Kubernetes (Rancher Desktop)"
echo "================================================"

# Build Docker image locally
echo "📦 Building Docker image..."
ls
docker build -t wiki:latest -f Dockerfile .

# Load image into Rancher Desktop's k3s
echo "📤 Loading image into Rancher Desktop..."

# Apply Kubernetes manifests
echo "☸️  Applying Kubernetes manifests..."
kubectl apply -k k8s/

# Wait for deployment to be ready
echo "⏳ Waiting for deployment to be ready..."
kubectl wait --for=condition=available --timeout=300s \
  deployment/wiki-deployment -n wiki

# Get pod status
echo ""
echo "✅ Deployment complete!"
echo ""
echo "Pod status:"
kubectl get pods -n wiki

echo ""
echo "📝 Access your wiki at: http://wiki.rancher.localhost"
echo ""
echo "Useful commands:"
echo "  View logs:        kubectl logs -f -n wiki deployment/wiki-deployment"
echo "  Check status:     kubectl get all -n wiki"
echo "  Delete wiki:      kubectl delete -k ."
echo "  Port forward:     kubectl port-forward -n wiki service/wiki-service 8080:80"
