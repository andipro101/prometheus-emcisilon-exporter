Prometheus-emcisilon-exporter
======

### Introduction

`prometheus-emcisilon-exporter` is a Golang Prometheus exporter for EMC Isilon (PowerScale) clusters. It uses the OneFS REST API to expose metrics and configuration items.

Supports **multiple clusters** from a single instance using the multi-target scraping pattern (like `blackbox_exporter`). Prometheus passes `?target=<host>` and the exporter returns metrics for that cluster.

### Usage

#### Building

```sh
go build ./...
```

#### Running — single cluster (CLI flags)

```sh
export ISILON_CLUSTER_PASSWORD=secret
./prometheus-emcisilon-exporter \
  --isilon.cluster.host=192.168.1.100 \
  --isilon.cluster.port=8080 \
  --isilon.cluster.username=admin \
  --isilon.cluster.password.env=ISILON_CLUSTER_PASSWORD \
  --isilon.cluster.site=dc1
```

Scrape metrics at `http://localhost:9300/metrics`.

#### Running — multiple clusters (config file)

Create a `clusters.yaml`:

```yaml
clusters:
  - host: 192.168.1.100
    port: "8080"
    username: admin
    password_env: ISILON_CLUSTER1_PASSWORD
    site: dc1
  - host: 10.0.0.50
    port: "8080"
    username: admin
    password_env: ISILON_CLUSTER2_PASSWORD
    site: dc2
```

```sh
export ISILON_CLUSTER1_PASSWORD=secret1
export ISILON_CLUSTER2_PASSWORD=secret2
./prometheus-emcisilon-exporter --config.file=clusters.yaml
```

Scrape per cluster:
```
http://localhost:9300/metrics?target=192.168.1.100
http://localhost:9300/metrics?target=10.0.0.50
```

#### Prometheus scrape config (multi-cluster)

```yaml
scrape_configs:
  - job_name: isilon
    metrics_path: /metrics
    static_configs:
      - targets:
          - 192.168.1.100
          - 10.0.0.50
    relabel_configs:
      - source_labels: [__address__]
        target_label: __param_target
      - target_label: __address__
        replacement: localhost:9300
      - source_labels: [__param_target]
        target_label: instance
```

---

### Adding a new cluster

Three places need updating — no restart of the exporter pod is required for the ConfigMap change (Kubernetes remounts it automatically), but the pod does need a restart to re-read the file.

**1. `isilon-exporter-configmap.yaml`** — add an entry under `clusters:`

```yaml
clusters:
  - host: 10.142.94.20
    port: "8080"
    username: admin
    password_env: ISILON_CLUSTER_PASSWORD
    site: sofia
  - host: 10.142.94.21        # new cluster
    port: "8080"
    username: admin
    password_env: ISILON_CLUSTER2_PASSWORD
    site: london
```

**2. `isilon-exporter-secret.yaml`** — add the password env var for the new cluster

```yaml
stringData:
  ISILON_CLUSTER_PASSWORD: secret1
  ISILON_CLUSTER2_PASSWORD: secret2   # new
```

**3. `isilon-exporter-servicemonitor.yaml`** — add a new `endpoints` entry

```yaml
endpoints:
  - interval: 30s
    path: /metrics
    port: metrics
    params:
      target: [10.142.94.20]
    relabelings:
    - sourceLabels: [__param_target]
      targetLabel: instance
  - interval: 30s            # new
    path: /metrics
    port: metrics
    params:
      target: [10.142.94.21]
    relabelings:
    - sourceLabels: [__param_target]
      targetLabel: instance
```

Apply in order:
```sh
kubectl apply -f isilon-exporter-secret.yaml
kubectl apply -f isilon-exporter-configmap.yaml
kubectl rollout restart deployment/isilon-exporter -n monitoring
kubectl apply -f isilon-exporter-servicemonitor.yaml
```

---

### Site label grouping

The `site` field adds a `site` label to **every metric** emitted for that cluster. It is purely a Prometheus label — no logic is tied to it.

**Without site:**
```
isilon_ifs_bytes_used{cluster="cluster1"} 1.2e+12
```

**With `site: sofia`:**
```
isilon_ifs_bytes_used{cluster="cluster1", site="sofia"} 1.2e+12
```

This lets you filter and aggregate across locations in PromQL:

```promql
# Capacity used across all clusters in sofia
sum(isilon_ifs_bytes_used{site="sofia"})

# Compare used capacity by site
sum by (site) (isilon_ifs_bytes_used)
```

And in Grafana you can use `site` as a variable to filter dashboards by location.

> **Note:** The `site` label is a **constant label** — it is baked into the descriptor at scrape time. If you change a cluster's `site` value in the ConfigMap, the old label value disappears and a new time series starts. Prometheus will show both the old and new series until the old one expires.

---

### Configuration

#### Config file fields (`clusters.yaml`)

| Field | Description | Default | Required |
|---|---|---|---|
| `host` | Hostname or IP address of the Isilon cluster | | Yes |
| `port` | API port | `8080` | No |
| `username` | API username | | Yes |
| `password_env` | Environment variable containing the password | `ISILON_CLUSTER_PASSWORD` | No |
| `site` | Data centre site label added to all metrics | | No |
| `quota_only` | Only collect quota metrics | `false` | No |
| `quota_retry` | Number of retries for quota collection | `3` | No |

#### CLI flags

| Flag | Description | Default |
|---|---|---|
| `--config.file` | Path to YAML config file (multi-cluster mode) | |
| `--isilon.cluster.host` | Hostname or IP of the cluster (single-cluster mode) | |
| `--isilon.cluster.port` | API port (single-cluster mode) | `8080` |
| `--isilon.cluster.username` | Username (single-cluster mode) | |
| `--isilon.cluster.password.env` | Password env var (single-cluster mode) | `ISILON_CLUSTER_PASSWORD` |
| `--isilon.cluster.site` | Site label (single-cluster mode) | |
| `--quota-only` | Collect quota metrics only (single-cluster mode) | `false` |
| `--web.listen-address` | Address to expose metrics on | `:9300` |
| `--web.telemtry-path` | HTTP path for metrics | `/metrics` |
| `--log.level` | Log level | `info` |
| `--log.format` | Log format | `logger:stderr` |

---

### Collectors

All collectors are enabled by default unless noted. Toggle with `--collector.<name>` / `--no-collector.<name>`.

| Flag | Collector | Description | Default |
|---|---|---|---|
| `--collector.capacity` | capacity | /ifs capacity (bytes avail/free/used) | enabled |
| `--collector.cluster_health` | cluster_health | Cluster health state | enabled |
| `--collector.cluster_protocol` | cluster_protocol | Protocol statistics at the cluster level | enabled |
| `--collector.cpu` | cpu | Per-node CPU statistics | enabled |
| `--collector.disk` | disk | Per-node disk statistics | enabled |
| `--collector.memory` | memory | Per-node memory statistics | enabled |
| `--collector.network` | network | Per-node network statistics | enabled |
| `--collector.nfs_exports` | nfs_exports | Total NFS export count | enabled |
| `--collector.node_health` | node_health | Per-node health state | enabled |
| `--collector.node_info` | node_info | Node hardware info and OneFS version | enabled |
| `--collector.node_partition` | node_partition | Node partition usage (/, /var, /var/crash, etc.) | enabled |
| `--collector.node_protocol` | node_protocol | Per-node protocol statistics | enabled |
| `--collector.quota` | quota | Per-quota thresholds and usage | disabled |
| `--collector.quota_summary` | quota_summary | Summary counts of all quota types | enabled |
| `--collector.smb_shares` | smb_shares | Total SMB share count | enabled |
| `--collector.snapshots` | snapshots | Snapshot counts and sizes by age | enabled |
| `--collector.statfs` | statfs | Filesystem block/inode statistics | enabled |
| `--collector.storagepools` | storagepools | Storage pool capacity and VHS info | enabled |
| `--collector.sync_iq` | sync_iq | SyncIQ policy state and last-run info | enabled |

---

### Provided Metrics

```
# HELP isilon_cluster_health Current health of the cluster.
# HELP isilon_cluster_onefs_version OneFS version (value always 1; version in label).
# HELP isilon_ifs_bytes_avail Current /ifs capacity available in bytes.
# HELP isilon_ifs_bytes_free Current /ifs capacity free in bytes.
# HELP isilon_ifs_bytes_total Current /ifs capacity total in bytes.
# HELP isilon_ifs_bytes_used Current /ifs capacity used in bytes.
# HELP isilon_ifs_percent_avail /ifs capacity available (0.0-1.0).
# HELP isilon_ifs_percent_free /ifs capacity free (0.0-1.0).
# HELP isilon_ifs_percent_used /ifs capacity used (0.0-1.0).
# HELP isilon_nfs_export_total Total number of NFS exports on a cluster.
# HELP isilon_node_boottime Unix timestamp of when a node booted.
# HELP isilon_node_cpu_sys_avg CPU busy percentage for sys mode (0.0-1.0).
# HELP isilon_node_cpu_user_avg CPU busy percentage for user mode (0.0-1.0).
# HELP isilon_node_disk_busy_all Disk busy percentage (0.0-1.0).
# HELP isilon_node_disk_count Number of disks per node.
# HELP isilon_node_disk_iosched_queued_all IO scheduler queue depth.
# HELP isilon_node_disk_unhealthy_count Number of unhealthy disks per node.
# HELP isilon_node_disk_xfers_in_rate_all Disk ingest transfer rate.
# HELP isilon_node_disk_xfers_out_rate_all Disk egress transfer rate.
# HELP isilon_node_health Current health of a node.
# HELP isilon_node_load_1min 1-minute node load average.
# HELP isilon_node_load_5min 5-minute node load average.
# HELP isilon_node_load_15min 15-minute node load average.
# HELP isilon_node_memory_cache RAM used for cache in bytes.
# HELP isilon_node_memory_free RAM free in bytes.
# HELP isilon_node_memory_used RAM in use in bytes.
# HELP isilon_node_net_ext_bytes_in_rate External network bytes-in rate.
# HELP isilon_node_net_ext_bytes_out_rate External network bytes-out rate.
# HELP isilon_node_partition_count Total partitions on a node.
# HELP isilon_node_partition_filenodes_free Free filenodes on a partition.
# HELP isilon_node_partition_filenodes_free_percent Free filenodes percentage.
# HELP isilon_node_partition_filenodes_total Total filenodes on a partition.
# HELP isilon_node_partition_used_space_percentage Space used percentage on a partition.
# HELP isilon_node_status_battery Battery status.
# HELP isilon_node_status_power_supply Power supply status.
# HELP isilon_node_uptime Node uptime in seconds.
# HELP isilon_quota_container 1 if container quota, 0 if not.
# HELP isilon_quota_enforced 1 if enforced, 2 if advisory.
# HELP isilon_quota_include_snapshots 1 if snapshots are included in usage.
# HELP isilon_quota_threshold_advisory Advisory threshold in bytes.
# HELP isilon_quota_threshold_advisory_exceeded 1 if advisory threshold exceeded.
# HELP isilon_quota_threshold_hard Hard threshold in bytes.
# HELP isilon_quota_threshold_hard_exceeded 1 if hard threshold exceeded.
# HELP isilon_quota_threshold_soft Soft threshold in bytes.
# HELP isilon_quota_threshold_soft_exceeded 1 if soft threshold exceeded.
# HELP isilon_quota_threshold_soft_grace Soft grace period in seconds.
# HELP isilon_quota_usage_inodes Inodes used by governed data.
# HELP isilon_quota_usage_logical Apparent bytes used by governed data.
# HELP isilon_quota_usage_physical Physical bytes used by governed data.
# HELP isilon_quota_summary_default_group_quotas_count Default group quota count.
# HELP isilon_quota_summary_default_user_quotas_count Default user quota count.
# HELP isilon_quota_summary_directory_quotas_count Directory quota count.
# HELP isilon_quota_summary_group_quotas_count Group quota count.
# HELP isilon_quota_summary_linked_quotas_count Linked quota count.
# HELP isilon_quota_summary_quotas_user User quota count.
# HELP isilon_quota_summary_total_quotas_count Total quota count.
# HELP isilon_smb_share_total Total number of SMB shares on a cluster.
# HELP isilon_snapshots_active_count Active snapshot count.
# HELP isilon_snapshots_active_size Active snapshot size in bytes.
# HELP isilon_snapshots_deleting_count Snapshots being deleted.
# HELP isilon_snapshots_deleting_size Size of snapshots being deleted in bytes.
# HELP isilon_snapshots_total_count Total snapshot count.
# HELP isilon_snapshots_total_size Total snapshot size in bytes.
# HELP isilon_snapshots_7_day_count Snapshots older than 7 days.
# HELP isilon_snapshots_15_day_count Snapshots older than 15 days.
# HELP isilon_snapshots_30_day_count Snapshots older than 30 days.
# HELP isilon_snapshots_60_day_count Snapshots older than 60 days.
# HELP isilon_snapshots_90_day_count Snapshots older than 90 days.
# HELP isilon_statfs_file_block_avail Available blocks in filesystem.
# HELP isilon_statfs_file_block_free Free blocks in filesystem.
# HELP isilon_statfs_file_block_size Filesystem fragment size.
# HELP isilon_statfs_file_block_total Total data blocks in filesystem.
# HELP isilon_statfs_file_io_size Optimal transfer block size.
# HELP isilon_statfs_file_name_max Maximum filename length.
# HELP isilon_statfs_file_node_free Free file nodes in filesystem.
# HELP isilon_statfs_file_node_free_percent Percentage of free file nodes.
# HELP isilon_statfs_file_node_total Total file nodes in filesystem.
# HELP isilon_storage_pool_balanced 0 = balanced, 1 = not balanced.
# HELP isilon_storage_pool_bytes_avail Bytes available on storage pool.
# HELP isilon_storage_pool_bytes_avail_ssd Bytes available on SSD for pool.
# HELP isilon_storage_pool_bytes_free Bytes free on storage pool.
# HELP isilon_storage_pool_bytes_free_ssd Bytes free on SSD for pool.
# HELP isilon_storage_pool_bytes_total Total bytes on storage pool.
# HELP isilon_storage_pool_bytes_total_ssd Total SSD bytes for pool.
# HELP isilon_storage_pool_bytes_virtual_hot_spare VHS bytes for pool.
# HELP isilon_storage_pool_manual 0 = auto-managed, 1 = manually managed.
# HELP isilon_storage_pool_total Total storage pool count.
# HELP isilon_sync_policies_total_count Total SyncIQ policy count.
# HELP isilon_sync_policy_enabled 1 = enabled, 0 = disabled.
# HELP isilon_sync_policy_last_start Epoch timestamp of last sync start.
# HELP isilon_sync_policy_last_success Epoch timestamp of last successful sync.
# HELP isilon_sync_policy_priority Policy priority.
# HELP isilon_sync_policy_state 0 = finished, 1 = other.
# HELP isilon_sync_policy_workers_per_node Worker threads per node.
# HELP isilon_scrape_collector_duration_seconds Duration of each collector scrape.
# HELP isilon_scrape_collector_success 1 = collector succeeded, 0 = failed.
# HELP isilon_exporter_duration_seconds Duration of the full exporter run.
# HELP isilon_stats_engine_call_duration_seconds Duration of a stats engine API call.
# HELP isilon_stats_engine_call_success 0 = success, 1 = failure.
```

---

### Licensing

This project is licensed under the MIT license. See [LICENSE](LICENSE) for more information.
