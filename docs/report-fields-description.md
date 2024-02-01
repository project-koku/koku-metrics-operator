# Report Fields and Queries

This document provides an overview of the fields included in the usage reports, along with their respective queries. The reports cover various metrics related to containers, persistent volumes, nodes, pods, and namespaces. Each report contains specific fields that provide valuable insights into resource utilization. The following sections outline the fields and queries for each report.

### 1. Common Fields

- `report_period_start`: The start timestamp of the reporting period.
- `report_period_end`: The end timestamp of the reporting period.
- `interval_start`: The start timestamp of the reporting interval.
- `interval_end`: The end timestamp of the reporting interval.

These common fields are included in all the reports and provide temporal information about the reporting period and interval.

### 2. Container Metrics Report

This report focuses on container-related metrics.

- Query: `containerMetricsQuery`

**Fields:**

- `container_name`: The name of the container.
- `pod`: The name of the pod that contains the container.
- `owner_name`: The name of the owner entity (e.g., deployment, statefulset) associated with the container.
- `owner_kind`: The kind of the owner entity (e.g., Deployment, StatefulSet) associated with the container.
- `workload`: The workload associated with the container.
- `workload_type`: The type of the workload (e.g., Deployment, StatefulSet).
- `namespace`: The namespace of the container.
- `image_name`: The name of the container's image.
- `node`: The node on which the container is running.
- `resource_id`: The ID of the resource.
- `cpu_request_container_avg`: Average CPU request for the container.
- `cpu_request_container_sum`: Total CPU request for the container.
- `cpu_limit_container_avg`: Average CPU limit for the container.
- `cpu_limit_container_sum`: Total CPU limit for the container.
- `cpu_usage_container_avg`: Average CPU usage for the container.
- `cpu_usage_container_min`: Minimum CPU usage for the container.
- `cpu_usage_container_max`: Maximum CPU usage for the container.
- `cpu_usage_container_sum`: Total CPU usage for the container.
- `cpu_throttle_container_avg`: Average CPU throttle for the container.
- `cpu_throttle_container_max`: Maximum CPU throttle for the container.
- `cpu_throttle_container_sum`: Total CPU throttle for the container.
- `memory_request_container_avg`: Average memory request for the container.
- `memory_request_container_sum`: Total memory request for the container.
- `memory_limit_container_avg`: Average memory limit for the container.
- `memory_limit_container_sum`: Total memory limit for the container.
- `memory_usage_container_avg`: Average memory usage for the container.
- `memory_usage_container_min`: Minimum memory usage for the container.
- `memory_usage_container_max`: Maximum memory usage for the container.
- `memory_usage_container_sum`: Total memory usage for the container.
- `memory_rss_usage_container_avg`: Average RSS memory usage for the container.
- `memory_rss_usage_container_min`: Minimum RSS memory usage for the container.
- `memory_rss_usage_container_max`: Maximum RSS memory usage for the container.
- `memory_rss_usage_container_sum`: Total RSS memory usage for the container.

### 3. Persistent Volume Metrics Report

This report provides insights into metrics related to persistent volumes.

- Query: `persistentVolumeMetricsQuery`

**Fields:**

- `namespace`: The namespace associated with the persistent volume claim.
- `pod`: The name of the pod associated with the persistent volume claim.
- `persistentvolumeclaim`: The

 name of the persistent volume claim.
- `persistentvolume`: The name of the persistent volume.
- `storageclass`: The storage class of the persistent volume claim.
- `persistentvolumeclaim_capacity_bytes`: Capacity of the persistent volume claim in bytes.
- `persistentvolumeclaim_capacity_byte_seconds`: Capacity of the persistent volume claim in byte seconds.
- `volume_request_storage_byte_seconds`: Storage byte seconds requested by the volume.
- `persistentvolumeclaim_usage_byte_seconds`: Usage byte seconds of the persistent volume claim.
- `persistentvolume_labels`: Labels associated with the persistent volume.
- `persistentvolumeclaim_labels`: Labels associated with the persistent volume claim.

### 4. Node Metrics Report

This report focuses on metrics related to nodes.

- Query: `nodeMetricsQuery`

**Fields:**

- `node`: The name of the node.
- `node_labels`: Labels associated with the node.

### 5. Pod Metrics Report

This report provides metrics specific to pods.

- Query: `podMetricsQuery`

**Fields:**

- `node`: The name of the node.
- `namespace`: The namespace of the pod.
- `pod`: The name of the pod.
- `pod_usage_cpu_core_seconds`: CPU core seconds used by the pod.
- `pod_request_cpu_core_seconds`: CPU core seconds requested by the pod.
- `pod_limit_cpu_core_seconds`: CPU core seconds limited for the pod.
- `pod_usage_memory_byte_seconds`: Memory byte seconds used by the pod.
- `pod_request_memory_byte_seconds`: Memory byte seconds requested by the pod.
- `pod_limit_memory_byte_seconds`: Memory byte seconds limited for the pod.
- `node_capacity_cpu_cores`: CPU cores capacity of the node.
- `node_capacity_cpu_core_seconds`: CPU core seconds capacity of the node.
- `node_capacity_memory_bytes`: Memory bytes capacity of the node.
- `node_capacity_memory_byte_seconds`: Memory byte seconds capacity of the node.
- `node_role`: The role of the node.
- `resource_id`: The ID of the resource.
- `pod_labels`: Labels associated with the pod.

### 6. Namespace Metrics Report

This report focuses on metrics related to namespaces.

- Query: `namespaceMetricsQuery`

**Fields:**

- `namespace`: The namespace.
- `namespace_labels`: Labels associated with the namespace.