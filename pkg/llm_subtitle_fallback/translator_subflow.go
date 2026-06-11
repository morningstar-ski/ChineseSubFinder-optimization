package llm_subtitle_fallback

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

type subflowTranslator struct{}

func (subflowTranslator) Translate(req TranslateRequest) error {
	if req.SubflowRootDir == "" {
		return fmt.Errorf("subflow root dir is empty")
	}
	if _, err := os.Stat(req.SubflowRootDir); err != nil {
		return fmt.Errorf("subflow root dir invalid: %w", err)
	}

	pythonExecutable := req.PythonExecutable
	if pythonExecutable == "" {
		pythonExecutable = defaultPythonExe
	}

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
	cmd.Dir = req.SubflowRootDir
	cmd.Env = buildPythonPathEnv(req.SubflowRootDir)

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

func buildPythonPathEnv(subflowRootDir string) []string {
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
		return env
	}

	return append(env, "PYTHONPATH="+srcDir)
}
