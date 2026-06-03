# Architecture

## Overview

The **koku-metrics-operator** is a Kubernetes operator that collects OpenShift cluster usage metrics and uploads them to Red Hat's cost management service (Koku). It runs as a controller that periodically queries Prometheus, generates CSV reports, packages them, and uploads to console.redhat.com.

**Key Responsibilities:**
- Monitor cluster resource usage (pods, nodes, storage, VMs, GPUs, namespaces)
- Query Prometheus/Thanos for metrics data
- Generate cost management reports in CSV format
- Package and upload reports to Red Hat Hybrid Cloud Console
- Manage local storage and report retention
- Integrate with Sources API for cluster registration

## Architecture Diagram

```
┌──────────────────────────────────────────────────────────────────────────┐
│                           OpenShift Cluster                              │
│                                                                          │
│  ┌────────────────┐   ┌──────────────────┐   ┌───────────────────────┐  │
│  │  Prometheus/   │   │  ClusterVersion  │   │  openshift-config/    │  │
│  │  Thanos        │   │  API             │   │  pull-secret          │  │
│  └───────┬────────┘   └────────┬─────────┘   └───────────┬───────────┘  │
│          │                     │                          │              │
│          │ Metrics Queries     │ Cluster ID/Version       │ Auth Token   │
│          ▼                     ▼                          ▼              │
│  ┌───────────────────────────────────────────────────────────────────┐   │
│  │                    koku-metrics-operator                          │   │
│  │                    (managed by OLM)                               │   │
│  │                                                                   │   │
│  │  ┌─────────────┐   ┌──────────────┐                              │   │
│  │  │ Controller  │──►│  Collector   │                              │   │
│  │  │ Reconciler  │   │ (Prometheus) │                              │   │
│  │  │ (5 min loop)│   └──────┬───────┘                              │   │
│  │  └──────┬──────┘          │                                      │   │
│  │         │                 ▼                                      │   │
│  │         │          ┌─────────────┐   ┌──────────────┐            │   │
│  │         │          │  Storage    │   │  Packaging   │            │   │
│  │         │          │  (PVC/tmp)  │   │  (tar.gz)    │            │   │
│  │         │          │             │   └──────┬───────┘            │   │
│  │         │          │  staging/ ──┼──►read──►│                    │   │
│  │         │          │  upload/ ◄──┼──write───┘                    │   │
│  │         │          │  data/      │                               │   │
│  │         │          └─────────────┘                               │   │
│  │         │                                                        │   │
│  └─────────┼────────────────────────────────────────────────────────┘   │
│            │                                                            │
└────────────┼────────────────────────────────────────────────────────────┘
             │ HTTPS
             ▼
┌───────────────────────────────┐
│     console.redhat.com        │
│                               │
│  ┌─────────────────────────┐  │
│  │  Ingress API            │◄─── Upload tar.gz reports
│  │  /api/ingress/v1/upload │  │
│  └─────────────────────────┘  │
│  ┌─────────────────────────┐  │
│  │  Sources API            │◄─── Credential validation,
│  │  /api/sources/v1.0/     │     source registration
│  └─────────────────────────┘  │
│  ┌─────────────────────────┐  │
│  │  SSO Token Exchange     │◄─── Service-account auth
│  └─────────────────────────┘  │
└───────────────────────────────┘
```

## Components

### 1. Custom Resource Definition (CRD)

**Location:** `api/v1beta1/metricsconfig_types.go`

Defines the `CostManagementMetricsConfig` custom resource that configures the operator.

**Key Specs:**
- **`authentication`**: How to authenticate (token, service-account)
- **`prometheus_config`**: Prometheus endpoint and connection settings
- **`upload`**: Upload schedule, validation, and API path
- **`packaging`**: Max file size and report retention
- **`source`**: Sources API integration settings
- **`api_url`** (optional): Override console.redhat.com API URL (development use)

**Status Fields:**
- Last collection/upload timestamps
- Authentication validation status
- Prometheus connection status
- Upload history
- Source registration status

### 2. Controller (Reconciler)

**Location:** `internal/controller/costmanagementmetricsconfig_controller.go`

The main reconciliation loop that orchestrates all operations.

**Responsibilities:**
- Watch for CostManagementMetricsConfig CR changes
- Validate and configure authentication
- Reconcile every 5 minutes; packaging and upload are gated by `upload_cycle` (default: 360 minutes / 6 hours)
- Trigger report generation and upload
- Update CR status with results
- Manage PVC lifecycle
- Integrate with Sources API

**Key Functions:**
- `Reconcile()`: Main reconciliation loop (requeues every 5 minutes)
- `setAuthentication()`: Configure auth (token/service-account)
- `validateCredentials()`: Check auth against Sources API
- `collectPromStats()`: Trigger Prometheus collection (package-level function in `prometheus.go`)
- `uploadFiles()`: Upload packaged reports
- `configurePVC()`: Set up persistent storage (package-level function)

**Reconciliation Triggers:**
- CostManagementMetricsConfig CR creation/update
- Scheduled intervals (upload schedule, source check)
- Manual reconciliation requests

### 3. Prometheus Collector

**Location:** `internal/collector/`

Queries Prometheus/Thanos for cluster metrics and generates reports.

**Files:**
- `collector.go`: Report generation logic
- `prometheus.go`: Prometheus client and connection
- `queries.go`: Prometheus query definitions
- `report.go`: CSV report formatting
- `types.go`: Data structures

**Metrics Collected:**
- **Pod usage**: CPU, memory, resource requests/limits
- **Node usage**: Allocatable resources, capacity, labels
- **Storage (PVC)**: Capacity, requests, storage class
- **Namespace**: Resource quotas and usage
- **Virtual Machines**: vCPU, memory for KubeVirt VMs
- **NVIDIA GPU**: GPU usage and allocation
- **ROS (Resource Optimization)**: Container and namespace recommendations

**Query Pattern:**
- Time ranges: UTC, truncated to hour boundaries
- Aggregation: max, min, avg, sum (see `getValue()`)
- Data windowing: Previous full hour by default
- Query lookback window: Auto-detected from cluster monitoring config; defaults to 14 days if unavailable (this is the Prometheus query range, not on-disk report retention)

**Report Generation:**
```go
GenerateReports(cr *MetricsConfig, dirCfg *DirectoryConfig, collector *PrometheusCollector)
```
> Note: `MetricsConfig` here is a type alias for `CostManagementMetricsConfig`, used throughout the codebase for brevity.
1. Query Prometheus for each metric type
2. Process and aggregate results
3. Generate CSV files with prefixes:
   - `cm-openshift-pod-usage-*.csv`
   - `cm-openshift-node-usage-*.csv`
   - `cm-openshift-storage-usage-*.csv`
   - `cm-openshift-namespace-usage-*.csv`
   - `cm-openshift-vm-usage-*.csv`
   - `cm-openshift-nvidia-gpu-usage-*.csv`
   - `ros-openshift-container-*.csv`
   - `ros-openshift-namespace-*.csv`
4. Write to staging directory

### 4. Storage Management

**Location:** `internal/storage/`, `internal/dirconfig/`

Manages local filesystem for reports.

**Directory Structure:**
```
/tmp/koku-metrics-operator-reports/  (or PVC mount)
├── upload/          # Packaged reports ready for upload
├── staging/         # Newly generated CSV reports
└── data/            # Working directory
```

**DirectoryConfig:**
- Configures paths based on PVC or temporary storage
- Manages directory creation and cleanup
- Report rotation based on `max_reports_to_store` setting

**Storage Options:**
- **PVC (preferred)**: Persistent storage survives pod restarts
- **Temporary**: `/tmp/` fallback if PVC not configured

### 5. Packaging

**Location:** `internal/packaging/packaging.go`

Compresses reports into tar.gz archives for upload.

**FilePackager:**
- Bundles multiple CSV files into single archive
- Splits by size limit (default: 100MB)
- Adds manifest with metadata (cluster ID, timestamps, file list)
- Generates unique upload filenames
- Moves completed packages to upload directory

**Package Format:**
```
<timestamp>-cost-mgmt.tar.gz          # single archive
<timestamp>-cost-mgmt-<index>.tar.gz  # split archives
├── manifest.json
├── cm-openshift-pod-usage-*.csv
├── cm-openshift-node-usage-*.csv
└── ... (other CSV files)
```

### 6. HTTP Client (Upload)

**Location:** `internal/crhchttp/`

Handles authentication and upload to console.redhat.com.

**AuthConfig:**
Supports two authentication methods:
- **Token** (default): Uses cluster pull secret token from `openshift-config/pull-secret`
- **Service Account**: Client ID/secret from custom secret

> Basic auth (username/password) exists in the codebase but is deprecated and should not be used.

**Upload Process:**
```go
Upload(authConfig, contentType, method, uri, body, fileInfo, file)
```
1. Authenticate based on auth type (token refresh if needed)
2. POST to Ingress API (`/api/ingress/v1/upload`)
3. Include headers: cluster ID, file metadata
4. Return upload status and request ID
5. Clean up uploaded files on success

**API Endpoints:**
- **Ingress API**: `https://console.redhat.com/api/ingress/v1/upload`
- **Sources API**: `https://console.redhat.com/api/sources/v1.0/`
- **Token endpoint**: `https://sso.redhat.com/auth/realms/redhat-external/protocol/openid-connect/token`

### 7. Sources API Integration

**Location:** `internal/sources/`

Manages cluster registration with Red Hat Sources API.

**Responsibilities:**
- Check if cluster is registered as a cost management source
- Validate authentication credentials
- Report source creation status in CR status
- Periodic checks (default: every 24 hours)

**Source Check Flow:**
1. Authenticate with Sources API
2. Query for source matching cluster ID
3. Update `source_defined` status in CR
4. Create source if `create_source: true` and missing (experimental)

### 8. Cluster Version Detection

**Location:** `internal/clusterversion/`

Queries OpenShift ClusterVersion resource.

**Purpose:**
- Detect OpenShift version
- Retrieve cluster ID
- Include cluster metadata in reports

### 9. OLM Integration

The operator is installed and managed via the [Operator Lifecycle Manager (OLM)](https://olm.operatorframework.io/).

**ClusterServiceVersion (CSV):**
- Base manifest: `config/manifests/bases/koku-metrics-operator.clusterserviceversion.yaml`
- Generated bundle: `bundle/manifests/koku-metrics-operator.clusterserviceversion.yaml` (produced by `make bundle`, not committed)
- Defines operator metadata, RBAC, deployment spec, and upgrade path (`spec.replaces`)

**Bundle Generation:**
```bash
make bundle          # generate OLM bundle from config/manifests
make bundle-build    # build bundle container image
make bundle-push     # push bundle image to registry
```

**OLM in the Codebase:**
- `internal/storage/storage.go` reads the owning `ClusterServiceVersion` to patch the operator Deployment's volume mounts when PVC storage is configured. This ensures the Deployment spec in the CSV stays consistent with runtime changes.
- RBAC markers on the reconciler include permissions for `clusterserviceversions` under `operators.coreos.com`.

**Distribution:**
- Upstream: Submitted to [community-operators-prod](https://github.com/redhat-openshift-ecosystem/community-operators-prod) for OperatorHub
- Downstream: Managed by Red Hat Konflux build system

## Data Flow

### High-Level Flow

```
1. CR Created/Updated
   ↓
2. Controller Reconcile (every 5 minutes)
   ↓
3. Authenticate (token/service-account)
   ↓
4. Collect Metrics (query Prometheus → generate CSV reports)
   ↓
5. [Gated by upload_cycle, default 6 hours]
   Package Reports (tar.gz) → Upload to console.redhat.com
   ↓
6. Clean up old reports
   ↓
7. [Gated by check_cycle, default 24 hours]
   Check Sources API
   ↓
8. Update CR Status
```

### Detailed Collection Flow

```
Reconcile()
  │
  ├─► setAuthentication()
  │   ├─► GetPullSecretToken() [token auth]
  │   └─► GetServiceAccountSecret() [service-account auth]
  │
  ├─► validateCredentials() [Sources API check]
  │
  ├─► collectPromStats()
  │   ├─► getPromCollector()
  │   │   └─► Create Prometheus client
  │   │
  │   └─► collector.GenerateReports()
  │       ├─► Query Prometheus for each metric type
  │       ├─► Process and aggregate data
  │       └─► Write CSV reports to staging/
  │
  ├─► Packaging
  │   ├─► packager.PackageReports()
  │   ├─► Create tar.gz archives
  │   └─► Move to upload/ directory
  │
  ├─► uploadFiles()
  │   ├─► crhchttp.Upload()
  │   ├─► POST to Ingress API
  │   └─► Delete uploaded files
  │
  ├─► storage.RemoveOldReports()
  │   └─► Clean up based on max_reports_to_store
  │
  └─► updateStatusAndLogError()
      └─► Update CR status with results
```

## Configuration and Scheduling

### Time Schedules

**Reconciliation Loop:** Every 5 minutes (`RequeueAfter: 5 * time.Minute`)

**Upload Cycle** (default: 360 minutes / 6 hours)
- Configured via `upload.upload_cycle`
- Gates when packaging and upload steps are allowed to run

**Source Check Cycle** (default: 1440 minutes / 24 hours)
- Configured via `source.check_cycle`
- Validates cluster registration

**Time Ranges:**
- Metrics collected for previous full hour (UTC)
- Example: At 14:32 UTC → collect 13:00:00 to 13:59:59

### Storage Limits

**Packaging Max Size:** 100 MB (configurable via `packaging.max_size_MB`)
- Reports split into multiple archives if exceeded

**Max Reports:** 30 (configurable via `packaging.max_reports_to_store`)
- Corresponds to ~7 days of data with default 6-hour cycle
- Older reports automatically deleted

## Authentication Flow

### Token Authentication (Default)

```
1. Read openshift-config/pull-secret
2. Extract cloud.openshift.com token
3. Use token for API requests
4. No refresh needed (cluster token)
```

### Service Account Authentication

```
1. Read custom secret (client_id + client_secret)
2. Exchange for OAuth token via SSO endpoint
3. Use token for API requests
4. Refresh when expired (TTL check)
```

### Basic Authentication (Deprecated -- do not use)

Basic auth is still supported in the codebase for backward compatibility but should not be used for new deployments. Use token or service-account auth instead.

## Error Handling and Retry

**Prometheus Connection Errors:**
- Log error and update CR status
- Retry on next reconciliation cycle

**Authentication Failures:**
- Cache validation results to avoid repeated checks
- Update CR status with auth error
- Retry after cooldown period

**Upload Failures:**
- Keep reports in upload/ directory
- Retry on next cycle
- Eventually cleaned up by max_reports limit

**No Data Collected:**
- Return `ErrNoData` if Prometheus has no metrics
- Mark collection as skipped in status
- Continue to next cycle

## Security Considerations

**Secrets Management:**
- Pull secret: Read-only access to `openshift-config/pull-secret`
- Custom secrets: Namespaced to operator namespace
- Service account tokens: Refreshed automatically

**RBAC Requirements:**
- `cluster-monitoring-view` ClusterRole for Prometheus/Thanos access
- Read access to `ClusterVersion` (`config.openshift.io`)
- Read access to `openshift-config/pull-secret` (cross-namespace, for token auth)
- Read access to cluster monitoring ConfigMap (for query lookback window detection)
- Read/write access to `ClusterServiceVersions` (`operators.coreos.com`, for PVC volume mount patching)
- PVC creation/management in operator namespace

**Network:**
- Outbound HTTPS to console.redhat.com
- Internal access to Prometheus/Thanos endpoint
- TLS verification configurable (skip for development)

## Operational Characteristics

- Runs as a single pod managed by OLM
- CPU/memory usage spikes during collection, low between cycles
- Prometheus queries are read-only against hourly aggregated data
- Report size scales with cluster resource count; archives are automatically split at 100MB
- Storage footprint bounded by `max_reports_to_store` (default: 30)

## Development and Testing

See [local-development.md](local-development.md) for setup, workflow, and troubleshooting.

**CI/CD:**
- Upstream: GitHub Actions (`.github/workflows/`)
- Downstream: Konflux/Tekton (`.tekton/`)

## Related Documentation

- **[local-development.md](local-development.md)** - Setup, workflow, and PR process
- **[upstream-releasing.md](upstream-releasing.md)** - Release process
- **[downstream-releasing.md](downstream-releasing.md)** - Downstream porting
- **[report-fields-description.md](report-fields-description.md)** - CSV field definitions
