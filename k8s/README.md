# Kubernetes Deployment for Personal Wiki

This directory contains Kubernetes manifests for deploying the Personal Wiki on Rancher Desktop or any Kubernetes cluster with Traefik.

## Prerequisites

- Rancher Desktop (or any k8s cluster)
- kubectl configured
- Docker or nerdctl for building images
- Traefik installed (comes with Rancher Desktop)

## Quick Deploy

```bash
# Run the deploy script
./deploy.sh
```

This will:
1. Build the Docker image
2. Create the wiki namespace
3. Deploy all resources
4. Wait for pods to be ready

## Manual Deployment

```bash
# Build the Docker image
docker build -t wiki:latest ..

# Apply all manifests using Kustomize
kubectl apply -k .

# Check deployment status
kubectl get all -n wiki

# Wait for deployment
kubectl wait --for=condition=available deployment/wiki-deployment -n wiki
```

## Accessing the Wiki

Once deployed, access your wiki at: **http://wiki.rancher.localhost**

If wiki.rancher.localhost doesn't resolve, you can:

1. Use port-forward:
```bash
kubectl port-forward -n wiki service/wiki-service 8080:80
# Access at http://localhost:8080
```

2. Add to /etc/hosts:
```bash
echo "127.0.0.1 wiki.rancher.localhost" | sudo tee -a /etc/hosts
```

## Files

- `namespace.yaml` - Creates the wiki namespace
- `persistentvolumeclaim.yaml` - Storage for SQLite database
- `configmap.yaml` - Configuration (PORT setting)
- `deployment.yaml` - Wiki pod deployment
- `service.yaml` - ClusterIP service
- `ingressroute.yaml` - Traefik IngressRoute (CRD)
- `ingress.yaml` - Standard Ingress (alternative)
- `kustomization.yaml` - Kustomize configuration
- `deploy.sh` - Automated deployment script

## Management Commands

### View logs
```bash
kubectl logs -f -n wiki deployment/wiki-deployment
```

### Check pod status
```bash
kubectl get pods -n wiki
kubectl describe pod -n wiki <pod-name>
```

### Access pod shell
```bash
kubectl exec -it -n wiki deployment/wiki-deployment -- /bin/sh
```

### Update deployment
```bash
# After rebuilding image
kubectl rollout restart deployment/wiki-deployment -n wiki
```

### Backup database
```bash
# Copy database from pod to local
POD=$(kubectl get pod -n wiki -l app=personal-wiki -o jsonpath='{.items[0].metadata.name}')
kubectl cp wiki/$POD:/app/data/wiki.db ./wiki-backup.db
```

### Restore database
```bash
# Copy database from local to pod
POD=$(kubectl get pod -n wiki -l app=personal-wiki -o jsonpath='{.items[0].metadata.name}')
kubectl cp ./wiki-backup.db wiki/$POD:/app/data/wiki.db
```

### Delete everything
```bash
kubectl delete -k .
# or
kubectl delete namespace wiki
```

## Customization

### Change hostname
Edit `ingressroute.yaml` or `ingress.yaml` and change the Host match:
```yaml
- match: Host(`your-domain.local`)
```

### Increase storage
Edit `persistentvolumeclaim.yaml`:
```yaml
resources:
  requests:
    storage: 5Gi  # Increase as needed
```

### Adjust resources
Edit `deployment.yaml`:
```yaml
resources:
  requests:
    memory: "128Mi"
    cpu: "100m"
  limits:
    memory: "512Mi"
    cpu: "1000m"
```

### Use external database
Modify the deployment to use an external PostgreSQL/MySQL instead of SQLite.

## Troubleshooting

### Pod not starting
```bash
kubectl describe pod -n wiki <pod-name>
kubectl logs -n wiki <pod-name>
```

### Can't access wiki.rancher.localhost
1. Check if Traefik is running:
```bash
kubectl get pods -n kube-system | grep traefik
```

2. Check IngressRoute:
```bash
kubectl describe ingressroute -n wiki wiki-ingressroute
```

3. Use port-forward as workaround:
```bash
kubectl port-forward -n wiki service/wiki-service 8080:80
```

### Storage issues
Check PVC status:
```bash
kubectl get pvc -n wiki
kubectl describe pvc -n wiki wiki-data-pvc
```

### Image not found
Make sure Docker/nerdctl is using the same runtime as your k8s cluster:
```bash
# For Docker Desktop / Rancher Desktop
docker build -t wiki:latest ..

# For containerd/nerdctl
nerdctl -n k8s.io build -t wiki:latest ..
```