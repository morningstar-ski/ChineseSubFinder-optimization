package llm_subtitle_fallback

import (
	"os"
	"path/filepath"
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
