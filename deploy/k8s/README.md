# Ollama Deployment Guide for Kubernetes

## 1. Starting Minikube

```bash
# Start minikube with allocated resources
minikube start --cpus=8 --memory=16g

# Mount local Ollama data directory to minikube
minikube mount ./ollama_data:/ollama_data
```

## 2. Deploying Ollama

```bash
# Apply the Kubernetes manifest
kubectl apply -f ollama-deployment.yaml

# Restart the deployment (if needed)
kubectl rollout restart deployment ollama
```

## 3. Monitoring and Diagnostics

```bash
# Track pod status
kubectl get pods -w

# View information about a specific pod
kubectl describe pod <pod-name>  # for example: ollama-6cb4d46fb5-psz5q

# Get a list of pods with a specific label
kubectl get pods -l app=ollama
```

## 4. Accessing the Ollama Service

In Minikube, NodePort services are not directly accessible from the host machine. There are two ways to access them:

### Option 1: Via Minikube Service

```bash
minikube service ollama-service --url
```

The command will return a URL, for example: `http://127.0.0.1:53029`  
The Ollama API will be accessible at this address.

### Option 2: Via Port-Forward

```bash
kubectl port-forward service/ollama-service 11434:11434
```

After executing this command, the API will be accessible at: `http://localhost:11434`

## 5. Testing the API

```bash
# Get a list of models
curl http://localhost:11434/api/tags

# Test embedding
curl http://localhost:11434/api/embeddings -d '{
  "model": "nomic-embed-text",
  "input": "hello world"
}'
```