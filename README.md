# Octelium initializer

A Kubernetes init container that prepares `ClusterConfig/default` before the Octelium resource server starts. It connects directly to PostgreSQL and Redis using Octelium's storage implementation, creates a minimal ClusterConfig with the supplied domain and upstream networking defaults, and reuses that configuration on subsequent starts.

The initializer exits successfully when initialization and optional ConfigMap publication finish. Failed attempts are retried every two seconds until the configured timeout. SIGINT and SIGTERM cancel initialization and close the storage connections.

## Configuration

| Flag | Environment variable | Default | Purpose |
| --- | --- | --- | --- |
| `--domain` | `OCTELIUM_CLUSTER_DOMAIN` | Empty | Domain to store when creating the ClusterConfig; required for initial creation. |
| `--cluster-config-map` | `OCTELIUM_INITIALIZER_CLUSTER_CONFIG_MAP` | Empty | Existing ConfigMap to receive the ClusterConfig identity. Set a name to enable publication. |
| `--timeout` | — | `2m` | Total initialization deadline, including retries. |

Flags take precedence over environment variables. The domain is stored in `ClusterConfig.status.domain`, matching Octelium's bootstrap API. The ClusterConfig's generated UUID persists across starts.

Configure the same storage settings as the resource server:

| Environment variables | Purpose |
| --- | --- |
| `OCTELIUM_POSTGRES_HOST`, `OCTELIUM_POSTGRES_PORT` | PostgreSQL address; port defaults to `5432`. |
| `OCTELIUM_POSTGRES_DATABASE`, `OCTELIUM_POSTGRES_USERNAME`, `OCTELIUM_POSTGRES_PASSWORD` | Existing database and credentials. |
| `OCTELIUM_POSTGRES_NOSSL` | Set to `true` to disable PostgreSQL TLS. |
| `OCTELIUM_REDIS_HOST`, `OCTELIUM_REDIS_PORT` | Redis address; port defaults to `6379`. |
| `OCTELIUM_REDIS_DATABASE`, `OCTELIUM_REDIS_USERNAME`, `OCTELIUM_REDIS_PASSWORD` | Redis database index and credentials. |
| `OCTELIUM_REDIS_USE_TLS` | Set to `true` to enable Redis TLS. |

Supply passwords through Kubernetes Secrets. Both storage services must be reachable within the initialization timeout. The upstream storage constructor performs its usual table and index migrations.

## Run as an init container

Build or pull an initializer image, then add it to the resource-server Pod. Store the `OCTELIUM_POSTGRES_*` and `OCTELIUM_REDIS_*` settings in a Secret named `octelium-storage`:

```yaml
spec:
  template:
    spec:
      initContainers:
        - name: octelium-initializer
          image: ghcr.io/wyndure/octelium-initializer:dev
          args: ["--domain=access.example.com", "--timeout=2m"]
          envFrom:
            - secretRef:
                name: octelium-storage
          securityContext:
            runAsNonRoot: true
            runAsUser: 65532
            runAsGroup: 65532
            readOnlyRootFilesystem: true
            allowPrivilegeEscalation: false
            capabilities:
              drop: [ALL]
      containers:
        - name: rscserver
          image: ghcr.io/octelium/octelium-rscserver:0.47.0
          envFrom:
            - secretRef:
                name: octelium-storage
```

Kubernetes starts the resource-server container after initialization succeeds. The initializer image runs as UID/GID `65532:65532`.

## Publish the ClusterConfig identity

Create a ConfigMap in the Pod's namespace and set `--cluster-config-map=octelium-cluster`. The initializer updates these fixed data keys on every successful start:

```yaml
data:
  name: default
  uid: <ClusterConfig UUID>
```

Other data keys are preserved. The initializer uses the Pod's service-account credentials and reads its namespace from `POD_NAMESPACE`, falling back to the projected service-account namespace file. Mount those credentials when publication is enabled and grant `patch` permission on the named ConfigMap:

```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: octelium-cluster
---
apiVersion: rbac.authorization.k8s.io/v1
kind: Role
metadata:
  name: octelium-initializer
rules:
  - apiGroups: [""]
    resources: [configmaps]
    resourceNames: [octelium-cluster]
    verbs: [patch]
```

Bind the Role to the resource-server Pod's ServiceAccount using a RoleBinding in the same namespace. ConfigMap publication participates in the initialization deadline, so the resource server starts after the identity is published.

## Build and test

Requires Go 1.27.3 or newer. Dependencies are pinned to published modules compatible with Octelium 0.47.0.

```sh
go mod download
go test -race ./...
sh ci/test-image-tags.sh
go build -buildvcs=false -o bin/initializer .
./bin/initializer --help
```

Run `golangci-lint run` with golangci-lint v2.15.0 for static checks. Race tests require a C compiler; `docker build --target test .` runs them in the Go build image.

Build the container with Docker BuildKit:

```sh
docker build -t ghcr.io/wyndure/octelium-initializer:dev .
```

## License

GNU Affero General Public License v3.0. See [LICENSE-AGPL-3.0](LICENSE-AGPL-3.0).

## Legal

This project is a community endeavour and is not associated with Octelium Labs, LLC.

Octelium and Octelium logo are trademarks of Octelium Labs, LLC.

WireGuard is a registered trademark of Jason A. Donenfeld.
