package logging

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

const (
	FieldAppname   = "appname"
	FieldComponent = "component"
)

type mod func(*logrus.Entry) error

func WithAppName(appName string) mod {
	return func(e *logrus.Entry) error {
		e.Data[FieldAppname] = appName
		return nil
	}
}

func WithComponentName(component string) mod {
	return func(e *logrus.Entry) error {
		e.Data[FieldComponent] = component
		return nil
	}
}

type hookWrapper struct {
	mods []mod
}

var _ logrus.Hook = (*hookWrapper)(nil)

func (h *hookWrapper) Levels() []logrus.Level {
	return logrus.AllLevels
}

func (h *hookWrapper) Fire(entry *logrus.Entry) error {
	for _, mod := range h.mods {
		if err := mod(entry); err != nil {
			return fmt.Errorf("logging mod encountered: %w", err)
		}
	}

	return nil
}

func Setup(mods ...mod) {
	logLevel, err := logrus.ParseLevel(os.Getenv("LOG_LEVEL"))
	if err != nil {
		logLevel = logrus.InfoLevel
	}

	if len(mods) > 0 {
		logrus.AddHook(&hookWrapper{mods: mods})
	}

	logrus.SetLevel(logLevel)
	logrus.SetOutput(os.Stdout)
	logrus.SetFormatter(&logrus.JSONFormatter{})

	if logLevel == logrus.DebugLevel {
		logrus.Debugf("Debug logging enabled~")
	}
}

func GinDefaultColorlessFormatter(param gin.LogFormatterParams) string {
	// Just use the default gin log formatter to format this line, but also log a warning
	if param.Latency > time.Minute {
		param.Latency = param.Latency.Truncate(time.Second)
	}
	return fmt.Sprintf("[GIN] %v |%3d| %13v | %15s |%-7s %#v\n%s",
		param.TimeStamp.Format("2006/01/02 - 15:04:05"),
		param.StatusCode,
		param.Latency,
		param.ClientIP,
		param.Method,
		param.Path,
		param.ErrorMessage,
	)
}

func GinJsonLogFormatter(param gin.LogFormatterParams) string {
	jsonBytes, err := json.Marshal(map[string]any{
		"timestamp":            param.TimeStamp,
		"status":               param.StatusCode,
		"method":               param.Method,
		"path":                 param.Path,
		"latency":              param.Latency,
		"client_ip":            param.ClientIP,
		"error_message":        param.ErrorMessage,
		"response_body_size":   param.BodySize,
		"request_context_keys": param.Keys,
		"request_host":         param.Request.Host,
	})
	if err != nil {
		logrus.Warnf("could not marshal log entry to JSON: %v", err)
		return GinDefaultColorlessFormatter(param)
	}

	buf := bytes.NewBuffer(jsonBytes)
	return buf.String() + "\n"
}
