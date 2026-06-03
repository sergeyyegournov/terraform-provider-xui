resource "xui_inbound" "example" {
  protocol = "hysteria"
  remark   = "example-hysteria"
  port     = 8445
  settings = jsonencode({
    version = 1
    clients = []
  })
  stream_settings = jsonencode({
    network  = "udp"
    security = "none"
  })
  sniffing = "{}"
}

resource "xui_hysteria_client" "example" {
  inbound_id = xui_inbound.example.id
  email      = "hysteria-client@example.com"
}
