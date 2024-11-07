package types

import (
	"github.com/pirogoeth/apps/pkg/config"
)

type MemosServerCfg struct {
	// GrpcEndpoint is the endpoint that should be used by grpc.DialContext to connect to the Memos server.
	// It should be formatted according to the [gRPC naming document](https://github.com/grpc/grpc/blob/master/doc/naming.md).
	GrpcEndpoint string `json:"grpc_endpoint" envconfig:"MEMOS_SERVER_GRPC_ENDPOINT"`
	// GrpcInsecure switches the credentials that will be used to connect to the Memos server.
	// If true, uses the `insecure` credentials provider. Otherwise, uses the system x509 store to validate the server certificate.
	GrpcInsecure bool   `json:"grpc_insecure" envconfig:"MEMOS_SERVER_GRPC_INSECURE" default:"false"`
	ApiToken     string `json:"api_token" envconfig:"MEMOS_SERVER_API_TOKEN"`
}

type Config struct {
	config.CommonConfig

	Uploads struct {
		Dir string `json:"dir" envconfig:"UPLOADS_DIR"`
	} `json:"uploads"`

	MemosServer MemosServerCfg `json:"memos_server"`

	MemoSettings struct {
		Suffix string `json:"suffix" envconfig:"MEMO_SETTINGS_SUFFIX"`
	} `json:"memo_settings"`

	OllamaServer struct {
		BaseUrl string `json:"base_url" envconfig:"OLLAMA_SERVER_BASE_URL" default:"http://localhost:11434"`
		Model   string `json:"model" envconfig:"OLLAMA_SERVER_MODEL" default:"llama3.2:latest"`
	} `json:"ollama_server"`

	FasterWhisperServer struct {
		BaseUrl string `json:"base_url" envconfig:"FASTER_WHISPER_SERVER_BASE_URL" default:"http://localhost:8000"`
		Model   string `json:"model" envconfig:"FASTER_WHISPER_SERVER_MODEL" default:"Systran/faster-distil-whisper-large-v3"`
	} `json:"faster_whisper_server"`
}
