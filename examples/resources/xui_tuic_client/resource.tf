resource "xui_inbound" "example" {
  protocol = "tuic"
  remark   = "example-tuic"
  port     = 8446
  settings = jsonencode({
    clients = []
  })
  stream_settings = "{}"
  sniffing        = "{}"
}

resource "xui_tuic_client" "example" {
  inbound_id = xui_inbound.example.id
  email      = "tuic-client@example.com"
  # uuid and password omitted — the panel generates both
}
