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

func Setup() {
	logLevel, err := logrus.ParseLevel(os.Getenv("LOG_LEVEL"))
	if err != nil {
		logLevel = logrus.InfoLevel
	}

	logrus.SetLevel(logLevel)
	logrus.SetOutput(os.Stdout)
	logrus.SetFormatter(&logrus.JSONFormatter{})
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
