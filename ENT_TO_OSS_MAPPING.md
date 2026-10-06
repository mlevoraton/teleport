# WarpNavigator: Teleport v15.5.4 Enterprise → OSS Capability Map

This map describes capabilities verified in this checkout. The presence of a
Go type, protobuf, CRD, client, or local storage implementation does not prove
that the OSS server exposes the feature.

## Verified scope

| Capability                              | OSS status                | Evidence / safe approach                                                                                                                    |
| --------------------------------------- | ------------------------- | ------------------------------------------------------------------------------------------------------------------------------------------- |
| GitHub SSO                              | Supported                 | Existing GitHub connector CRUD and web login flow; use `teams_to_roles`.                                                                    |
| OIDC SSO                                | Enterprise-gated          | `defaultModules.Features()` leaves `OIDC` false. Do not expose OIDC as an OSS feature merely because connector types exist.                 |
| SAML SSO                                | Enterprise-gated          | `defaultModules.Features()` leaves `SAML` false. Do not add an OSS SAML editor without a real server implementation and entitlement review. |
| Roles / basic RBAC                      | Supported                 | Use the existing Role YAML editor and existing role CRUD.                                                                                   |
| Login Rules                             | Not exposed in OSS        | `lib/auth/grpcserver.go` registers `loginrulev1.NotImplementedService{}`.                                                                   |
| Access Lists                            | Not exposed over OSS gRPC | Local storage exists, but no Access List gRPC server is registered in this checkout. A generated client alone is insufficient.              |
| Advanced access workflows               | Enterprise-gated          | `defaultModules.Features()` leaves `AdvancedAccessWorkflows` false.                                                                         |
| Identity Governance / Access Monitoring | Enterprise-gated          | OSS module feature flags are disabled and related extension points are no-ops or return `NotImplemented`.                                   |
| Custom Enterprise theme license field   | Enterprise-gated          | OSS `CustomTheme` is empty. This fork's branding work must provide its own explicit configuration path.                                     |

## White-label implementation status

The branding configuration is wired end to end through one supported source:

- `proxy_service.ui.branding` in `teleport.yaml` is parsed into the proxy
  runtime configuration and emitted by `/web/config.js` as
  `WebConfig.Branding`.
- `cfg.init(window.GRV_CONFIG)` is the only browser-side configuration entry
  point. The earlier `window.__BRANDING__`, `window.__THEMES__`, and
  `window.__CFG__` alternatives were removed.
- Login logo, navigation logo, product name, favicon, and the compiled theme
  selector consume `cfg.branding` and retain WarpNavigator defaults when unset.
- Arbitrary CSS, a second MOTD, and unused domain configuration are excluded.
- Branding image URLs must be root-relative, preventing CSP drift and keeping
  assets on the Web UI origin. `custom_theme` can only select a theme already
  compiled into the UI.

Helm deployments can set the same source under `proxy.teleportConfig`:

```yaml
proxy:
  teleportConfig:
    proxy_service:
      ui:
        branding:
          product_name: Example Access
          logo: /branding/logo.svg
          logo_small: /branding/logo-small.svg
          favicon: /branding/favicon.png
          custom_theme: bblp
```

The chart does not serve those files automatically. Configure the same-origin
reverse proxy or ingress to serve the `/branding/` paths.

## GitHub connector deployment

`examples/chart/teleport-cluster/values.github-connector.yaml` configures the
chart to select a GitHub connector named `github`; it does not create the
dynamic connector resource.

Create the connector separately with `tctl create -f <private-file.yaml>` using
the schema-valid template in
`web/packages/teleport/src/AuthConnectors/templates/github.yaml`.

Security requirements:

- Never commit the real GitHub `client_secret`.
- Do not claim that `valueFrom.secretKeyRef` works in `GithubConnectorSpecV3`;
  the schema requires a string.
- Keep local authentication enabled until GitHub login has been tested, so an
  invalid connector does not lock administrators out.

## Safe Phase 3 scope

The existing role template already includes common OSS fields such as
`logins`, `node_labels`, `app_labels`, and `kubernetes_resources`.

Do not add `impersonate` permissions to the default new-role template: that
would grant elevated privileges to every role created from it. If an operator
needs impersonation, it should be an explicit, reviewed role change.

Do not implement Login Rule or Access List web handlers/routes until the OSS
server exposes and authorizes the corresponding service.

## Validation status

- Branding config and URL-validation Go tests pass.
- Login, navigation, favicon, and FormLogin focused Jest suites pass (38 tests,
  including WarpNavigator defaults and snapshots).
- Targeted Prettier checks pass; targeted ESLint is part of the final check.
- `helm lint` and `helm template` pass for the GitHub values example (apart
  from the chart's pre-existing metadata warning).
- `yarn type-check --pretty false` passes after generating the IronRDP package.
- The focused `lib/web` runtime test requires the generated
  `webassets/teleport/index.html`; compile-only Go checks do not.
