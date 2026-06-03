resource "xui_inbound" "example" {
  protocol = "shadowsocks"
  remark   = "example-shadowsocks"
  port     = 8388
  settings = jsonencode({
    method   = "aes-256-gcm"
    password = "inbound-shared-secret"
    network  = "tcp,udp"
    clients  = []
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

resource "xui_shadowsocks_client" "example" {
  inbound_id = xui_inbound.example.id
  email      = "ss-client@example.com"
}
