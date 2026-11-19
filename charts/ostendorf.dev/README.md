# ostendorf-dev

Installs [ostendorf.dev](https://github.com/ostendorf-cloud/ostendorf.dev) on a Kubernetes
cluster.

## Prerequisites

- Kubernetes 1.27 or newer.
- Only if `httpRoute.enabled` is set: the Gateway API CRDs and a running `Gateway` with a
  listener that allows this release's namespace. See "External routing" below; this chart
  ships neither.

## Installing

```sh
helm install ostendorf-dev charts/ostendorf.dev
```

This chart is also published as an OCI artifact, at
`oci://ghcr.io/ostendorf-cloud/charts/ostendorf-dev`, versioned in lock-step with every release:

```sh
helm install ostendorf-dev oci://ghcr.io/ostendorf-cloud/charts/ostendorf-dev --version <version>
```

or pull it without installing:

```sh
helm pull oci://ghcr.io/ostendorf-cloud/charts/ostendorf-dev --version <version>
```

## Health, readiness and metrics

`/healthz`, `/readyz` and `/metrics` are served on a second, administrative listener
(`service.adminPort`, default `8081`), never on the public `service.port`:
`templates/deployment.yaml`'s `startupProbe`, `livenessProbe` and `readinessProbe` all point at
the `admin` container port, and `templates/ingress.yaml` and `templates/httproute.yaml` only ever
route to `service.port`. A `ServiceMonitor` or `PodMonitor` (not part of this chart) can scrape
`service.adminPort` directly, in-cluster.

## External routing

This chart reaches external traffic to its own `Service` two ways, chosen by
`ingress.enabled` and `httpRoute.enabled`. Both default to `false`: a default install creates
neither, reachable only from inside the cluster, until one is deliberately turned on. They are
mutually exclusive: turning both on fails the install or upgrade with a readable message
naming which one to turn off, enforced once in `templates/deployment.yaml`, the one template
that always renders regardless of which routing flag is set.

### Ingress

Set `ingress.enabled: true`. `ingress.hosts` names the host and path rules, and `ingress.tls`
carries the certificate configuration directly on the `Ingress` object: this chart owns that
configuration.

### Gateway API HTTPRoute

Set `httpRoute.enabled: true`, `httpRoute.parentRefs.name` and `httpRoute.parentRefs.namespace`
to the `Gateway` this release attaches to. The namespace is required, not optional the way
`ingress.className` is: a `Gateway` is almost always platform owned and lives in a different
namespace than this release, and the Gateway API's own default for an unset
`parentRefs.namespace` is the `HTTPRoute`'s own namespace, normally wrong here.

This chart never creates the `Gateway` and never ships the Gateway API CRDs (`Gateway`,
`HTTPRoute` and the rest): both are cluster-level prerequisites, installed and operated once
per cluster by the cluster operator, outside this chart. The `Gateway`'s own `allowedRoutes`
decides whether it actually accepts this `HTTPRoute`; a `httpRoute.enabled: true` release that
the `Gateway` does not allow installs without error but never actually receives traffic.

Unlike `Ingress`, an `HTTPRoute` carries no `tls` field of its own. TLS terminates on the
`Gateway`'s listener instead, which the cluster operator, not this chart, owns and configures;
`httpRoute.hostnames` only has to match a hostname the listener itself already serves for
traffic to reach this `HTTPRoute` at all.

Routing itself is fixed to this chart's own `Service`, on `service.port`, never
`service.adminPort`: no values-driven way to add a second backend or a per-path rule, the same
single-backend assumption `ingress.hosts` already makes. An `HTTPRoute` rule with no `matches`
defaults to a `/` prefix match per the Gateway API spec, the same default an `Ingress` with one
`/` path rule gives, so this chart's `HTTPRoute` sets none explicitly.

## Values

See `values.yaml` for the full, commented list. The ones most deployments touch:

| Key | Description | Default |
| --- | --- | --- |
| `image.repository` | Container image | `ghcr.io/ostendorf-cloud/ostendorf.dev` |
| `image.tag` | Image tag | `.Chart.AppVersion` |
| `replicaCount` | Pod replicas | `1` |
| `service.port` | Public Service and container port, the only one `Ingress`/`HTTPRoute` ever route to | `8080` |
| `service.adminPort` | govue's admin listener: health, readiness and metrics, in-cluster only | `8081` |
| `ingress.enabled` | Create an `Ingress`. Mutually exclusive with `httpRoute.enabled`; see "External routing" above. | `false` |
| `ingress.annotations` | Annotations for the `Ingress` | `{}` |
| `ingress.tls` | TLS configuration for the `Ingress` | `[]` |
| `httpRoute.enabled` | Create a Gateway API `HTTPRoute`. Mutually exclusive with `ingress.enabled`; see "External routing" above. | `false` |
| `httpRoute.hostnames` | Hostnames the `HTTPRoute` matches | `[]` |
| `httpRoute.annotations` | Annotations for the `HTTPRoute` | `{}` |
| `httpRoute.parentRefs.name` | Name of the `Gateway` to attach to. Required when `httpRoute.enabled` is true | `""` |
| `httpRoute.parentRefs.namespace` | Namespace the `Gateway` lives in. Required when `httpRoute.enabled` is true; see "External routing" above for why this has no same-namespace fallback | `""` |
| `httpRoute.parentRefs.sectionName` | Specific listener on the `Gateway` to attach to, when it exposes more than one | `""` |
| `config.umami.url` | `UMAMI_URL`: base URL of the Umami instance. Not a secret. Page views are counted only when this and `config.umami.websiteId` are both set; one alone fails the start. See `docs/analytics.md` in the repository. | `""` |
| `config.umami.websiteId` | `UMAMI_WEBSITE_ID`: UUID of the website in Umami. Not a secret. | `""` |
| `config.trustedProxyCidrs` | `TRUSTED_PROXIES`: CIDRs of the `Ingress` controller or `Gateway` in front. Without it every request comes from the proxy and nothing is counted. | `""` |
| `resources` | Pod resource requests/limits | see `values.yaml` |

## Upgrading

No breaking change has shipped yet; `Chart.yaml`'s `version` tracks that going forward.

## Uninstalling

```sh
helm uninstall ostendorf-dev
```

Removes the `Deployment`, `Service`, `ServiceAccount`, and the `Ingress` or `HTTPRoute`,
whichever was enabled, all created by this same chart.

## Verification

This chart is verified with:

- `helm lint`, over the default values and the documented ingress-enabled and httpRoute-enabled
  combinations. `helm lint` never propagates a template's `fail` call; it logs it at INFO and
  still exits zero, so it covers a values mistake schema validation would catch, not this
  chart's own render-time guard.
- `helm template`, for the same combinations as `helm lint` above, and for the must-fail set:
  one combination per render-time guard in `templates/deployment.yaml` and
  `templates/httproute.yaml`, each asserted to fail with that guard's own message rather than
  merely to render. `helm lint` cannot stand in for this: see the bullet above.
- `kubeconform`, against the rendered manifests of the must-succeed combinations. The default
  schema set carries no Gateway API schemas, so an httpRoute-enabled render's `HTTPRoute` is
  validated with a second `-schema-location` added after the default one, pointing at the
  community [CRDs-catalog](https://github.com/datreeio/CRDs-catalog).
- [`helm-unittest`](https://github.com/helm-unittest/helm-unittest), run with
  `helm unittest charts/ostendorf.dev`; see `mise.toml` for how the plugin is pinned. Its
  `failedTemplate` suites cover the same render-time guards as the must-fail set above,
  independently: both have to agree before a guard counts as checked.

`task lint:helm` and `task validate:kubeconform` hold the must-succeed and must-fail `--set`
combinations; `task test:helm` runs `helm-unittest` separately. Each has its own job in CI. A
live end-to-end install into a real cluster is a separate, tracked follow-up, not part of this
chart's static verification.
