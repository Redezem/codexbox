package image

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestLaunchScriptConfigBootstrap(t *testing.T) {
	t.Run("creates notify config when missing", func(t *testing.T) {
		dir := t.TempDir()
		scriptPath := writeLaunchScript(t, dir)
		peonDir := filepath.Join(dir, "peon")
		peonShPath := filepath.Join(peonDir, "peon.sh")
		adapterPath := filepath.Join(peonDir, "adapters", "codex.sh")
		if err := os.MkdirAll(peonDir, 0o755); err != nil {
			t.Fatalf("mkdir peon dir: %v", err)
		}
		if err := os.WriteFile(peonShPath, []byte("#!/bin/sh\n"), 0o755); err != nil {
			t.Fatalf("write peon.sh: %v", err)
		}
		if err := os.MkdirAll(filepath.Dir(adapterPath), 0o755); err != nil {
			t.Fatalf("mkdir adapters: %v", err)
		}
		if err := os.WriteFile(adapterPath, []byte("#!/bin/sh\n"), 0o755); err != nil {
			t.Fatalf("write adapter: %v", err)
		}

		codexHome := filepath.Join(dir, "codex-home")
		runLaunchScript(t, scriptPath, codexHome, peonDir)

		configPath := filepath.Join(codexHome, "config.toml")
		content, err := os.ReadFile(configPath)
		if err != nil {
			t.Fatalf("read config: %v", err)
		}
		want := `notify = ["bash", "` + adapterPath + `"]`
		if strings.TrimSpace(string(content)) != want {
			t.Fatalf("unexpected config: %q", string(content))
		}
	})

	t.Run("replaces existing notify entry", func(t *testing.T) {
		dir := t.TempDir()
		scriptPath := writeLaunchScript(t, dir)
		peonDir := filepath.Join(dir, "peon")
		peonShPath := filepath.Join(peonDir, "peon.sh")
		adapterPath := filepath.Join(peonDir, "adapters", "codex.sh")
		if err := os.MkdirAll(peonDir, 0o755); err != nil {
			t.Fatalf("mkdir peon dir: %v", err)
		}
		if err := os.WriteFile(peonShPath, []byte("#!/bin/sh\n"), 0o755); err != nil {
			t.Fatalf("write peon.sh: %v", err)
		}
		if err := os.MkdirAll(filepath.Dir(adapterPath), 0o755); err != nil {
			t.Fatalf("mkdir adapters: %v", err)
		}
		if err := os.WriteFile(adapterPath, []byte("#!/bin/sh\n"), 0o755); err != nil {
			t.Fatalf("write adapter: %v", err)
		}

		codexHome := filepath.Join(dir, "codex-home")
		if err := os.MkdirAll(codexHome, 0o755); err != nil {
			t.Fatalf("mkdir codex home: %v", err)
		}
		configPath := filepath.Join(codexHome, "config.toml")
		initial := "model = \"gpt-5\"\nnotify = [\"bash\", \"/tmp/old-hook.sh\"]\n"
		if err := os.WriteFile(configPath, []byte(initial), 0o644); err != nil {
			t.Fatalf("write config: %v", err)
		}

		runLaunchScript(t, scriptPath, codexHome, peonDir)

		content, err := os.ReadFile(configPath)
		if err != nil {
			t.Fatalf("read config: %v", err)
		}
		got := string(content)
		if !strings.Contains(got, `model = "gpt-5"`) {
			t.Fatalf("config lost existing content: %q", got)
		}
		if !strings.Contains(got, `notify = ["bash", "`+adapterPath+`"]`) {
			t.Fatalf("config missing peon notify entry: %q", got)
		}
		if strings.Contains(got, "/tmp/old-hook.sh") {
			t.Fatalf("old notify entry still present: %q", got)
		}
	})

	t.Run("moves injected notify before first table", func(t *testing.T) {
		dir := t.TempDir()
		scriptPath := writeLaunchScript(t, dir)
		peonDir := filepath.Join(dir, "peon")
		peonShPath := filepath.Join(peonDir, "peon.sh")
		adapterPath := filepath.Join(peonDir, "adapters", "codex.sh")
		if err := os.MkdirAll(peonDir, 0o755); err != nil {
			t.Fatalf("mkdir peon dir: %v", err)
		}
		if err := os.WriteFile(peonShPath, []byte("#!/bin/sh\n"), 0o755); err != nil {
			t.Fatalf("write peon.sh: %v", err)
		}
		if err := os.MkdirAll(filepath.Dir(adapterPath), 0o755); err != nil {
			t.Fatalf("mkdir adapters: %v", err)
		}
		if err := os.WriteFile(adapterPath, []byte("#!/bin/sh\n"), 0o755); err != nil {
			t.Fatalf("write adapter: %v", err)
		}

		codexHome := filepath.Join(dir, "codex-home")
		if err := os.MkdirAll(codexHome, 0o755); err != nil {
			t.Fatalf("mkdir codex home: %v", err)
		}
		configPath := filepath.Join(codexHome, "config.toml")
		initial := "[notice.model_migrations]\nnotify = \"stay-string\"\n"
		if err := os.WriteFile(configPath, []byte(initial), 0o644); err != nil {
			t.Fatalf("write config: %v", err)
		}

		runLaunchScript(t, scriptPath, codexHome, peonDir)

		content, err := os.ReadFile(configPath)
		if err != nil {
			t.Fatalf("read config: %v", err)
		}
		got := string(content)
		wantPrefix := `notify = ["bash", "` + adapterPath + `"]` + "\n[notice.model_migrations]\n"
		if !strings.HasPrefix(got, wantPrefix) {
			t.Fatalf("notify should be inserted before first table: %q", got)
		}
		if !strings.Contains(got, `notify = "stay-string"`) {
			t.Fatalf("table notify should be preserved: %q", got)
		}
	})

	t.Run("removes previously appended broken notify entry", func(t *testing.T) {
		dir := t.TempDir()
		scriptPath := writeLaunchScript(t, dir)
		peonDir := filepath.Join(dir, "peon")
		peonShPath := filepath.Join(peonDir, "peon.sh")
		adapterPath := filepath.Join(peonDir, "adapters", "codex.sh")
		if err := os.MkdirAll(peonDir, 0o755); err != nil {
			t.Fatalf("mkdir peon dir: %v", err)
		}
		if err := os.WriteFile(peonShPath, []byte("#!/bin/sh\n"), 0o755); err != nil {
			t.Fatalf("write peon.sh: %v", err)
		}
		if err := os.MkdirAll(filepath.Dir(adapterPath), 0o755); err != nil {
			t.Fatalf("mkdir adapters: %v", err)
		}
		if err := os.WriteFile(adapterPath, []byte("#!/bin/sh\n"), 0o755); err != nil {
			t.Fatalf("write adapter: %v", err)
		}

		codexHome := filepath.Join(dir, "codex-home")
		if err := os.MkdirAll(codexHome, 0o755); err != nil {
			t.Fatalf("mkdir codex home: %v", err)
		}
		configPath := filepath.Join(codexHome, "config.toml")
		initial := "[notice.model_migrations]\nnotify = \"stay-string\"\nnotify = [\"bash\", \"" + adapterPath + "\"]\n"
		if err := os.WriteFile(configPath, []byte(initial), 0o644); err != nil {
			t.Fatalf("write config: %v", err)
		}

		runLaunchScript(t, scriptPath, codexHome, peonDir)

		content, err := os.ReadFile(configPath)
		if err != nil {
			t.Fatalf("read config: %v", err)
		}
		got := string(content)
		if strings.Count(got, `notify = ["bash", "`+adapterPath+`"]`) != 1 {
			t.Fatalf("expected exactly one top-level notify entry: %q", got)
		}
		if strings.Contains(got, "[notice.model_migrations]\nnotify = \"stay-string\"\nnotify = [\"bash\", \""+adapterPath+"\"]") {
			t.Fatalf("broken nested notify entry still present: %q", got)
		}
	})

	t.Run("warns when peon install is incomplete", func(t *testing.T) {
		dir := t.TempDir()
		scriptPath := writeLaunchScript(t, dir)
		peonDir := filepath.Join(dir, "peon")
		if err := os.MkdirAll(peonDir, 0o755); err != nil {
			t.Fatalf("mkdir peon dir: %v", err)
		}

		codexHome := filepath.Join(dir, "codex-home")
		output := runLaunchScript(t, scriptPath, codexHome, peonDir)

		if !strings.Contains(output, "peon-ping is enabled but") {
			t.Fatalf("missing warning, got %q", output)
		}
		if _, err := os.Stat(filepath.Join(codexHome, "config.toml")); !os.IsNotExist(err) {
			t.Fatalf("config.toml should not be created when peon.sh is missing")
		}
	})
}

func writeLaunchScript(t *testing.T, dir string) string {
	t.Helper()

	scriptPath := filepath.Join(dir, "codexbox-launch.sh")
	if err := os.WriteFile(scriptPath, []byte(launchScript), 0o755); err != nil {
		t.Fatalf("write launch script: %v", err)
	}
	return scriptPath
}

func runLaunchScript(t *testing.T, scriptPath, codexHome, peonDir string) string {
	t.Helper()

	cmd := exec.Command("sh", scriptPath, "true")
	cmd.Env = append(os.Environ(),
		"CODEX_HOME="+codexHome,
		"CLAUDE_PEON_DIR="+peonDir,
		"HOME="+filepath.Dir(codexHome),
	)
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("launch script failed: %v\n%s", err, output)
	}
	return string(output)
}
