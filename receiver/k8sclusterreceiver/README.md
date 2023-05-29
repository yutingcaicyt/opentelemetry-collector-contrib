# Kubernetes Cluster Receiver

| Status                   |           |
| ------------------------ |-----------|
| Stability                | [beta]    |
| Supported pipeline types | metrics   |
| Distributions            | [contrib] |

The Kubernetes Cluster receiver collects cluster-level metrics from the Kubernetes
API server. It uses the K8s API to listen for updates. A single instance of this
receiver can be used to monitor a cluster.

Currently this receiver supports authentication via service accounts only. See [example](#example)
for more information.

## Configuration

The following settings are required:

- `auth_type` (default = `serviceAccount`): Determines how to authenticate to
the K8s API server. This can be one of `none` (for no auth), `serviceAccount`
(to use the standard service account token provided to the agent pod), or
`kubeConfig` to use credentials from `~/.kube/config`.

The following settings are optional:

- `collection_interval` (default = `10s`): This receiver continuously watches
for events using K8s API. However, the metrics collected are emitted only
once every collection interval. `collection_interval` will determine the
frequency at which metrics are emitted by this receiver.
- `node_conditions_to_report` (default = `[Ready]`): An array of node
conditions this receiver should report. See
[here](https://kubernetes.io/docs/concepts/architecture/nodes/#condition) for
list of node conditions. The receiver will emit one metric per entry in the
array.
- `distribution` (default = `kubernetes`): The Kubernetes distribution being used
by the cluster. Currently supported versions are `kubernetes` and `openshift`. Setting
the value to `openshift` enables OpenShift specific metrics in addition to standard
kubernetes ones.
- `allocatable_types_to_report` (default = `[]`): An array of allocatable resource types this receiver should report.
The following allocatable resource types are available.
  - cpu
  - memory
  - ephemeral-storage
  - storage

Example:

```yaml
  k8s_cluster:
    auth_type: kubeConfig
    node_conditions_to_report: [Ready, MemoryPressure]
    allocatable_types_to_report: [cpu, memory]
```

The full list of settings exposed for this receiver are documented [here](./config.go)
with detailed sample configurations [here](./testdata/config.yaml).

### node_conditions_to_report

For example, with the config below the receiver will emit two metrics
`k8s.node.condition_ready` and `k8s.node.condition_memory_pressure`, one
for each condition in the config. The value will be `1` if the `ConditionStatus` for the
corresponding `Condition` is `True`, `0` if it is `False` and -1 if it is `Unknown`.

```yaml
...
k8s_cluster:
  node_conditions_to_report:
    - Ready
    - MemoryPressure
...
```

### metadata_exporters

A list of metadata exporters to which metadata being collected by this receiver
should be synced. Exporters specified in this list are expected to implement the
following interface. If an exporter that does not implement the interface is listed,
startup will fail.

```yaml
type MetadataExporter interface {
  ConsumeMetadata(metadata []*MetadataUpdate) error
}

type MetadataUpdate struct {
  ResourceIDKey string
  ResourceID    ResourceID
  MetadataDelta
}

type MetadataDelta struct {
  MetadataToAdd    map[string]string
  MetadataToRemove map[string]string
  MetadataToUpdate map[string]string
}
```

See [here](internal/collection/metadata.go) for details about the above types.


## Example

Here is an example deployment of the collector that sets up this receiver along with
the logging exporter.

Follow the below sections to setup various Kubernetes resources required for the deployment.

### Configuration

Create a ConfigMap with the config for `otelcontribcol`:

```bash
cat <<EOF | kubectl apply -f -
apiVersion: v1
kind: ConfigMap
metadata:
  name: otelcontribcol
  labels:
    app: otelcontribcol
data:
  config.yaml: |
    receivers:
      k8s_cluster:
        collection_interval: 10s
    exporters:
      logging:
    service:
      pipelines:
        metrics:
          receivers: [k8s_cluster]
          exporters: [logging]
EOF
```

### Service Account

Create a service account that the collector should use.

```bash
<<EOF | kubectl apply -f -
apiVersion: v1
kind: ServiceAccount
metadata:
  labels:
    app: otelcontribcol
  name: otelcontribcol
EOF
```

### RBAC

Use the below commands to create a `ClusterRole` with required permissions and a 
`ClusterRoleBinding` to grant the role to the service account created above.

```bash
<<EOF | kubectl apply -f -
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRole
metadata:
  name: otelcontribcol
  labels:
    app: otelcontribcol
rules:
- apiGroups:
  - ""
  resources:
  - events
  - namespaces
  - namespaces/status
  - nodes
  - nodes/spec
  - pods
  - pods/status
  - replicationcontrollers
  - replicationcontrollers/status
  - resourcequotas
  - services
  verbs:
  - get
  - list
  - watch
- apiGroups:
  - apps
  resources:
  - daemonsets
  - deployments
  - replicasets
  - statefulsets
  verbs:
  - get
  - list
  - watch
- apiGroups:
  - extensions
  resources:
  - daemonsets
  - deployments
  - replicasets
  verbs:
  - get
  - list
  - watch
- apiGroups:
  - batch
  resources:
  - jobs
  - cronjobs
  verbs:
  - get
  - list
  - watch
- apiGroups:
    - autoscaling
  resources:
    - horizontalpodautoscalers
  verbs:
    - get
    - list
    - watch
EOF
```

```bash
<<EOF | kubectl apply -f -
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRoleBinding
metadata:
  name: otelcontribcol
  labels:
    app: otelcontribcol
roleRef:
  apiGroup: rbac.authorization.k8s.io
  kind: ClusterRole
  name: otelcontribcol
subjects:
- kind: ServiceAccount
  name: otelcontribcol
  namespace: default
EOF
```

### Deployment

Create a [Deployment](https://kubernetes.io/docs/concepts/workloads/controllers/deployment/) to deploy the collector.

```bash
<<EOF | kubectl apply -f -
apiVersion: apps/v1
kind: Deployment
metadata:
  name: otelcontribcol
  labels:
    app: otelcontribcol
spec:
  replicas: 1
  selector:
    matchLabels:
      app: otelcontribcol
  template:
    metadata:
      labels:
        app: otelcontribcol
    spec:
      serviceAccountName: otelcontribcol
      containers:
      - name: otelcontribcol
        image: otelcontribcol:latest # specify image
        args: ["--config", "/etc/config/config.yaml"]
        volumeMounts:
        - name: config
          mountPath: /etc/config
        imagePullPolicy: IfNotPresent
      volumes:
        - name: config
          configMap:
            name: otelcontribcol
EOF
```

### OpenShift

You can enable OpenShift support to collect OpenShift specific metrics in addition to the default
kubernetes ones. To do this, set the `distribution` key to `openshift`.

Example:

```yaml
  k8s_cluster:
    distribution: openshift
```

Add the following rules to your ClusterRole:

```yaml
- apigroups:
  - quota.openshift.io
  resources:
  - clusterresourcequotas
  verbs:
  - get
  - list
  - watch
```

[beta]: https://github.com/open-telemetry/opentelemetry-collector#beta
[contrib]: https://github.com/open-telemetry/opentelemetry-collector-releases/tree/main/distributions/otelcol-contrib
