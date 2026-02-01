# Kubernetes Deployment Guide

This guide covers deploying EquiShare to Kubernetes using Kustomize overlays.

## Directory Structure

```
infrastructure/k8s/
├── base/                    # Common resources
│   ├── kustomization.yaml
│   ├── namespace.yaml
│   ├── service-account.yaml
│   ├── network-policy.yaml
│   └── <service>/
│       ├── deployment.yaml
│       ├── service.yaml
│       └── hpa.yaml
├── staging/                 # Staging overlay
│   └── kustomization.yaml
└── production/              # Production overlay
    ├── kustomization.yaml
    ├── pdb.yaml
    ├── ingress.yaml
    └── service-account.yaml
```

## Prerequisites

1. **Kubernetes cluster** (GKE, EKS, AKS, or similar)
2. **kubectl** configured to access the cluster
3. **Kustomize** (built into kubectl v1.14+)
4. **Secrets** configured (see Secrets section)

## Quick Start

### Deploy to Staging

```bash
# Preview what will be deployed
kubectl kustomize infrastructure/k8s/staging/

# Apply to cluster
kubectl apply -k infrastructure/k8s/staging/
```

### Deploy to Production

```bash
# Preview
kubectl kustomize infrastructure/k8s/production/

# Apply
kubectl apply -k infrastructure/k8s/production/
```

## Services Overview

| Service | Port | Replicas (Prod) | Purpose |
|---------|------|-----------------|---------|
| api-gateway | 8000 | 2-5 | Entry point, routing |
| auth-service | 8001 | 2-3 | Authentication |
| user-service | 8002 | 2-3 | User profiles |
| trading-service | 8003 | 2-5 | Order execution |
| payment-service | 8004 | 2-3 | M-Pesa payments |
| ussd-service | 8005 | 2-3 | USSD interface |
| notification-service | 8006 | 2-4 | SMS/Push/Email |
| market-data-service | 8007 | 2-4 | Real-time prices |
| portfolio-service | 8008 | 2-3 | Portfolio tracking |

## Secrets Configuration

Before deploying, create the required secrets:

```bash
# Database credentials
kubectl create secret generic equishare-db-credentials \
  --namespace=equishare-production \
  --from-literal=username=equishare \
  --from-literal=password='YOUR_DB_PASSWORD'

# JWT secret
kubectl create secret generic equishare-jwt-secret \
  --namespace=equishare-production \
  --from-literal=secret='YOUR_JWT_SECRET_MIN_32_CHARS'

# M-Pesa credentials
kubectl create secret generic equishare-mpesa-credentials \
  --namespace=equishare-production \
  --from-literal=consumer_key='YOUR_CONSUMER_KEY' \
  --from-literal=consumer_secret='YOUR_CONSUMER_SECRET' \
  --from-literal=pass_key='YOUR_PASS_KEY'

# Alpaca credentials
kubectl create secret generic equishare-alpaca-credentials \
  --namespace=equishare-production \
  --from-literal=api_key='YOUR_API_KEY' \
  --from-literal=api_secret='YOUR_API_SECRET'
```

### Using External Secrets Operator (Recommended)

For production, use External Secrets Operator with AWS SSM or GCP Secret Manager:

```yaml
apiVersion: external-secrets.io/v1beta1
kind: ExternalSecret
metadata:
  name: equishare-db-credentials
spec:
  refreshInterval: 1h
  secretStoreRef:
    kind: ClusterSecretStore
    name: gcp-secrets
  target:
    name: equishare-db-credentials
  data:
    - secretKey: username
      remoteRef:
        key: equishare-db-username
    - secretKey: password
      remoteRef:
        key: equishare-db-password
```

## Resource Configuration

### Staging

Staging uses minimal resources for cost efficiency:
- 1 replica per service
- 50m CPU request, 200m limit
- 64Mi memory request, 256Mi limit
- HPA disabled (max 2 replicas)

### Production

Production is configured for high availability:
- 2+ replicas per service
- Full resource allocation
- HPA enabled with auto-scaling
- PodDisruptionBudgets for zero-downtime updates
- Topology spread across zones

## Health Checks

All services implement three probe types:

1. **Liveness Probe** - Restarts unhealthy pods
   ```yaml
   livenessProbe:
     httpGet:
       path: /health
       port: http
     initialDelaySeconds: 15
     periodSeconds: 20
   ```

2. **Readiness Probe** - Controls traffic routing
   ```yaml
   readinessProbe:
     httpGet:
       path: /health
       port: http
     initialDelaySeconds: 5
     periodSeconds: 10
   ```

3. **Startup Probe** - Handles slow-starting containers
   ```yaml
   startupProbe:
     httpGet:
       path: /health
       port: http
     initialDelaySeconds: 10
     failureThreshold: 30
   ```

## Ingress Configuration

Production uses NGINX Ingress with TLS:

| Domain | Service | Notes |
|--------|---------|-------|
| api.equishare.com | api-gateway | Main API |
| ussd.equishare.com | ussd-service | Africa's Talking callbacks |
| mpesa.equishare.com | payment-service | M-Pesa callbacks |

### TLS Certificates

Using cert-manager with Let's Encrypt:

```bash
# Install cert-manager
kubectl apply -f https://github.com/cert-manager/cert-manager/releases/download/v1.13.0/cert-manager.yaml

# Create ClusterIssuer
cat <<EOF | kubectl apply -f -
apiVersion: cert-manager.io/v1
kind: ClusterIssuer
metadata:
  name: letsencrypt-prod
spec:
  acme:
    server: https://acme-v02.api.letsencrypt.org/directory
    email: admin@equishare.com
    privateKeySecretRef:
      name: letsencrypt-prod
    solvers:
      - http01:
          ingress:
            class: nginx
EOF
```

## Scaling

### Manual Scaling

```bash
kubectl scale deployment/prod-trading-service \
  --replicas=5 \
  -n equishare-production
```

### HPA Scaling

HPA automatically scales based on CPU/memory:

```bash
# View HPA status
kubectl get hpa -n equishare-production

# Describe specific HPA
kubectl describe hpa prod-api-gateway -n equishare-production
```

## Monitoring Deployment

```bash
# Watch rollout status
kubectl rollout status deployment/prod-api-gateway -n equishare-production

# View pods
kubectl get pods -n equishare-production -l app.kubernetes.io/name=api-gateway

# View logs
kubectl logs -f deployment/prod-api-gateway -n equishare-production

# View events
kubectl get events -n equishare-production --sort-by='.lastTimestamp'
```

## Rollback

```bash
# View rollout history
kubectl rollout history deployment/prod-api-gateway -n equishare-production

# Rollback to previous version
kubectl rollout undo deployment/prod-api-gateway -n equishare-production

# Rollback to specific revision
kubectl rollout undo deployment/prod-api-gateway --to-revision=2 -n equishare-production
```

## Network Policies

Network policies restrict pod-to-pod communication:

- Default deny all ingress
- Allow same-namespace communication
- Allow api-gateway external access
- Allow egress to PostgreSQL, Redis, Kafka
- Allow DNS resolution

## Troubleshooting

### Pod not starting

```bash
kubectl describe pod <pod-name> -n equishare-production
kubectl logs <pod-name> -n equishare-production --previous
```

### Service not accessible

```bash
kubectl get endpoints <service-name> -n equishare-production
kubectl run debug --rm -it --image=busybox -- /bin/sh
# From debug pod: wget -qO- http://service-name:port/health
```

### HPA not scaling

```bash
kubectl describe hpa <hpa-name> -n equishare-production
# Check if metrics-server is running
kubectl get pods -n kube-system | grep metrics-server
```
