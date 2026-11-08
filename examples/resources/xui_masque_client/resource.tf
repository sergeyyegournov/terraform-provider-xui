resource "xui_inbound" "example" {
  protocol = "masque"
  remark   = "example-masque"
  port     = 8447
  settings = jsonencode({
    clients = []
    address = ["10.14.0.1/24", "fd14::1/64"]
  })
  stream_settings = jsonencode({
    network  = "tcp"
    security = "tls"
  })
  sniffing = "{}"
}

resource "xui_masque_client" "example" {
  inbound_id = xui_inbound.example.id
  email      = "masque-client@example.com"
  # password omitted — the panel generates one
}
