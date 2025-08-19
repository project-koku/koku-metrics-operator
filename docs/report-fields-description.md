# Report Fields For Collected Metrics

This document provides an outline of the fields included in the collected usage metrics. These metrics relate to containers, persistent volumes, nodes, pods, and namespaces.


**NOTE:**

* The [Prometheus queries](https://github.com/project-koku/koku-metrics-operator/blob/main/internal/collector/queries.go) that the operator uses to collect metrics are detailed in the linked file.

* To enable the collection ROS (Resource Optimization) metrics, ensure that the namespaces are labeled with `cost_management_optimizations='true'`. Note in operator versions below 4.1.0 you must use the label `insights_cost_management_optimizations='true'`.

    * Queries responsible for collecting ROS metrics are identified in the `QueryMap` with the prefix `ros:` and include a specific filter to target the appropriately labeled namespaces:
        ```
        kube_namespace_labels{label_cost_management_optimizations='true', namespace!~'kube-.*|openshift|openshift-.*'}
        ```


### Common Fields

* `report_period_start`: The start timestamp of the reporting period.
* `report_period_end`: The end timestamp of the reporting period.
* `interval_start`: The start timestamp of the reporting interval.
* `interval_end`: The end timestamp of the reporting interval.

These common fields are included in all the reports and provide temporal information about the reporting period and interval.


## Cost Management Reports:


### 1. Node Metrics

Fields for metrics related to nodes:

* `node`: The name of the node.
* `node_labels`: The labels associated with the node.


### 2. Namespace Metrics

Fields for metrics related to namespaces:

* `namespace`: The namespace.
* `namespace_labels`: The labels associated with the namespace.


### 3. Pod Metrics

Fields for metrics related to pods:

* `node`: The name of the node.
* `namespace`: The namespace of the pod.
* `pod`: The name of the pod.
* `pod_usage_cpu_core_seconds`: The CPU core seconds used by the pod.
* `pod_request_cpu_core_seconds`: The CPU core seconds requested by the pod.
* `pod_limit_cpu_core_seconds`: The CPU core seconds limited for the pod.
* `pod_usage_memory_byte_seconds`: The memory byte seconds used by the pod.
* `pod_request_memory_byte_seconds`: The memory byte seconds requested by the pod.
* `pod_limit_memory_byte_seconds`: The memory byte seconds limited for the pod.
* `node_capacity_cpu_cores`: The CPU cores capacity of the node.
* `node_capacity_cpu_core_seconds`: The CPU core seconds capacity of the node.
* `node_capacity_memory_bytes`: The memory bytes capacity of the node.
* `node_capacity_memory_byte_seconds`: The memory byte seconds capacity of the node.
* `node_role`: The role of the node.
* `resource_id`: The unique identifier of the resource.
* `pod_labels`: The labels associated with the pod.


### 4. Virtual Machine Metrics

Fields for metrics related to running virtual machines (VMs) and their associated resources:

* `node`: The name of the node where the virtual machine instance (VMI) is currently running.
* `namespace`: The namespace where the virtual machine (VM) is defined.
* `resource_id`: The unique identifier of the VM on the node. This is derived from the `provider_id` of the node, specifically the segment after the last `/` character. For example, if the `provider_id` is `aws:///us-east-1a/i-0abcdef1234567890`, the `resource_id` is `i-0abcdef1234567890`.
* `vm_name`: The name of the VM.
* `vm_instance_type`: The instance type associated with the VM, if defined.
* `vm_os`: The operating system reported by the VM's info.
* `vm_guest_os_arch`: The guest operating system architecture reported by the VM. For example x86_64.
* `vm_guest_os_name`: The guest operating system name reported by the VM. For example RHEL or Fedora.
* `vm_guest_os_version`: The guest operating system version number reported by the VM. For example 8.6.
* `vm_uptime_total_seconds`: The total uptime of the VMI in seconds since it started.
* `vm_cpu_limit_cores`: The CPU core limit configured, representing the maximum number of cores the VM can use.
* `vm_cpu_limit_core_seconds`: The total CPU core seconds limited for the VM over the reporting period.
* `vm_cpu_request_cores`: The CPU core request configured for the VM, representing the guaranteed number of cores the VM will receive.
* `vm_cpu_request_core_seconds`: The total CPU core seconds requested for the VM over the reporting period.
* `vm_cpu_request_sockets`: The number of CPU sockets requested for the VM.
* `vm_cpu_request_socket_seconds`: The total CPU socket seconds requested for the VM over the reporting period.
* `vm_cpu_request_threads`: The number of CPU threads requested per core for the VM.
* `vm_cpu_request_thread_seconds`: The total CPU thread seconds requested for the VM over the reporting period.
* `vm_cpu_usage_total_seconds`: The total CPU usage of the VM in seconds over the reporting period.
* `vm_memory_limit_bytes`: The memory limit configured for the VM in bytes, representing the maximum memory the VM can use.
* `vm_memory_limit_byte_seconds`: The total memory byte seconds limited for the VM over the reporting period.
* `vm_memory_request_bytes`: The memory request configured for the VM in bytes, representing the guaranteed memory the VM will receive.
* `vm_memory_request_byte_seconds`: The total memory byte seconds requested for the VM over the reporting period.
* `vm_memory_usage_byte_seconds`: The total memory usage of the VM in byte seconds over the reporting period.
* `vm_device`: The name of the virtual device attached to the VM, typically referring to a disk.
* `vm_volume_mode`: The volume mode of the attached disk. For example Block or Filesystem.
* `vm_persistentvolumeclaim_name`: The name of the `PersistentVolumeClaim` backing the VM's disk.
* `vm_disk_allocated_size_byte_seconds`: The total allocated disk size for the VM's storage in byte seconds over the reporting period.
* `vm_labels`: A JSON string representing key-value pairs of labels applied to the VM.


### 5. Persistent Volume Metrics

Fields for metrics related to Persistent Volumes (PVs):

* `namespace`: The namespace associated with the persistent volume claim (PVC).
* `pod`: The name of the pod associated with the persistent volume claim.
* `persistentvolumeclaim`: The name of the persistent volume claim.
* `persistentvolume`: The name of the persistent volume.
* `storageclass`: The storage class of the persistent volume claim.
* `persistentvolumeclaim_capacity_bytes`: The capacity of the persistent volume claim in bytes.
* `persistentvolumeclaim_capacity_byte_seconds`: The capacity of the persistent volume claim in byte seconds.
* `volume_request_storage_byte_seconds`: The storage byte seconds requested by the volume.
* `persistentvolumeclaim_usage_byte_seconds`: The usage byte seconds of the persistent volume claim.
* `persistentvolume_labels`: The labels associated with the persistent volume.
* `persistentvolumeclaim_labels`: The labels associated with the persistent volume claim.



## Resource Optimization (ROS) Reports:

### 1. Container Metrics

Fields for metrics related to containers:

* `container_name`: The name of the container.
* `pod`: The name of the pod that associated with the container.
* `owner_name`: The name of the owner entity that is associated with the container. For example Deployment or StatefulSet.
* `owner_kind`: The kind of the owner entity that is associated with the container. For example Deployment or StatefulSet.
* `workload`: The workload associated with the container.
* `workload_type`: The type of the workload.
* `namespace`: The namespace of the container.
* `image_name`: The name of the container's image.
* `node`: The node on which the container is running.
* `resource_id`: The unique identifier of the resource.
* `cpu_request_container_avg`: The average CPU request for the container.
* `cpu_request_container_sum`: The total CPU request for the container.
* `cpu_limit_container_avg`: The average CPU limit for the container.
* `cpu_limit_container_sum`: The total CPU limit for the container.
* `cpu_usage_container_avg`: The average CPU usage for the container.
* `cpu_usage_container_min`: The minimum CPU usage for the container.
* `cpu_usage_container_max`: The maximum CPU usage for the container.
* `cpu_usage_container_sum`: The total CPU usage for the container.
* `cpu_throttle_container_avg`: The average CPU throttle for the container.
* `cpu_throttle_container_max`: The maximum CPU throttle for the container.
* `cpu_throttle_container_sum`: The total CPU throttle for the container.
* `memory_request_container_avg`: The average memory request for the container.
* `memory_request_container_sum`: The total memory request for the container.
* `memory_limit_container_avg`: The average memory limit for the container.
* `memory_limit_container_sum`: The total memory limit for the container.
* `memory_usage_container_avg`: The average memory usage for the container.
* `memory_usage_container_min`: The minimum memory usage for the container.
* `memory_usage_container_max`: The maximum memory usage for the container.
* `memory_usage_container_sum`: The total memory usage for the container.
* `memory_rss_usage_container_avg`: The average RSS memory usage for the container.
* `memory_rss_usage_container_min`: The minimum RSS memory usage for the container.
* `memory_rss_usage_container_max`: The maximum RSS memory usage for the container.
* `memory_rss_usage_container_sum`: The total RSS memory usage for the container.


### 2. Namespace Metrics

Fields for metrics related to namespaces:

* `namespace`: The name of the namespace.
* `cpu_request_namespace_sum`: The total CPU cores requested by all containers in the namespace, derived from resource quotas.
* `cpu_limit_namespace_sum`: The total CPU core limits configured for all containers in the namespace, derived from resource quotas.
* `cpu_usage_namespace_avg`: The average CPU usage rate across all containers in the namespace over a 15 minute window.
* `cpu_usage_namespace_max`: The maximum CPU usage rate observed across all containers in the namespace over a 15 minute window.
* `cpu_usage_namespace_min`: The minimum CPU usage rate observed across all containers in the namespace over a 15 minute window.
* `cpu_throttle_namespace_avg`: The average CPU throttling rate for all containers in the namespace over a 15 minute window, indicating how often containers hit their CPU limits.
* `cpu_throttle_namespace_max`: The maximum CPU throttling rate observed for all containers in the namespace over a 15 minute window.
* `cpu_throttle_namespace_min`: The minimum CPU throttling rate observed for all containers in the namespace over a 15 minute window.
* `memory_request_namespace_sum`: The total memory requested by all containers in the namespace, derived from resource quotas.
* `memory_limit_namespace_sum`: The total memory limits configured for all containers in the namespace, derived from resource quotas.
* `memory_usage_namespace_avg`: The average working set memory usage across all containers in the namespace over a 15 minute window.
* `memory_usage_namespace_max`: The maximum working set memory usage observed across all containers in the namespace over a 15 minute window.
* `memory_usage_namespace_min`: The minimum working set memory usage observed across all containers in the namespace over a 15 minute window.
* `memory_rss_usage_namespace_avg`: The average RSS (Resident Set Size) memory usage across all containers in the namespace over a 15 minute window.
* `memory_rss_usage_namespace_max`: The maximum RSS memory usage observed across all containers in the namespace over a 15 minute window.
* `memory_rss_usage_namespace_min`: The minimum RSS memory usage observed across all containers in the namespace over a 15 minute window.
* `namespace_running_pods_max`: The maximum number of pods in a running state observed in the namespace over a 15 minute window.
* `namespace_running_pods_avg`: The average number of pods in a running state in the namespace over a 15 minute window.
* `namespace_total_pods_max`: The maximum total number of pods (all phases) observed in the namespace over a 15 minute window.
* `namespace_total_pods_avg`: The average total number of pods (all phases) in the namespace over a 15 minute window.
