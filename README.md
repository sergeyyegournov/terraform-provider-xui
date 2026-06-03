# Terraform Provider for 3x-ui (`xui`)

Terraform provider to manage [3x-ui](https://github.com/MHSanaei/3x-ui/) **v3+** panels over the HTTP API. It targets inbounds, per-protocol clients, panel settings, and Xray templates—the same objects you configure in the web UI.

Requires a 3x-ui panel that exposes the v3 API (`/panel/api/*`, CSRF-protected session auth, optional API tokens).

## Features

- **Inbounds** — create, update, import, and destroy Xray inbounds (`xui_inbound`).
- **Clients** — manage users on an existing inbound via the dedicated clients API (no inbound RMW races):
  - `xui_vless_client`
  - `xui_vmess_client`
  - `xui_trojan_client`
  - `xui_shadowsocks_client`
  - `xui_hysteria_client`
- **Panel & Xray** — `xui_panel_settings`, `xui_xray_template`.
- **Data sources** — `xui_inbounds`, `xui_inbound` (optional protocol filter).

Client secrets (`uuid`, `password`, `auth`) can be set in Terraform or left unset so the panel generates them and the provider reads them into state after create.

## Requirements

- [Terraform](https://www.terraform.io/downloads) >= 1.0
- A reachable 3x-ui v3+ panel

## Provider configuration

```hcl
terraform {
  required_providers {
    xui = {
      source  = "sergeyyegournov/xui" # after Terraform Registry publish; see below for local dev
      version = "~> 0.5"
    }
  }
}

provider "xui" {
  base_url = "https://panel.example.com/your-random-path/" # trailing slash as in the panel URL

  # Option A: API token (recommended for inbounds / clients)
  api_token = var.xui_api_token

  # Option B: username + password (session + CSRF)
  # username = var.xui_username
  # password = var.xui_password

  # Option A + B: token for /panel/api/*, password session for settings / Xray template
  # username = var.xui_username
  # password = var.xui_password
  # api_token = var.xui_api_token

  insecure_skip_verify = false # set true only for self-signed TLS in lab setups
}
```

| Argument | Description |
|----------|-------------|
| `base_url` | Panel root URL including the random path prefix. |
| `api_token` | Bearer token for `/panel/api/*` (Settings → API tokens in the panel). |
| `username` / `password` | Session auth; required for `xui_panel_settings` and `xui_xray_template`, and when not using a token. |
| `insecure_skip_verify` | Skip TLS certificate verification. |

You must set **`api_token`**, or **`username` and `password`**, or both (token plus credentials for settings endpoints).

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

Acceptance tests use [testcontainers](https://golang.testcontainers.org/) with `ghcr.io/mhsanaei/3x-ui:v3.2.0`. Docker must be running.

## Releasing

Tagged releases are built with GoReleaser and signed checksums. See [RELEASING.md](RELEASING.md).

## Contributing

Issues and pull requests are welcome. Run `make test` and `make testacc` before opening a PR.
