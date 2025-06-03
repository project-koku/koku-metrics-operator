# Report Fields For Collected Metrics

This document provides an outline of the fields included in the collected usage metrics. These metrics relate to containers, persistent volumes, nodes, pods, and namespaces.


**NOTE:**

* The [Prometheus queries](https://github.com/project-koku/koku-metrics-operator/blob/main/internal/collector/queries.go) used by the operator to collect metrics are detailed in the linked file.

* To enable the collection ROS (Resource Optimization) metrics, ensure that the namespace(s) are labeled with `insights_cost_management_optimizations='true'`.

    * Queries responsible for collecting ROS metrics are identified in the `QueryMap` with the prefix `ros:` and include a specific filter to target the appropriately labeled namespaces:
        ```
        kube_namespace_labels{label_insights_cost_management_optimizations='true', namespace!~'kube-.*|openshift|openshift-.*'}
        ```


### 1. Common Fields

* `report_period_start`: The start timestamp of the reporting period.
* `report_period_end`: The end timestamp of the reporting period.
* `interval_start`: The start timestamp of the reporting interval.
* `interval_end`: The end timestamp of the reporting interval.

These common fields are included in all the reports and provide temporal information about the reporting period and interval.

---

### 2. Container Metrics

Fields for metrics related to containers:

* `container_name`: The name of the container.
* `pod`: The name of the pod that contains the container.
* `owner_name`: The name of the owner entity (e.g., Deployment, StatefulSet) associated with the container.
* `owner_kind`: The kind of the owner entity (e.g., Deployment, StatefulSet) associated with the container.
* `workload`: The workload associated with the container.
* `workload_type`: The type of the workload (e.g., Deployment, StatefulSet).
* `namespace`: The namespace of the container.
* `image_name`: The name of the container's image.
* `node`: The node on which the container is running.
* `resource_id`: The ID of the resource.
* `cpu_request_container_avg`: Average CPU request for the container.
* `cpu_request_container_sum`: Total CPU request for the container.
* `cpu_limit_container_avg`: Average CPU limit for the container.
* `cpu_limit_container_sum`: Total CPU limit for the container.
* `cpu_usage_container_avg`: Average CPU usage for the container.
* `cpu_usage_container_min`: Minimum CPU usage for the container.
* `cpu_usage_container_max`: Maximum CPU usage for the container.
* `cpu_usage_container_sum`: Total CPU usage for the container.
* `cpu_throttle_container_avg`: Average CPU throttle for the container.
* `cpu_throttle_container_max`: Maximum CPU throttle for the container.
* `cpu_throttle_container_sum`: Total CPU throttle for the container.
* `memory_request_container_avg`: Average memory request for the container.
* `memory_request_container_sum`: Total memory request for the container.
* `memory_limit_container_avg`: Average memory limit for the container.
* `memory_limit_container_sum`: Total memory limit for the container.
* `memory_usage_container_avg`: Average memory usage for the container.
* `memory_usage_container_min`: Minimum memory usage for the container.
* `memory_usage_container_max`: Maximum memory usage for the container.
* `memory_usage_container_sum`: Total memory usage for the container.
* `memory_rss_usage_container_avg`: Average RSS memory usage for the container.
* `memory_rss_usage_container_min`: Minimum RSS memory usage for the container.
* `memory_rss_usage_container_max`: Maximum RSS memory usage for the container.
* `memory_rss_usage_container_sum`: Total RSS memory usage for the container.

---

### 3. Persistent Volume Metrics

Fields for metrics related to persistent volumes:

* `namespace`: The namespace associated with the persistent volume claim.
* `pod`: The name of the pod associated with the persistent volume claim.
* `persistentvolumeclaim`: The name of the persistent volume claim.
* `persistentvolume`: The name of the persistent volume.
* `storageclass`: The storage class of the persistent volume claim.
* `persistentvolumeclaim_capacity_bytes`: Capacity of the persistent volume claim in bytes.
* `persistentvolumeclaim_capacity_byte_seconds`: Capacity of the persistent volume claim in byte seconds.
* `volume_request_storage_byte_seconds`: Storage byte seconds requested by the volume.
* `persistentvolumeclaim_usage_byte_seconds`: Usage byte seconds of the persistent volume claim.
* `persistentvolume_labels`: Labels associated with the persistent volume.
* `persistentvolumeclaim_labels`: Labels associated with the persistent volume claim.

---

### 4. Node Metrics

Fields for metrics related to nodes:

* `node`: The name of the node.
* `node_labels`: Labels associated with the node.

---

### 5. Pod Metrics

Fields for metrics related to pods:

* `node`: The name of the node.
* `namespace`: The namespace of the pod.
* `pod`: The name of the pod.
* `pod_usage_cpu_core_seconds`: CPU core seconds used by the pod.
* `pod_request_cpu_core_seconds`: CPU core seconds requested by the pod.
* `pod_limit_cpu_core_seconds`: CPU core seconds limited for the pod.
* `pod_usage_memory_byte_seconds`: Memory byte seconds used by the pod.
* `pod_request_memory_byte_seconds`: Memory byte seconds requested by the pod.
* `pod_limit_memory_byte_seconds`: Memory byte seconds limited for the pod.
* `node_capacity_cpu_cores`: CPU cores capacity of the node.
* `node_capacity_cpu_core_seconds`: CPU core seconds capacity of the node.
* `node_capacity_memory_bytes`: Memory bytes capacity of the node.
* `node_capacity_memory_byte_seconds`: Memory byte seconds capacity of the node.
* `node_role`: The role of the node.
* `resource_id`: The ID of the resource.
* `pod_labels`: Labels associated with the pod.

---

### 6. Namespace Metrics Report

Fields for metrics related to namespaces:

* `namespace`: The namespace.
* `namespace_labels`: Labels associated with the namespace.

---

### 7. VM Metrics

Fields for metrics related to running virtual machines and their associated resources:

* `node`: The name of the node where the VM instance (VMI) is currently running.
* `namespace`: The namespace where the VM (VM) is defined.
* `resource_id`: The unique identifier of the VM on the node. This is derived from the `provider_id` of the node, specifically the segment after the last `/` character (e.g., if `provider_id` is `aws:///us-east-1a/i-0abcdef1234567890`, `resource_id` would be `i-0abcdef1234567890`).
* `vm_name`: The name of the VM.
* `vm_instance_type`: The instance type associated with the VM, if defined.
* `vm_os`: The operating system reported by the VM's info.
* `vm_guest_os_arch`: The guest operating system architecture reported by the VM (e.g., `x86_64`).
* `vm_guest_os_name`: The guest operating system name reported by the VM (e.g., `RHEL`, `Fedora`).
* `vm_guest_os_version`: The guest operating system version ID reported by the VM (e.g., `8.6`).
* `vm_uptime_total_seconds`: The total uptime of the VMI in seconds since it started.
* `vm_cpu_limit_cores`: CPU core limit configured, representing the maximum number of cores the VM can use.
* `vm_cpu_limit_core_seconds`: Total CPU core seconds limited for the VM over the reporting period.
* `vm_cpu_request_cores`: CPU core request configured for the VM, representing the guaranteed number of cores the VM will receive.
* `vm_cpu_request_core_seconds`: Total CPU core seconds requested for the VM over the reporting period.
* `vm_cpu_request_sockets`: Number of CPU sockets requested for the VM.
* `vm_cpu_request_socket_seconds`: Total CPU socket seconds requested for the VM over the reporting period.
* `vm_cpu_request_threads`: Number of CPU threads requested per core for the VM.
* `vm_cpu_request_thread_seconds`: Total CPU thread seconds requested for the VM over the reporting period.
* `vm_cpu_usage_total_seconds`: Total CPU usage of the VM in seconds over the reporting period.
* `vm_memory_limit_bytes`: Memory limit configured for the VM in bytes, representing the maximum memory the VM can use.
* `vm_memory_limit_byte_seconds`: Total memory byte seconds limited for the VM over the reporting period.
* `vm_memory_request_bytes`: Memory request configured for the VM in bytes, representing the guaranteed memory the VM will receive.
* `vm_memory_request_byte_seconds`: Total memory byte seconds requested for the VM over the reporting period.
* `vm_memory_usage_byte_seconds`: Total memory usage of the VM in byte seconds over the reporting period.
* `vm_device`: Name of the virtual device attached to the VM, typically referring to a disk.
* `vm_volume_mode`: The volume mode of the attached disk (e.g., `Block`, `Filesystem`).
* `vm_persistentvolumeclaim_name`: Name of the PersistentVolumeClaim backing the VM's disk.
* `vm_disk_allocated_size_byte_seconds`: Total allocated disk size for the VM's storage in byte seconds over the reporting period.
* `vm_labels`: A JSON string representing key-value pairs of labels applied to the VM.

---
