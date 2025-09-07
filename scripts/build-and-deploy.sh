#!/bin/bash

# GoSlack Build and Deploy Script
# This script builds the Docker image and deploys to Kubernetes

set -e  # Exit on any error

# Configuration
IMAGE_NAME="goslack-api"
IMAGE_TAG="latest"
NAMESPACE="goslack"

echo "🚀 GoSlack Build and Deploy Script"
echo "=================================="

# Check if kubectl is available
if ! command -v kubectl &> /dev/null; then
    echo "❌ kubectl is not installed or not in PATH"
    exit 1
fi

# Check if docker is available
if ! command -v docker &> /dev/null; then
    echo "❌ docker is not installed or not in PATH"
    exit 1
fi

# Function to build Docker image
build_image() {
    echo "📦 Building Docker image..."
    docker build -t ${IMAGE_NAME}:${IMAGE_TAG} .
    
    # If using minikube, load the image into minikube
    if command -v minikube &> /dev/null && minikube status &> /dev/null; then
        echo "📥 Loading image into minikube..."
        minikube image load ${IMAGE_NAME}:${IMAGE_TAG}
    fi
    
    echo "✅ Docker image built successfully"
}

# Function to deploy to Kubernetes
deploy_k8s() {
    echo "🚀 Deploying to Kubernetes..."
    
    # Apply all manifests
    kubectl apply -f k8s/namespace.yaml
    kubectl apply -f k8s/secret.yaml
    kubectl apply -f k8s/configmap.yaml
    kubectl apply -f k8s/postgres-pvc.yaml
    kubectl apply -f k8s/postgres-deployment.yaml
    kubectl apply -f k8s/postgres-service.yaml
    
    echo "⏳ Waiting for PostgreSQL to be ready..."
    kubectl wait --for=condition=ready pod -l app=postgres -n ${NAMESPACE} --timeout=300s
    
    # Run database migration
    kubectl apply -f k8s/migration-job.yaml
    echo "⏳ Waiting for database migration to complete..."
    kubectl wait --for=condition=complete job/goslack-migration -n ${NAMESPACE} --timeout=300s
    
    # Deploy the API
    kubectl apply -f k8s/goslack-api-deployment.yaml
    kubectl apply -f k8s/goslack-api-service.yaml
    kubectl apply -f k8s/ingress.yaml
    
    echo "⏳ Waiting for GoSlack API to be ready..."
    kubectl wait --for=condition=ready pod -l app=goslack-api -n ${NAMESPACE} --timeout=300s
    
    echo "✅ Deployment completed successfully"
}

# Function to show status
show_status() {
    echo "📊 Current Status:"
    echo "=================="
    kubectl get pods -n ${NAMESPACE}
    echo ""
    kubectl get services -n ${NAMESPACE}
    echo ""
    kubectl get ingress -n ${NAMESPACE}
    
    # Show service URL
    if command -v minikube &> /dev/null && minikube status &> /dev/null; then
        echo ""
        echo "🌐 Access your application at:"
        echo "   $(minikube service goslack-api-service -n ${NAMESPACE} --url)"
    fi
}

# Main execution
case "${1:-all}" in
    "build")
        build_image
        ;;
    "deploy")
        deploy_k8s
        show_status
        ;;
    "status")
        show_status
        ;;
    "all")
        build_image
        deploy_k8s
        show_status
        ;;
    *)
        echo "Usage: $0 [build|deploy|status|all]"
        echo "  build  - Build Docker image only"
        echo "  deploy - Deploy to Kubernetes only"
        echo "  status - Show current status"
        echo "  all    - Build and deploy (default)"
        exit 1
        ;;
esac

echo "🎉 Script completed successfully!"

