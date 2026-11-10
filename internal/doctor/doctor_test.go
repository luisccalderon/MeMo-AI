package doctor

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/luisccalderon/MeMo-AI/internal/detect"
	"github.com/luisccalderon/MeMo-AI/internal/install"
	"github.com/luisccalderon/MeMo-AI/internal/pi"
)

func modelCheck(t *testing.T, paths pi.Paths) Check {
	t.Helper()
	for _, check := range Run(paths, detect.Environment{}, "en").Checks {
		if check.Name == "Model" {
			return check
		}
	}
	t.Fatal("no Model check in the report")
	return Check{}
}

func installModel(t *testing.T, model string) pi.Paths {
	t.Helper()
	paths := pi.At(filepath.Join(t.TempDir(), "agent"))
	if _, err := install.Run(paths, install.Options{Lang: "en", Orchestrator: true, PrimaryModel: model}); err != nil {
		t.Fatal(err)
	}
	return paths
}

func TestLocalModelWithoutDeclaredProviderWarns(t *testing.T) {
	paths := installModel(t, "ollama/qwen3:8b")
	if got := modelCheck(t, paths); got.Status != Warn {
		t.Fatalf("status = %s (%s), want warn: Pi only lists models of declared providers", got.Status, got.Detail)
	}
}

func TestLocalModelWithDeclaredProviderIsOK(t *testing.T) {
	paths := installModel(t, "ollama/qwen3:8b")
	const models = `{"providers":{"ollama":{"baseUrl":"http://localhost:11434/v1","api":"openai-completions","apiKey":"ollama","models":[{"id":"qwen3:8b"}]}}}`
	if err := os.WriteFile(paths.ModelsFile, []byte(models), 0o644); err != nil {
		t.Fatal(err)
	}
	if got := modelCheck(t, paths); got.Status != OK {
		t.Fatalf("status = %s (%s), want ok", got.Status, got.Detail)
	}
}

func TestHostedModelIsNotSecondGuessed(t *testing.T) {
	paths := installModel(t, "anthropic/claude-sonnet-5-5")
	if got := modelCheck(t, paths); got.Status != OK {
		t.Fatalf("status = %s (%s): hosted providers authenticate outside models.json", got.Status, got.Detail)
	}
}
