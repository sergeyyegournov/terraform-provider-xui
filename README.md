# Terraform Provider for 3x-ui (`xui`)

Terraform provider to manage [3x-ui](https://github.com/MHSanaei/3x-ui/) **v3+** panels over the HTTP API. It targets inbounds, per-protocol clients, panel settings, and Xray templates—the same objects you configure in the web UI.

Requires a 3x-ui panel that exposes the v3 API (`/panel/api/*`, CSRF-protected session auth or optional API tokens). Requires 3x-ui **v3.4.0+** (settings/Xray under `/panel/api`, `panelOutbound` / `remarkTemplate` field names).

## Panel compatibility

This provider release (**v0.9.0**) is developed and acceptance-tested against **[3x-ui v3.9.0](https://github.com/MHSanaei/3x-ui/releases/tag/v3.9.0)**. Older panels from **v3.4.0+** still work for core resources. Attributes added in 3.6–3.9 stay optional: on an older panel that omits them, the provider keeps the prior state instead of zeroing it.

### 3.9 behavior

- **Inbound enable.** `POST /panel/api/inbounds/update/:id` ignores `enable` (and `settings.clients`). Create still honors `enable`. The provider calls `POST /panel/api/inbounds/setEnable/:id` when `enable` changes. `exclude_from_sub` and `sub_sort_index` are sent on every update; the panel resets them when they are omitted (`sub_sort_index` `0` is stored as `1`).
- **`sub_profile_mode`.** `none`, `builtin`, or `custom`. Omit the attribute and the provider leaves the panel value in place (an upgraded panel with a saved profile URL is `custom`; a factory install is `none`). An explicit `none` keeps a stored URL but stops sending it.
- **Weekly client renewal.** `reset_weekday` is 1–7 (Monday–Sunday). It cannot be combined with `reset` or `reset_day`. Nodes older than 3.9 ignore it, and an older panel can delete a weekly-only client.
- **AmneziaWG `keep_alive`.** `0` is sent explicitly so an update turns keepalive off. Omitting the field on the API would keep the stored value.
- **`restart_xray_on_client_disable`.** Since 3.8.5 this also restarts Xray on a manual client disable or delete, not only expiry or quota.

After the panel's Xray-core v26.9.30 startup migration, update stored JSON before the next apply. The new core rewrites XDNS finalmasks to an object and moves WireGuard outbound `domainStrategy` into `sockopt` / `targetStrategy`. An `xui_xray_template` or `stream_settings` value in the old shape plans a revert, and the new core rejects it.

Other panel checks the provider does not encode: an empty REALITY `minClientVer` no longer implies a minimum client version (3.8); WireGuard and AmneziaWG `allowedIPs` that overlap another client are rejected on save (existing overlaps keep working until the next client update); saving or enabling an inbound with a port collision is an API error (3.8.5), including ports an AmneziaWG peer forwards.

### Not implemented yet

These 3x-ui surfaces exist in the panel API but are **not** managed as Terraform resources. Declarative CRUD that is a poor fit for one resource, and every imperative action, stay out of the provider.

Declarative APIs left to a later change:

- **Hosts** — add / update / delete by `groupId` (`/panel/api/hosts`). Reorder and bulk enable / delete are actions.
- **Nodes** — add / update / delete (`/panel/api/nodes`). Probe, certificate fingerprint, mTLS reload, and `updatePanel` are actions. An inbound `nodeId` is preserved when the provider omits it.
- **Outbound subscriptions** — create / update / delete (`/panel/api/xray/outbound-subs`). Refresh, move, and parse are actions.
- **Subscription balancers** — create / update / delete (`/panel/api/sub-balancers`).
- **Client groups** — create / rename / delete. Membership is the client `group` attribute. Deleting a group clears that label on every member, so a separate resource is a poor fit.
- **Fallbacks** — `POST /panel/api/inbounds/:id/fallbacks` replaces the whole list. A nested block later, not one resource per rule.
- **Per-client external links** — replace-all (`/panel/api/clients/{email}/externalLinks`).
- **MTProto and WireGuard clients** — the clients API already accepts them (`secret` / `adTag`, and WireGuard keys). They stay out of this release, same as the 3.7 provider shipping AmneziaWG only.
- **HWID device registry** — list / clear / delete individual devices (`/panel/api/clients/hwids/…`). Per-client **`limit_hwid`** is supported on client resources.
- **API token management** — create / enable / delete panel API tokens (`/panel/api/setting/apiTokens/…`). v3.9 only changes CLI regeneration (scope and expiry are kept). Consuming a token via provider `api_token` is supported. On 3.7+ the token must be **admin** scope. A wrong Bearer token is **401**; cookie auth still retries 404/403.

Imperative APIs (no resources): traffic reset and quota edits, delete depleted / orphan / all clients, bulk enable / disable / delete / adjust / attach / detach, group traffic reset, online clients, IP lists, link and QR helpers, renewal preview, UUID / key / certificate generators, REALITY scans, status, logs, metrics, restart or install Xray, update the panel or geofiles, import a database, backup to Telegram, SMTP / Telegram / Discord tests, WARP / Nord / PIA actions, outbound tests, balancer override, and route tests.

## Features

- **Inbounds** — create, update, import, and destroy Xray inbounds (`xui_inbound`).
- **Clients** — manage users on an existing inbound via the dedicated clients API (no inbound RMW races):
  - `xui_vless_client`
  - `xui_vmess_client`
  - `xui_trojan_client`
  - `xui_shadowsocks_client`
  - `xui_hysteria_client`
  - `xui_amneziawg_client` (AmneziaWG peers; inbound via `xui_inbound` with `protocol = "amneziawg"`)
  - `xui_tuic_client` (TUIC v5 users; inbound via `xui_inbound` with `protocol = "tuic"`)
- **Panel & Xray** — `xui_panel_settings`, `xui_xray_template`.
- **Data sources** — `xui_inbounds`, `xui_inbound` (optional protocol filter).

Client secrets (`uuid`, `password`, `auth`) can be set in Terraform or left unset so the panel generates them and the provider reads them into state after create.

## Requirements

- [Terraform](https://www.terraform.io/downloads) >= 1.0
- A reachable 3x-ui **v3.4.0+** panel (recommended: **v3.9.0**, matching this provider)

## Provider configuration

```hcl
terraform {
  required_providers {
    xui = {
      source  = "sergeyyegournov/xui" # after Terraform Registry publish; see below for local dev
      version = "~> 0.9"
    }
  }
}

provider "xui" {
  base_url = "https://panel.example.com/your-random-path/" # trailing slash as in the panel URL

  # Option A: API token (recommended; on 3x-ui v3.7+ use an admin-scope token)
  api_token = var.xui_api_token

  # Option B: username + password (session + CSRF)
  # username = var.xui_username
  # password = var.xui_password

  insecure_skip_verify = false # set true only for self-signed TLS in lab setups
}
```

| Argument | Description |
|----------|-------------|
| `base_url` | Panel root URL including the random path prefix. |
| `api_token` | Bearer token for `/panel/api/*` (Settings → API tokens in the panel). On 3x-ui **v3.7+**, use an **admin**-scope token (`monitor` / `node-sync` cannot manage resources). |
| `username` / `password` | Session auth when not using a token. |
| `insecure_skip_verify` | Skip TLS certificate verification. |

You must set **`api_token`**, or **`username` and `password`**.

## Quick start

```hcl
resource "xui_inbound" "vless" {
  protocol = "vless"
  remark   = "terraform-vless"
  port     = 443
  settings = jsonencode({
    clients    = []
    decryption = "none"
  })
  stream_settings = jsonencode({
    network  = "tcp"
    security = "none"
    tcpSettings = {
      acceptProxyProtocol = false
      header              = { type = "none" }
    }
  })
  sniffing = "{}"
}

resource "xui_vless_client" "user" {
  inbound_id = xui_inbound.vless.id
  email      = "user@example.com"
  # uuid omitted — panel generates; provider stores it in state after create
}
```

More examples live under [`examples/`](examples/) (per-resource `resource.tf` and `import.sh` files).

### Importing clients

Import ID format: **`inbound_id:email`**

```bash
terraform import xui_vless_client.user 42:user@example.com
```

## Documentation

- Generated provider docs: [`docs/`](docs/) (run `make docs` to refresh from schema).
- Registry-style resource pages: `docs/resources/*.md`, `docs/data-sources/*.md`.

## Installing the provider

### Terraform Registry

After the provider is published under your namespace, use a `required_providers` block with `source = "<namespace>/xui"` and a released version. See [RELEASING.md](RELEASING.md) for publish steps.

### Local development build

```bash
git clone https://github.com/sergeyyegournov/terraform-provider-xui.git
cd terraform-provider-xui
go build -o terraform-provider-xui .
```

Or use the debug workflow:

```bash
cd examples/debug
./dev-setup.sh
export TF_CLI_CONFIG_FILE="$(pwd)/dev.terraform.rc"
terraform init
terraform plan
```

`dev-setup.sh` wires `example.com/xui/xui` dev overrides to your local binary (see `examples/debug/terraform.rc.sample`).

## Development

```bash
make build    # build provider binary
make test     # unit tests
make testacc  # acceptance tests (Docker + 3x-ui container)
make docs     # regenerate docs/ from schema
```

Acceptance tests use [testcontainers](https://golang.testcontainers.org/) with `ghcr.io/mhsanaei/3x-ui:v3.9.0`. Docker must be running.

## Releasing

Tagged releases are built with GoReleaser and signed checksums. See [RELEASING.md](RELEASING.md).

## Contributing

Issues and pull requests are welcome. Run `make test` and `make testacc` before opening a PR.
