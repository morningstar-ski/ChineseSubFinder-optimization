package llm_subtitle_fallback

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

type subflowTranslator struct{}

func (subflowTranslator) Translate(req TranslateRequest) error {
	subflowRootDir, err := resolveSubflowRootDir(req.SubflowRootDir)
	if err != nil {
		return err
	}

	pythonExecutable := resolvePythonExecutable(req.PythonExecutable)

	args := []string{
		"-m", "subflow.translate_job",
		"--input", req.InputPath,
		"--output", req.OutputPath,
		"--provider", req.Provider,
		"--target-language", req.TargetLanguage,
		"--json",
	}
	if req.Model != "" {
		args = append(args, "--model", req.Model)
	}
	if req.SourceLanguage != "" {
		args = append(args, "--source-language", req.SourceLanguage)
	}
	if req.Style != "" {
		args = append(args, "--style", req.Style)
	}

	cmd := exec.Command(pythonExecutable, args...)
	cmd.Dir = subflowRootDir
	cmd.Env = buildTranslateEnv(subflowRootDir, req)

	output, err := cmd.CombinedOutput()
	logPath := filepath.Join(req.TaskDir, "translate.stdout.log")
	_ = os.WriteFile(logPath, output, 0o644)
	if err != nil {
		return fmt.Errorf("run subflow translate: %w: %s", err, strings.TrimSpace(string(output)))
	}
	if _, err := os.Stat(req.OutputPath); err != nil {
		return fmt.Errorf("translated subtitle not produced: %w", err)
	}

	return nil
}

func resolveSubflowRootDir(configured string) (string, error) {
	candidates := make([]string, 0, 5)
	seen := make(map[string]struct{})

	appendCandidate := func(path string) {
		path = strings.TrimSpace(path)
		if path == "" {
			return
		}
		cleaned := filepath.Clean(filepath.FromSlash(path))
		if _, ok := seen[cleaned]; ok {
			return
		}
		seen[cleaned] = struct{}{}
		candidates = append(candidates, cleaned)
	}

	appendCandidate(configured)
	appendCandidate(os.Getenv("CSF_LLM_SUBTITLE_FALLBACK_SUBFLOW_ROOT"))
	appendCandidate("/opt/subflow")

	if _, thisFile, _, ok := runtime.Caller(0); ok {
		appendCandidate(filepath.Join(filepath.Dir(thisFile), "..", "..", "third_party", "subflow"))
	}
	if cwd, err := os.Getwd(); err == nil {
		appendCandidate(filepath.Join(cwd, "third_party", "subflow"))
	}

	for _, candidate := range candidates {
		if isValidSubflowRootDir(candidate) {
			return candidate, nil
		}
	}

	if len(candidates) == 0 {
		return "", fmt.Errorf("subflow root dir is empty")
	}
	return "", fmt.Errorf("subflow root dir invalid, tried: %s", strings.Join(candidates, ", "))
}

func resolvePythonExecutable(configured string) string {
	configured = strings.TrimSpace(configured)
	if configured != "" {
		return configured
	}

	for _, candidate := range []string{
		os.Getenv("CSF_LLM_SUBTITLE_FALLBACK_PYTHON"),
		os.Getenv("CSF_DDDDOCR_PYTHON"),
	} {
		candidate = strings.TrimSpace(candidate)
		if candidate == "" {
			continue
		}
		if info, err := os.Stat(filepath.Clean(filepath.FromSlash(candidate))); err == nil && info.IsDir() == false {
			return filepath.Clean(filepath.FromSlash(candidate))
		}
	}

	return defaultPythonExe
}

func isValidSubflowRootDir(root string) bool {
	info, err := os.Stat(root)
	if err != nil || info.IsDir() == false {
		return false
	}
	translateJobPath := filepath.Join(root, "src", "subflow", "translate_job.py")
	info, err = os.Stat(translateJobPath)
	return err == nil && info.IsDir() == false
}

func buildTranslateEnv(subflowRootDir string, req TranslateRequest) []string {
	env := os.Environ()
	srcDir := filepath.Join(subflowRootDir, "src")

	pythonPathIndex := -1
	for i, item := range env {
		if strings.HasPrefix(strings.ToUpper(item), "PYTHONPATH=") {
			pythonPathIndex = i
			break
		}
	}

	if pythonPathIndex >= 0 {
		current := strings.TrimPrefix(env[pythonPathIndex], env[pythonPathIndex][:len("PYTHONPATH=")])
		if current == "" {
			env[pythonPathIndex] = "PYTHONPATH=" + srcDir
		} else {
			env[pythonPathIndex] = "PYTHONPATH=" + srcDir + string(os.PathListSeparator) + current
		}
	} else {
		env = append(env, "PYTHONPATH="+srcDir)
	}

	env = appendEnvIfPresent(env, "SUBFLOW_TRANSLATE_API_KEY", req.APIKey)
	env = appendEnvIfPresent(env, "SUBFLOW_TRANSLATE_BASE_URL", req.BaseURL)
	return env
}

func appendEnvIfPresent(env []string, key string, value string) []string {
	value = strings.TrimSpace(value)
	if value == "" {
		return env
	}
	entry := key + "=" + value
	prefix := strings.ToUpper(key) + "="
	for i, item := range env {
		if strings.HasPrefix(strings.ToUpper(item), prefix) {
			env[i] = entry
			return env
		}
	}
	return append(env, entry)
}
