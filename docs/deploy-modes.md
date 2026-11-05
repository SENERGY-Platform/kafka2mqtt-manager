# Deploy modes

kafka2mqtt-manager runs every instance as a container of `TRANSFER_IMAGE`. Where
and how that container runs is chosen with `DEPLOY_MODE`.

## Scope

Describes `DEPLOY_MODE` `kubernetes` as of v1.1.0, plus the one difference to
`rancher2` that callers notice. The `docker` and `rancher1` drivers are not
covered.

The update and delete behavior below belongs to the `kubernetes` driver only.
`rancher2` deletes and recreates the workload on every update, which restarts the
instance and pulls the image again even when nothing changed; that cannot be
assumed for `kubernetes`, and the reverse holds as well.

## Modes

| `DEPLOY_MODE` | Package | Instances run as |
|---|---|---|
| `docker` | `pkg/deploy/dockerClient` | containers on the local Docker daemon |
| `rancher1` | `pkg/deploy/rancher-api` | Rancher 1 services |
| `rancher2` | `pkg/deploy/rancher2-api` | workloads created through the Rancher v3 API, configured with `RANCHER_URL`, `RANCHER_ACCESS_KEY`, `RANCHER_SECRET_KEY`, `RANCHER_PROJECT_ID`, `RANCHER_NAMESPACE_ID` |
| `kubernetes` | `pkg/deploy/kubernetes-api` | Deployments created through the Kubernetes API |

Any other value stops the start with `unknown deploy_mode`.

## kubernetes

### Requirements

- The manager runs inside the cluster and authenticates as its pod's service
  account. There is no kubeconfig fallback, so the mode cannot be run from a
  workstation.
- `KUBERNETES_NAMESPACE` names the namespace for the instances, default
  `kafka2mqtt-instances`. The manager itself may run in another namespace.
- The service account needs a Role in that namespace with `get`, `list`,
  `create`, `update` and `delete` on `deployments` in the `apps` group, bound to
  it with a RoleBinding. `list` is used only by the start check: the manager lists
  the Deployments once and stops with
  `unable to list deployments in namespace <namespace>: <error>` if it may not.

### What an instance looks like

One Deployment per instance, named `k2m-<uuid>` after the instance id, with one
replica and these properties:

| Field | Value |
|---|---|
| labels on Deployment and pod | `kafka2mqtt: <name>`, `user: <user id>` |
| selector | `kafka2mqtt: <name>` |
| strategy | `Recreate` |
| node selector | `role: worker` |
| service account token | not mounted |
| image pull policy | `Always` |
| environment | the instance settings, sorted by name |

The user id becomes a label value, so it has to be one: at most 63 characters of
letters, digits, `-`, `_` and `.`. The API server rejects anything else and the
create request fails with status 500. `rancher2` has the same limit.

### Behavior

- **Update** replaces the pod template of the existing Deployment and retries
  when the Deployment changed between read and write. `Recreate` stops the old
  pod before the new one starts, so two pods never forward the same topic at
  once. With unchanged settings the template stays the same and the pod is not
  restarted, which also means a new image under the same tag is not pulled. If
  the Deployment is missing, it is created.
- **Delete** treats a missing Deployment as deleted. The controller removes the
  instance from its database only after the delete succeeded, so an error here
  would leave the instance undeletable.
- **Exists** answers `false` for a missing Deployment; `STARTUP_ENSURE_DEPLOYED`
  recreates those instances on start.
- A container that is not restarted on exit is not supported:
  `deploy_mode kubernetes runs instances only as restarting deployments`. The
  controller always asks for restarting containers.
