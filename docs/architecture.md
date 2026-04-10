# Architecture

> **For AI-assisted development context**, see [CLAUDE.md](koku-metrics-operator/.claude/CLAUDE.md). This document describes the system architecture, component relationships, and data flow.

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
┌─────────────────────────────────────────────────────────────────┐
│                    OpenShift Cluster                            │
│                                                                 │
│  ┌────────────────┐      ┌──────────────────┐                 │
│  │  Prometheus/   │◄─────│  Cluster         │                 │
│  │  Thanos        │      │  Resources       │                 │
│  └────────┬───────┘      └──────────────────┘                 │
│           │                                                     │
│           │ Metrics Queries                                    │
│           ▼                                                     │
│  ┌──────────────────────────────────────────────────────────┐ │
│  │         koku-metrics-operator                            │ │
│  │                                                          │ │
│  │  ┌─────────────┐   ┌──────────────┐   ┌─────────────┐  │ │
│  │  │ Controller  │──►│  Collector   │──►│   Reports   │  │ │
│  │  │ Reconciler  │   │ (Prometheus) │   │ (CSV files) │  │ │
│  │  └─────────────┘   └──────────────┘   └──────┬──────┘  │ │
│  │         │                                     │         │ │
│  │         │                                     ▼         │ │
│  │         │           ┌──────────────┐   ┌─────────────┐ │ │
│  │         └──────────►│  Packaging   │──►│   Storage   │ │ │
│  │                     │  (tar.gz)    │   │  (PVC/tmp)  │ │ │
│  │                     └──────┬───────┘   └─────────────┘ │ │
│  │                            │                           │ │
│  └────────────────────────────┼───────────────────────────┘ │
│                               │                             │
└───────────────────────────────┼─────────────────────────────┘
                                │ HTTPS Upload
                                ▼
                    ┌───────────────────────────┐
                    │  console.redhat.com       │
                    │  ┌─────────────────────┐  │
                    │  │  Ingress API        │  │
                    │  └─────────────────────┘  │
                    │  ┌─────────────────────┐  │
                    │  │  Sources API        │  │
                    │  └─────────────────────┘  │
                    └───────────────────────────┘
```

## Components

### 1. Custom Resource Definition (CRD)

**Location:** `api/v1beta1/metricsconfig_types.go`

Defines the `MetricsConfig` (alias: `CostManagementMetricsConfig`) custom resource that configures the operator.

**Key Specs:**
- **`authentication`**: How to authenticate (token, basic, service-account)
- **`prometheus_config`**: Prometheus endpoint and connection settings
- **`upload`**: Upload schedule, validation, and API path
- **`packaging`**: Max file size and report retention
- **`source`**: Sources API integration settings
- **`volume`**: PersistentVolumeClaim configuration for storage

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
- Watch for MetricsConfig CR changes
- Validate and configure authentication
- Schedule periodic metric collection (default: every 6 hours)
- Trigger report generation and upload
- Update CR status with results
- Manage PVC lifecycle
- Integrate with Sources API

**Key Functions:**
- `Reconcile()`: Main reconciliation loop
- `setAuthentication()`: Configure auth (token/basic/service-account)
- `validateCredentials()`: Check auth against Sources API
- `collectPromStats()`: Trigger Prometheus collection
- `uploadFiles()`: Upload packaged reports
- `configurePVC()`: Set up persistent storage

**Reconciliation Triggers:**
- MetricsConfig CR creation/update
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
- Retention period: Auto-detected from Prometheus config (default 14 days)

**Report Generation:**
```go
GenerateReports(cr *MetricsConfig, dirCfg *DirectoryConfig, collector *PrometheusCollector)
```
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
cost-mgmt-<cluster_id>-<timestamp>-<uuid>.tar.gz
├── manifest.json
├── cm-openshift-pod-usage-*.csv
├── cm-openshift-node-usage-*.csv
└── ... (other CSV files)
```

### 6. HTTP Client (Upload)

**Location:** `internal/crhchttp/`

Handles authentication and upload to console.redhat.com.

**AuthConfig:**
Supports multiple authentication methods:
- **Token** (default): Uses cluster pull secret token from `openshift-config/pull-secret`
- **Service Account**: Client ID/secret from custom secret
- **Basic** (deprecated): Username/password

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

## Data Flow

### High-Level Flow

```
1. CR Created/Updated
   ↓
2. Controller Reconcile
   ↓
3. Authenticate (token/service-account/basic)
   ↓
4. [Every 6 hours] Collect Metrics
   ↓
5. Query Prometheus → Generate CSV Reports
   ↓
6. Package Reports (tar.gz)
   ↓
7. Upload to console.redhat.com
   ↓
8. Clean up old reports
   ↓
9. [Every 24 hours] Check Sources API
   ↓
10. Update CR Status
```

### Detailed Collection Flow

```
Reconcile()
  │
  ├─► setAuthentication()
  │   ├─► GetPullSecretToken() [token auth]
  │   ├─► GetServiceAccountSecret() [service-account auth]
  │   └─► GetAuthSecret() [basic auth]
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

**Upload Cycle** (default: 360 minutes / 6 hours)
- Configured via `upload.upload_wait`
- Triggers metric collection and upload

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

### Basic Authentication (Deprecated)

```
1. Read custom secret (username + password)
2. Use basic auth for API requests
3. No token exchange
```

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
- Access to Prometheus/Thanos (cluster-monitoring-view)
- Read access to ClusterVersion
- PVC creation/management in operator namespace

**Network:**
- Outbound HTTPS to console.redhat.com
- Internal access to Prometheus/Thanos endpoint
- TLS verification configurable (skip for development)

## Performance and Scalability

**Resource Usage:**
- Lightweight: Runs as single pod
- CPU/Memory: Spikes during collection, low between cycles
- Storage: Depends on cluster size and report count

**Cluster Impact:**
- Prometheus queries: Minimal impact (read-only, hourly data)
- API calls: Periodic, not continuous
- No impact on cluster workloads

**Scalability:**
- Handles clusters of any size
- Report size grows with cluster resources
- Automatic report splitting by size

## Development and Testing

**Local Development:**
- See [local-development.md](local-development.md)
- Requires OpenShift cluster access
- Can run outside cluster with proper credentials

**Testing:**
- Unit tests: Ginkgo/Gomega
- Mocked Prometheus responses
- Fake Kubernetes client for controller tests

**CI/CD:**
- Upstream: GitHub Actions (`.github/workflows/`)
- Downstream: Konflux/Tekton (`.tekton/`)

## Related Documentation

- **[local-development.md](local-development.md)** - Setup and local testing
- **[upstream-releasing.md](upstream-releasing.md)** - Release process
- **[downstream-releasing.md](downstream-releasing.md)** - Downstream porting
- **[report-fields-description.md](report-fields-description.md)** - CSV field definitions
- **[../.claude/CLAUDE.md](../.claude/CLAUDE.md)** - AI development context
