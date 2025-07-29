package config

type HTTPConfig struct {
	ListenAddress string `json:"listen_address" envconfig:"HTTP_LISTEN_ADDRESS" default:":8000"`
}

type TracingConfig struct {
	Enabled bool `json:"enabled" envconfig:"TRACING_ENABLED" default:"false"`

	// SamplerRate is the rate at which samples should be sent to the tracing backend.
	SamplerRate float64 `json:"sampler_rate" envconfig:"TRACING_SAMPLER_RATE" default:"1.0"`

	// ExporterEndpoint is the URL to the OTLP ingest endpoint
	ExporterEndpoint string `json:"exporter_endpoint" envconfig:"TRACING_EXPORTER_ENDPOINT"`

	// ExporterProtocol is the choice of protocol the exporter should use to submit traces. Either http or grpc
	ExporterProtocol string `json:"exporter_protocol" envconfig:"TRACING_EXPORTER_PROTOCOL" default:"grpc"`

	// ExporterInsecure disables TLS validation on the connection
	ExporterInsecure bool `json:"exporter_insecure" envconfig:"TRACING_EXPORTER_INSECURE" default:"false"`

	// ExporterHeaders is HTTP headers to be provided to the exporter
	ExporterHeaders map[string]string `json:"exporter_headers" envconfig:"TRACING_EXPORTER_HEADERS"`
}

type CommonConfig struct {
	HTTP HTTPConfig `json:"http"`

	Tracing TracingConfig `json:"tracing"`
}
