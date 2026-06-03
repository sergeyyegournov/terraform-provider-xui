resource "xui_inbound" "example" {
  protocol = "vmess"
  remark   = "example-vmess"
  port     = 8443
  settings = jsonencode({
    clients = []
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

resource "xui_vmess_client" "example" {
  inbound_id = xui_inbound.example.id
  email      = "vmess-client@example.com"
  security   = "auto"
}
