package api

import (
	"github.com/gin-gonic/gin"
	ollamaApi "github.com/ollama/ollama/api"

	"github.com/pirogoeth/apps/voice-memos/clients/memos"
	"github.com/pirogoeth/apps/voice-memos/types"
)

const (
	// ErrInvalidParameter is used when the parameter value can't be parsed or is otherwise invalid
	ErrInvalidParameter = "invalid parameter value"
	// ErrNoQueryProvided is used when no query parameter is provided but is expected
	ErrNoQueryProvided = "no query parameter provided"
	// ErrFailedToBind is used when request body failed to bind to the destination object
	ErrFailedToBind = "failed to bind parameter"
	// ErrFailedToWrite is used when an input file could not be written to disk
	ErrFailedToWrite = "failed to write to disk"
	// ErrFailedToRead is used when an input file could not be read from disk
	ErrFailedToRead = "failed to read from disk"
	// ErrTranscriptionFailed
	ErrTranscriptionFailed = "failed to transcribe audio"
	// ErrFailedToCreateMemo
	ErrFailedToCreateMemo = "failed to create memo"
	// ErrFailedToGenerate
	ErrFailedToGenerate = "failed to generate response"
)

type ApiContext struct {
	// Config is the application configuration
	Config *types.Config
	// OllamaClient is an initialized HTTP client for the ollama server
	OllamaClient *ollamaApi.Client
	// MemosClient is an initialized gRPC client for the memos server
	MemosClient *memos.Client
}

func MustRegister(router *gin.Engine, apiContext *ApiContext) {
	groupV1 := router.Group("/v1")

	v1Memo := &v1MemoEndpoints{apiContext}
	v1Memo.RegisterRoutesTo(groupV1)
}
