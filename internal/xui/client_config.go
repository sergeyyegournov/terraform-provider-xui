package xui

// ClientConfig holds connection settings for a 3x-ui panel (v3+).
type ClientConfig struct {
	BaseURL            string
	Username           string
	Password           string
	APIToken           string
	InsecureSkipVerify bool
}
