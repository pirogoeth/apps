package config

type CommonConfig struct {
	HTTP struct {
		ListenAddress string `json:"listen_address" envconfig:"HTTP_LISTEN_ADDRESS"`
	} `json:"http"`
}
