package mqtt

type Config struct {
	Servers      []string `json:"servers"`
	ClientID     string   `json:"client_id"`
	Username     string   `json:"username"`
	Password     string   `json:"password"`
	CleanSession bool     `json:"clean_session"`
	Keepalive    int64

	TLSConfig *TLSConfig `json:"tls,omitempty"`
}

type TLSConfig struct {
	ServerName             string   `json:"server_name"`
	InsecureSkipVerify     bool     `json:"insecure_skip_verify"`
	RootCACertificatePaths []string `json:"root_ca_certificates"`
}
