# MinIO / S3-Compatible Object Storage

MinIO is implemented as an **optional** infrastructure capability. The application starts, operates, and passes all tests normally when MinIO is disabled.

---

## 1. Enabling and Disabling MinIO

### Disabled by Default
By default, `MINIO_ENABLED` is `false`. When disabled:
- No MinIO client is created or initialized.
- Zero network calls to MinIO/S3 are made on startup.
- MinIO does NOT participate in or affect `/api/v1/health/ready`.
- Unused MinIO settings are ignored.

### Enabling MinIO
Set `MINIO_ENABLED=true` and supply required configuration:
```env
MINIO_ENABLED=true
MINIO_ENDPOINT=127.0.0.1:9000
MINIO_REGION=us-east-1
MINIO_BUCKET=app-bucket
MINIO_ACCESS_KEY=minioadmin
MINIO_SECRET_KEY=minioadmin
MINIO_USE_SSL=false
MINIO_CONNECT_TIMEOUT=5s
MINIO_AUTO_CREATE_BUCKET=true
```

#### Connection Endpoint by Deployment Context
The `MINIO_ENDPOINT` must match the network environment where the application binary/container is executing:

- **Host Workstation** (running directly via `go run ./cmd/api`):
  ```env
  MINIO_ENDPOINT=127.0.0.1:9000
  MINIO_USE_SSL=false
  ```
- **Docker Compose Network** (running both `api` and `minio` in Docker Compose):
  ```env
  MINIO_ENDPOINT=minio:9000
  MINIO_USE_SSL=false
  ```
- **Kubernetes** (running as a pod in a Kubernetes cluster):
  ```env
  MINIO_ENDPOINT=minio-service.default.svc.cluster.local:9000
  MINIO_USE_SSL=false
  ```
- **AWS S3 / Production Cloud**:
  ```env
  MINIO_ENDPOINT=s3.us-east-1.amazonaws.com
  MINIO_USE_SSL=true
  ```

> [!NOTE]
> Do not prefix `MINIO_ENDPOINT` with `http://` or `https://`. Use `MINIO_USE_SSL=true` to enable TLS.

When enabled:
- Configuration is strictly validated at application startup.
- The client verifies connectivity and bucket accessibility.
- If `MINIO_AUTO_CREATE_BUCKET=true` and the bucket does not exist, it is created automatically.
- MinIO health is automatically included in `/api/v1/health/ready`.

---

## 2. Health and Readiness Behavior

| Endpoint | When MinIO is Disabled | When MinIO is Enabled & Reachable | When MinIO is Enabled & Unreachable |
|---|---|---|---|
| **`/api/v1/health/live`** | `200 OK` | `200 OK` | `200 OK` (liveness never fails due to MinIO) |
| **`/api/v1/health/ready`** | `200 OK` (ignores MinIO) | `200 OK` (`"minio": "ready"`) | `503 Service Unavailable` (`"minio": "unavailable"`) |

### Sanitized Health Responses
Readiness check failures sanitize all error messages. Access keys, secret keys, and internal credentials are never exposed in JSON responses.

---

## 3. Reusable Storage Abstraction

The package [`internal/platform/storage/minio`](file:///internal/platform/storage/minio) provides a clean, decoupled `Storage` interface:

```go
type Storage interface {
    PutObject(ctx context.Context, key string, reader io.Reader, size int64, contentType string) error
    GetObject(ctx context.Context, key string) (io.ReadCloser, error)
    StatObject(ctx context.Context, key string) (*ObjectInfo, error)
    DeleteObject(ctx context.Context, key string) error
    PresignedGetURL(ctx context.Context, key string, expiry time.Duration) (string, error)
    PresignedPutURL(ctx context.Context, key string, expiry time.Duration) (string, error)
}
```

### Example Usage in a Module

```go
import (
    "context"
    "strings"
    "time"

    "github.com/moonrill/go-backend-template/internal/platform/storage/minio"
)

// Upload an object
content := "Hello MinIO"
err := storage.PutObject(ctx, "documents/hello.txt", strings.NewReader(content), int64(len(content)), "text/plain")

// Download an object
rc, err := storage.GetObject(ctx, "documents/hello.txt")
if err != nil {
    if errors.Is(err, minio.ErrObjectNotFound) {
        // Handle object not found
    }
}
defer rc.Close()

// Generate presigned download URL for direct client access
url, err := storage.PresignedGetURL(ctx, "documents/hello.txt", 15*time.Minute)

// Generate presigned upload URL for direct client upload
uploadURL, err := storage.PresignedPutURL(ctx, "uploads/avatar.jpg", 15*time.Minute)

// Delete an object
err = storage.DeleteObject(ctx, "documents/hello.txt")
```

---

## 4. Docker Compose with MinIO

MinIO runs under an isolated Docker Compose profile (`minio`) so it never starts in the default configuration:

```bash
# Start only the API (default)
docker compose up -d

# Start API with MinIO
docker compose --profile minio up -d

# Start API with PostgreSQL, Redis, and MinIO
docker compose --profile postgres --profile redis --profile minio up -d
```

### Ports and Console
- **MinIO S3 API**: `127.0.0.1:9000`
- **MinIO Web Console**: `127.0.0.1:9001` (login with `MINIO_ROOT_USER` / `MINIO_ROOT_PASSWORD`, default `minioadmin` / `minioadmin`)

### Security and Storage
- **Network Isolation**: Ports bind to `127.0.0.1` to prevent public host network exposure.
- **Persistence**: Data is persisted in Docker named volume `minio_data`.
