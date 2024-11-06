package cmd

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"os"

	"github.com/gin-gonic/gin"
	ollamaApi "github.com/ollama/ollama/api"
	"github.com/spf13/cobra"
	"github.com/vimeo/go-magic/magic"

	"github.com/pirogoeth/apps/pkg/system"
	"github.com/pirogoeth/apps/voice-memos/api"
	"github.com/pirogoeth/apps/voice-memos/clients/memos"
	"github.com/pirogoeth/apps/voice-memos/types"
)

var serveCmd = &cobra.Command{
	Use:   "serve",
	Short: "Serve the voice-memos API",
	Run:   serveFunc,
}

type App struct {
	cfg *types.Config
}

func serveFunc(cmd *cobra.Command, args []string) {
	cfg := appStart(ComponentApi)
	gin.EnableJsonDecoderDisallowUnknownFields()

	app := &App{cfg}

	ctx, cancel := context.WithCancel(context.Background())

	// Initialize clients
	memosClient, err := memos.New(ctx, cfg.MemosServer)
	if err != nil {
		panic(fmt.Errorf("could not start: failed to initialize memos client: %w", err))
	}

	// Parse ollama URL
	ollamaBaseUrl, err := url.Parse(cfg.OllamaServer.BaseUrl)
	if err != nil {
		panic(fmt.Errorf("could not start: failed to parse ollama base URL: %w", err))
	}

	ollamaClient := ollamaApi.NewClient(ollamaBaseUrl, http.DefaultClient)

	// Initialize go-magic
	magic.AddMagicDir(magic.GetDefaultDir())

	router := system.DefaultRouter()
	api.MustRegister(router, &api.ApiContext{
		Config:       app.cfg,
		MemosClient:  memosClient,
		OllamaClient: ollamaClient,
	})

	go router.Run(app.cfg.HTTP.ListenAddress)

	sw := system.NewSignalWaiter(os.Interrupt)
	sw.Wait(ctx, cancel)
}
