package api

import (
	"fmt"
	"mime"
	"net/http"
	"os"
	"path"
	"slices"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/imroc/req/v3"
	ollamaApi "github.com/ollama/ollama/api"
	"github.com/sirupsen/logrus"
	apiv1 "github.com/usememos/memos/proto/gen/api/v1"
	"github.com/vimeo/go-magic/magic"

	"github.com/pirogoeth/apps/pkg/ctxtools"
	"github.com/pirogoeth/apps/voice-memos/prompts"
)

type v1MemoEndpoints struct {
	*ApiContext
}

type transcriptionResponse struct {
	Text string `json:"text"`
}

func (e *v1MemoEndpoints) RegisterRoutesTo(router *gin.RouterGroup) {
	router.POST("/memo", e.createMemo)
}

func (e *v1MemoEndpoints) createMemo(ctx *gin.Context) {
	// Pull audio file from input field
	file, err := ctx.FormFile("audio")
	if err != nil {
		logrus.Errorf("could get audio file: %s", err)
		ctx.AbortWithStatusJSON(http.StatusInternalServerError, &gin.H{
			"message": ErrFailedToBind,
			"error":   err.Error(),
		})
		return
	}

	// Write a timestamped file to the uploads directory
	receivedTime := time.Now()
	timestamp := receivedTime.Format(time.RFC3339)
	fileOnDisk := path.Join(
		e.Config.Uploads.Dir,
		fmt.Sprintf("%d", receivedTime.Year()),
		fmt.Sprintf("%d", receivedTime.Month()),
		timestamp,
	)

	if err := ctx.SaveUploadedFile(file, fileOnDisk); err != nil {
		logrus.Errorf("could not write audio file to disk: %s", err)
		ctx.AbortWithStatusJSON(http.StatusInternalServerError, &gin.H{
			"message": ErrFailedToWrite,
			"error":   err.Error(),
		})
		return
	}

	// Send to faster-whisper-server for transcription
	fasterWhisperTranscribeUrl := fmt.Sprintf(
		"%s/v1/audio/transcriptions",
		e.Config.FasterWhisperServer.BaseUrl,
	)
	var result transcriptionResponse
	transcribeResp, err := req.R().
		SetContext(ctx).
		SetFiles(map[string]string{
			"file": fileOnDisk,
		}).
		SetFormData(map[string]string{
			"model": e.Config.FasterWhisperServer.Model,
		}).
		SetSuccessResult(&result).
		Post(fasterWhisperTranscribeUrl)
	if err != nil {
		logrus.Errorf("error firing transcription request: %s", err)
		ctx.AbortWithStatusJSON(http.StatusInternalServerError, &gin.H{
			"message": ErrTranscriptionFailed,
			"error":   err.Error(),
		})
		return
	}

	if transcribeResp.IsErrorState() {
		logrus.Errorf("could not transcribe audio: %s", err)
		ctx.AbortWithStatusJSON(http.StatusInternalServerError, &gin.H{
			"message":  ErrTranscriptionFailed,
			"response": transcribeResp.String(),
		})
		return
	}

	logrus.Debugf("Transcribed audio: %s", result.Text)

	// Create memo from transcription
	memo, err := e.MemosClient.MemoService.CreateMemo(
		ctxtools.BoundedBy(e.MemosClient.Context(), ctx),
		&apiv1.CreateMemoRequest{
			Content: fmt.Sprintf("%s\n%s", result.Text, e.Config.MemoSettings.Suffix),
		},
	)
	if err != nil {
		logrus.Errorf("creating memo: %s", err)
		ctx.AbortWithStatusJSON(http.StatusInternalServerError, &gin.H{
			"message": ErrFailedToCreateMemo,
			"error":   err.Error(),
		})
		return
	}

	// Upload audio file to memos server for attach
	fileContents, err := os.ReadFile(fileOnDisk)
	if err != nil {
		logrus.Errorf("could not read audio file from disk for attachment: %s", err)
		ctx.AbortWithStatusJSON(http.StatusInternalServerError, &gin.H{
			"message": ErrFailedToRead,
			"error":   err.Error(),
		})
		return
	}

	mimeType := magic.MimeFromBytes(fileContents)
	logrus.Debugf("Magic returns mimetype %s", mimeType)

	var extension string
	extensions, err := mime.ExtensionsByType(mimeType)
	if err != nil || len(extensions) == 0 {
		logrus.Errorf("could not load extension for mimetype %s, defaulting to `.bin` (err %s)", mimeType, err)
		extension = ".bin"
	} else {
		extension = extensions[0]
	}

	logrus.Debugf("Audio file is %s (ext: %s)", mimeType, extension)

	e.MemosClient.ResourceService.CreateResource(
		ctxtools.BoundedBy(e.MemosClient.Context(), ctx),
		&apiv1.CreateResourceRequest{
			Resource: &apiv1.Resource{
				Filename: fmt.Sprintf("audio-%s%s", timestamp, extension),
				Content:  fileContents,
				Size:     int64(len(fileContents)),
				Memo:     &memo.Name,
			},
		},
	)
}

// callModelForTags temporarily unused
func (e *v1MemoEndpoints) callModelForTags(ctx *gin.Context, memoText string) {
	// Collect existing tags from memos
	memos, err := e.MemosClient.MemoService.ListMemos(
		ctxtools.BoundedBy(e.MemosClient.Context(), ctx),
		&apiv1.ListMemosRequest{
			// TODO: I'm lazy so let's grab a bunch of memos until I feel like paginating
			PageSize: 1000,
		},
	)
	if err != nil {
		logrus.Errorf("error listing memos: %s", err)
		ctx.AbortWithStatusJSON(http.StatusInternalServerError, &gin.H{
			"message": ErrFailedToGenerate,
			"error":   err.Error(),
		})
		return
	}

	tags := make([]string, 100)
	for _, memo := range memos.Memos {
		for _, tag := range memo.Tags {
			if !slices.Contains(tags, tag) {
				tags = append(tags, tag)
			}
		}
		tags = append(tags, memo.Tags...)
	}

	userPrompt, err := prompts.RenderPromptTemplateTagCreation(&prompts.PromptTemplateTagCreationInputs{
		MemoText: memoText,
		Tags:     tags,
	})
	if err != nil {
		logrus.Errorf("rendering user prompt: %s", err)
		ctx.AbortWithStatusJSON(http.StatusInternalServerError, &gin.H{
			"message": ErrFailedToGenerate,
			"error":   err.Error(),
		})
		return
	}

	// Use Ollama to select tags for the memo based on the list of existing tags
	var modelToolCalls []ollamaApi.ToolCall
	reqStreaming := false
	err = e.OllamaClient.Chat(ctx, &ollamaApi.ChatRequest{
		Model:  e.Config.OllamaServer.Model,
		Format: "json",
		Stream: &reqStreaming,
		Tools:  prompts.SystemToolsTagCreation,
		Messages: []ollamaApi.Message{
			{
				Role:    "system",
				Content: prompts.SystemPromptTagCreation,
			},
			{
				Role:    "user",
				Content: userPrompt,
			},
		},
	}, func(resp ollamaApi.ChatResponse) error {
		logrus.Debugf("Model response: %#v", resp)
		modelToolCalls = resp.Message.ToolCalls
		return nil
	})
	if err != nil {
		logrus.Errorf("generating tags for memo: %s", err)
		ctx.AbortWithStatusJSON(http.StatusInternalServerError, &gin.H{
			"message": ErrFailedToGenerate,
			"error":   err.Error(),
		})
		return
	}

	logrus.Infof("Model called tools: %#v", modelToolCalls)
}
