# Redis Integration

Redis is implemented as an **optional** infrastructure capability. The application starts, operates, and passes all tests normally when Redis is disabled.

---

## 1. Enabling and Disabling Redis

### Disabled by Default
By default, `REDIS_ENABLED` is `false`. When disabled:
- No Redis client is created or initialized.
- Zero network calls to Redis are made on startup.
- Redis does NOT participate in or affect `/api/v1/health/ready`.
- Unused Redis settings are ignored.

### Enabling Redis
Set `REDIS_ENABLED=true` and supply a valid `REDIS_URL`:
```env
REDIS_ENABLED=true
REDIS_URL=redis://127.0.0.1:6379/0
REDIS_PASSWORD=
REDIS_DB=0
REDIS_POOL_SIZE=10
REDIS_DIAL_TIMEOUT=5s
REDIS_READ_TIMEOUT=3s
REDIS_WRITE_TIMEOUT=3s
```

#### Connection URL by Deployment Context
The `REDIS_URL` must match the network environment where the application binary/container is executing:

- **Host Workstation** (running directly via `go run ./cmd/api`):
  Use the localhost loopback address:
  ```env
  REDIS_URL=redis://127.0.0.1:6379/0
  ```
- **Docker Compose Network** (running both `api` and `redis` containers in Docker Compose):
  Use the service name defined in `compose.yaml`:
  ```env
  REDIS_URL=redis://redis:6379/0
  ```
- **Kubernetes** (running as a pod in a Kubernetes cluster):
  Use the in-cluster Kubernetes Service DNS name:
  ```env
  REDIS_URL=redis://redis-service.default.svc.cluster.local:6379/0
  ```

When enabled:
- Configuration is strictly validated at application startup.
- The client connection pool is verified via an initial `PING`.
- Redis health is automatically included in `/api/v1/health/ready`.
- The client connection pool is gracefully closed during application shutdown.

---

## 2. Health and Readiness Behavior

| Endpoint | When Redis is Disabled | When Redis is Enabled & Reachable | When Redis is Enabled & Unreachable |
|---|---|---|---|
| **`/api/v1/health/live`** | `200 OK` | `200 OK` | `200 OK` (liveness never fails due to Redis) |
| **`/api/v1/health/ready`** | `200 OK` (ignores Redis) | `200 OK` (`"redis": "ready"`) | `503 Service Unavailable` (`"redis": "unavailable"`) |

### Sanitized Health Responses
Readiness check failures sanitize all error messages. Connection strings, hostnames, ports, and credentials are never exposed in JSON responses.

---

## 3. Reusable Cache Abstraction

The package [`internal/platform/redis`](file:///internal/platform/redis) provides a lightweight, JSON-aware `Cache` helper:

```go
import (
    "context"
    "time"

    "github.com/moonrill/go-backend-template/internal/platform/redis"
)

// Initialize cache with the active Redis client
cache := redis.NewCache(redisClient)

// Set value with TTL (structs are automatically marshaled to JSON)
user := MyUser{ID: "123", Name: "Alice"}
err := cache.Set(ctx, "user:123", user, 10*time.Minute)

// Get value into pointer (structs are automatically unmarshaled from JSON)
var cachedUser MyUser
err := cache.Get(ctx, "user:123", &cachedUser)
if errors.Is(err, redis.ErrCacheMiss) {
    // Cache miss: fetch from database and optionally backfill
}

// Delete key
err := cache.Delete(ctx, "user:123")

// Check TTL
remainingTTL, err := cache.TTL(ctx, "user:123")
```

### Module Isolation
Core application modules (such as `users`) do not depend on Redis or the cache abstraction. Modules remain decoupled and independently testable without caching infrastructure.

---

## 4. Docker Compose with Redis

Redis runs under an isolated Docker Compose profile (`redis`) so it never starts in the default configuration:

```bash
# Start only the API (default)
docker compose up -d

# Start API with Redis
docker compose --profile redis up -d

# Start API with both PostgreSQL and Redis
docker compose --profile postgres --profile redis up -d
```

### Security and Storage
- **Network Isolation**: Redis binds to `127.0.0.1:6379` on the host to prevent accidental public exposure.
- **Cache Persistence**: In-memory ephemeral storage is used by default for cache workloads. If persistent append-only logs (`AOF`) or snapshots (`RDB`) are required, add a Docker volume mapping to `/data` in `compose.yaml`.
