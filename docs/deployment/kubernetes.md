# Kubernetes deployment

Minimal Deployment + Service + Secret for a single-account setup. For a
multi-account pool with a management UI, use CLIProxyAPI instead.

## Manifest

`examples/kubernetes/cursor-proxy.yaml` — apply with `kubectl apply -f`.

```yaml
apiVersion: v1
kind: Secret
metadata:
  name: cursor-proxy-secrets
type: Opaque
stringData:
  # comma-separated allowlist
  CURSOR_PROXY_API_KEYS: "sk-cp-CHANGE-ME"
---
apiVersion: v1
kind: ConfigMap
metadata:
  name: cursor-proxy-account
data:
  # paste your cursor-<email>.json (base64 or stringified) here
  current.json: |
    { "...": "your CPA-format auth JSON" }
---
apiVersion: apps/v1
kind: Deployment
metadata:
  name: cursor-proxy
spec:
  replicas: 1
  selector: { matchLabels: { app: cursor-proxy } }
  template:
    metadata: { labels: { app: cursor-proxy } }
    spec:
      containers:
        - name: cursor-proxy
          image: ghcr.io/greensheep999/cursor-proxy:cursor3.11-v0.3.7
          imagePullPolicy: Always
          ports:
            - containerPort: 8317
          env:
            - name: CURSOR_PROXY_ACCOUNT_FILE
              value: /data/accounts/current.json
          envFrom:
            - secretRef:
                name: cursor-proxy-secrets
          volumeMounts:
            - name: account
              mountPath: /data/accounts
              readOnly: true
      volumes:
        - name: account
          configMap:
            name: cursor-proxy-account
---
apiVersion: v1
kind: Service
metadata:
  name: cursor-proxy
spec:
  selector: { app: cursor-proxy }
  ports:
    - port: 8317
      targetPort: 8317
```

## Access

Inside the cluster:

```
http://cursor-proxy.<namespace>.svc.cluster.local:8317
```

Expose externally with an Ingress (nginx / Caddy / Traefik) — always
behind TLS, always with `CURSOR_PROXY_API_KEYS` set.

## Rotating the auth file

Update the `cursor-proxy-account` ConfigMap and roll the Deployment:

```bash
kubectl create configmap cursor-proxy-account --from-file=current.json \
  --dry-run=client -o yaml | kubectl apply -f -
kubectl rollout restart deployment/cursor-proxy
```

## Health / readiness

`GET /v1/models` is a cheap authenticated liveness probe. `GET /v1/usage`
also works and additionally exercises the account.
