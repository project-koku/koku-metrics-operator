# Local Development

This guide covers the full developer journey: environment setup, running the operator, making changes, and submitting a pull request. For general contribution policies, see [CONTRIBUTING](../CONTRIBUTING).

## Pre-reqs

* Access to a supported version of an Openshift cluster
  * [Red Hat OpenShift Local](https://crc.dev/docs/installing/) can be used if the monitoring stack is enabled.
  * A cluster can also be provisioned [here](https://demo.redhat.com/catalog)
* A clone of [koku-metrics-operator](https://github.com/project-koku/koku-metrics-operator)
* [Go 1.13 or greater](https://golang.org/doc/install)
* [Openshift-CLI](https://docs.redhat.com/en/documentation/openshift_container_platform/latest/html/cli_tools/openshift-cli-oc#cli-about-cli_cli-developer-commands) (preferably a version that matches your Openshift cluster version)
* [kubebuilder](https://book.kubebuilder.io/quick-start.html#installation)
* [kubectl](https://kubernetes.io/docs/tasks/tools/install-kubectl/)
* [kustomize](https://kubectl.docs.kubernetes.io/installation/kustomize/) (before installing this separately, check that it was not already installed along with kubectl)
* [Docker Desktop](https://www.docker.com/products/docker-desktop)
* [quay.io](quay.io) account

## Getting Started

1. **Fork and clone:**

```bash
git clone https://github.com/<your-username>/koku-metrics-operator.git
cd koku-metrics-operator
```

2. **Prevent accidental commits to kustomization.yaml:**

```bash
git update-index --assume-unchanged config/manager/kustomization.yaml
```

3. **Create a branch:**

```bash
git checkout -b feature/your-feature-name
```

Branch naming conventions: `feature/`, `fix/`, `docs/`, `refactor/`

## Running the operator locally

1. Log into your OCP cluster from a terminal, create an `koku-metrics-operator` namespace, and switch to the new namespace:

    ```
    $ oc login --token=<token> --server=<server>
    $ oc new-project koku-metrics-operator
    ```

2. Build the manager binary:

    ```
    $ make build
    ```

3. Register the CRD with the Kubernetes apiserver:

    ```
    $ make install
    ```

4. Deploy the ServiceAccount:

    ```
    $ oc apply -f testing/sa.yaml
    ```

5. Grant monitoring read access to the ServiceAccount:

    The local operator uses the `koku-metrics-controller-manager` ServiceAccount token to query the Prometheus/Thanos endpoint.
    Grant the ServiceAccount `cluster-monitoring-view` permissions:

    ```bash
    oc adm policy add-cluster-role-to-user \
      cluster-monitoring-view \
      system:serviceaccount:koku-metrics-operator:koku-metrics-controller-manager
    ```

    If this permission is missing, reconciliation can fail with:
    `prometheus test query failed: client_error: client error: 403`

6.  Retrieve ServiceAccount Token and CA Certificate:

    The operator's local environment needs a serviceAccount token and the cluster's service CA certificate to authenticate with the Kubernetes API. The `get-token-and-cert` Make command handles this retrieval:

    ```bash
    make get-token-and-cert
    ```

    This generates a new token for the `koku-metrics-controller-manager` ServiceAccount and retrieves the cluster's Service CA certificate from the `kube-root-ca.crt` ConfigMap.

    These files (`token` and `service-ca.crt`) will be placed in the `testing` directory but you can specify a different output location by setting the `SECRET_ABSPATH` environment variable:

    ```bash
    SECRET_ABSPATH=/absolute/path/to/local/secrets make get-token-and-cert
    ```

7. Deploy the operator

    ```
    make run ENABLE_WEBHOOKS=false SECRET_ABSPATH=/absolute/path/to/local/secrets
    ```

    At this point, you will see the operator spin up in your terminal. After a few seconds, you should see something similar to the following output:
    ```
    2020-10-21T09:31:37.195-0400    INFO    controller-runtime.controller   Starting workers        {"controller": "kokumetricsconfig", "worker count": 1}
    ```
    The operator is running but is not doing any work. We need to create a CR.

8. Deploy a CR. For local development, uses default token auth. However service-account authentication can be used with the following, which creates the appropriate authentication spec within the CR. This is the `client_id` and `client_secret` for your Red Hat Hybrid Cloud Console:

    ```
    $ make deploy-local-cr AUTH=service-account CLIENT_ID=<client_id> CLIENT_SECRET=<client_secret>
    ```
    This command copies `testing/costmanagementmetricsconfig-template.yaml` to `testing/costmanagement-metrics-cfg_v1beta1_costmanagementmetricsconfig.yaml`, adds an external prometheus route, disables TLS verification for the prometheus route, and (when `AUTH=service-account`) adds the authentication spec and applies `testing/authentication_secret.yaml`. The command then applies `testing/costmanagement-metrics-cfg_v1beta1_costmanagementmetricsconfig.yaml` to the cluster.

    After this CR has been created in the cluster, reconciliation will begin.

    Running `make deploy-local-cr` as-is will create the external prometheus route, disable TLS verification for prometheus, and use token authentication for console.redhat.com.

9. To continue development, make code changes. To apply those changes, stop the operator, and redeploy it. If changes are made to the api, the CRD needs to be re-registered, and the operator re-deployed.

## Development Workflow

1. Read existing code in the area you're modifying
2. Write failing tests first (TDD)
3. Implement the feature or fix
4. Run checks:

```bash
make test              # run all tests
make fmt               # format code
make lint              # run linters
make verify-manifests  # if CRD or RBAC changed
```

5. Update documentation if the change is user-facing

For code conventions (Go standards, import ordering, Kubernetes operator patterns, testing patterns), see [AGENTS.md](../AGENTS.md).

## CRD Changes

**CRD changes can break existing clusters.** If modifying the schema:

1. Update `api/v1beta1/metricsconfig_types.go`
2. Add kubebuilder markers for validation
3. Run `make manifests` to regenerate
4. Test backward compatibility
5. Document in your PR description

**Never** remove fields, change field types, or make optional fields required.

## Submitting a Pull Request

### PR Title

Use conventional commit style — keep under 70 characters:

- `feat: add GPU metrics collection for NVIDIA cards`
- `fix: handle nil pointer in reconciliation loop`
- `docs: update local development prerequisites`

### PR Description

```markdown
## Description
Brief description of changes.

## Motivation
Why is this change needed?

## Testing
- [ ] Unit tests added/updated
- [ ] Integration tests pass
- [ ] Manually tested on OpenShift cluster (if applicable)
- [ ] Verified backward compatibility

## Checklist
- [ ] `make fmt` passes
- [ ] `make lint` passes
- [ ] `make test` passes
- [ ] Documentation updated (if user-facing)
- [ ] No secrets committed
```

### Do NOT

- Modify `vendor/` manually (use `make vendor`)
- Skip pre-commit hooks
- Commit secrets or credentials
- Change downstream files when on upstream (or vice versa)

### Review Process

1. Automated CI runs (tests, linting, build)
2. Code review from maintainers
3. Address review comments — push new commits (don't force-push unless requested)
4. Approval and merge

## Troubleshooting

| Problem | Solution |
|---------|----------|
| `403 from Prometheus` | Grant `cluster-monitoring-view` to the ServiceAccount (see step 5 above) |
| `CRD not found` | Run `make install` |
| Tests fail with nil pointer | Check mock initialization in `BeforeEach` |
| Build fails with missing package | Run `make vendor` |

**Useful commands:**

```bash
# Check operator logs
oc logs -n koku-metrics-operator deployment/koku-metrics-operator-controller-manager -f

# Get CR status
oc get costmanagementmetricsconfig -o yaml

# Check RBAC
oc adm policy who-can get prometheuses.monitoring.coreos.com

# Run a specific test
go test ./internal/collector -run TestGenerateReports
```
