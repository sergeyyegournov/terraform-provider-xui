resource "xui_inbound" "example" {
  protocol = "amneziawg"
  remark   = "example-amneziawg"
  port     = 51820
  settings = jsonencode({
    server = {
      subnetIp   = "10.8.1.0"
      subnetCidr = 24
    }
    clients = []
  })
  stream_settings = "{}"
  sniffing        = "{}"
}

resource "xui_amneziawg_client" "example" {
  inbound_id = xui_inbound.example.id
  email      = "awg-client@example.com"
  # private_key / public_key omitted — panel generates a keypair
  # keep_alive accepts a seconds string or an AmneziaWG range, e.g. "15-25"
  keep_alive = "25"
}
