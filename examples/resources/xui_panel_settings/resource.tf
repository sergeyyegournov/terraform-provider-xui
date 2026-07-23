resource "xui_panel_settings" "this" {
  web_port      = 2053
  web_base_path = "/my-secret-path/"
  web_cert_file = "/etc/letsencrypt/live/example.com/fullchain.pem"
  web_key_file  = "/etc/letsencrypt/live/example.com/privkey.pem"

  session_max_age = 120
  time_location   = "Europe/Tallinn"

  tg_bot_enable  = true
  tg_bot_token   = var.tg_bot_token
  tg_bot_chat_id = var.tg_chat_id
  tg_memory      = 85

  smtp_enable = true
  smtp_host   = "smtp.example.com"
  smtp_port   = 587
  smtp_from   = "panel@example.com"
  smtp_to     = "admin@example.com"

  sub_enable               = true
  sub_port                 = 2096
  sub_path                 = "/my-sub/"
  sub_json_auto_detect     = true
  sub_clash_enable_routing = true

  restart_panel = true
}
