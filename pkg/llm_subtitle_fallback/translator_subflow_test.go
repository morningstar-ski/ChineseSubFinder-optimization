package llm_subtitle_fallback

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestResolveSubflowRootDirUsesConfiguredRoot(t *testing.T) {
	root := makeFakeSubflowRoot(t)

	got, err := resolveSubflowRootDir(root)
	if err != nil {
		t.Fatalf("resolveSubflowRootDir() error = %v", err)
	}
	if got != root {
		t.Fatalf("resolveSubflowRootDir() = %q, want %q", got, root)
	}
}

func TestResolveSubflowRootDirFallsBackToEnv(t *testing.T) {
	root := makeFakeSubflowRoot(t)
	t.Setenv("CSF_LLM_SUBTITLE_FALLBACK_SUBFLOW_ROOT", root)

	got, err := resolveSubflowRootDir(filepath.Join(t.TempDir(), "missing"))
	if err != nil {
		t.Fatalf("resolveSubflowRootDir() error = %v", err)
	}
	if got != root {
		t.Fatalf("resolveSubflowRootDir() = %q, want env fallback %q", got, root)
	}
}

func TestBuildTranslateEnvIncludesFallbackCredentials(t *testing.T) {
	root := makeFakeSubflowRoot(t)
	t.Setenv("PYTHONPATH", "C:\\existing")

	env := buildTranslateEnv(root, TranslateRequest{
		BaseURL: "https://example.com/v1",
		APIKey:  "secret-key",
	})
	joined := "\n" + strings.Join(env, "\n") + "\n"

	if strings.Contains(joined, "\nSUBFLOW_TRANSLATE_BASE_URL=https://example.com/v1\n") == false {
		t.Fatalf("missing SUBFLOW_TRANSLATE_BASE_URL in env: %s", joined)
	}
	if strings.Contains(joined, "\nSUBFLOW_TRANSLATE_API_KEY=secret-key\n") == false {
		t.Fatalf("missing SUBFLOW_TRANSLATE_API_KEY in env: %s", joined)
	}
	if strings.Contains(joined, "\nPYTHONPATH="+filepath.Join(root, "src")+string(os.PathListSeparator)+"C:\\existing\n") == false {
		t.Fatalf("missing merged PYTHONPATH in env: %s", joined)
	}
}

func TestResolvePythonExecutableFallsBackToEnv(t *testing.T) {
	pythonExe := filepath.Join(t.TempDir(), "python3")
	if err := os.WriteFile(pythonExe, []byte(""), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}
	t.Setenv("CSF_DDDDOCR_PYTHON", pythonExe)

	got := resolvePythonExecutable("")
	if got != pythonExe {
		t.Fatalf("resolvePythonExecutable() = %q, want %q", got, pythonExe)
	}
}

func makeFakeSubflowRoot(t *testing.T) string {
	t.Helper()

	root := t.TempDir()
	translateJobPath := filepath.Join(root, "src", "subflow", "translate_job.py")
	if err := os.MkdirAll(filepath.Dir(translateJobPath), 0o755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}
	if err := os.WriteFile(translateJobPath, []byte("print('ok')\n"), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	return root
}
