package prompts

import (
	"bytes"
	_ "embed"
	"encoding/json"
	"fmt"
	tt "text/template"

	ollamaApi "github.com/ollama/ollama/api"
)

//go:embed system-prompt-tag-creation.txt
var SystemPromptTagCreation string

//go:embed system-tools-tag-creation.json
var rawSystemToolsTagCreation string
var SystemToolsTagCreation ollamaApi.Tools

//go:embed prompt-template-tag-creation.tmpl
var rawPromptTemplateTagCreation string
var PromptTemplateTagCreation *tt.Template

func init() {
	if err := json.Unmarshal([]byte(rawSystemToolsTagCreation), &SystemToolsTagCreation); err != nil {
		panic(err)
	}

	PromptTemplateTagCreation = tt.Must(
		tt.New("prompt-template-tag-creation.tmpl").Parse(rawPromptTemplateTagCreation),
	)
}

type PromptTemplateTagCreationInputs struct {
	MemoText string
	Tags     []string
}

func RenderPromptTemplateTagCreation(inputs *PromptTemplateTagCreationInputs) (string, error) {
	buf := new(bytes.Buffer)

	if err := PromptTemplateTagCreation.Execute(buf, inputs); err != nil {
		return "", fmt.Errorf("rendering prompt-template-tag-creation.tmpl: %w", err)
	}

	return buf.String(), nil
}
