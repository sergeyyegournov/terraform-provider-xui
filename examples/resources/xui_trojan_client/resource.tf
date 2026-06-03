resource "xui_inbound" "example" {
  protocol = "trojan"
  remark   = "example-trojan"
  port     = 8444
  settings = jsonencode({
    clients   = []
    fallbacks = []
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

resource "xui_trojan_client" "example" {
  inbound_id = xui_inbound.example.id
  email      = "trojan-client@example.com"
}
