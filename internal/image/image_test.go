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
		runLaunchScript(t, scriptPath, codexHome, peonDir, nil)

		configPath := filepath.Join(codexHome, "config.toml")
		content, err := os.ReadFile(configPath)
		if err != nil {
			t.Fatalf("read config: %v", err)
		}
		want := `notify = ["bash", "` + filepath.Join(codexHome, "peon-ping", "adapters", "codex.sh") + `"]`
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

		runLaunchScript(t, scriptPath, codexHome, peonDir, nil)

		content, err := os.ReadFile(configPath)
		if err != nil {
			t.Fatalf("read config: %v", err)
		}
		got := string(content)
		runtimeAdapterPath := filepath.Join(codexHome, "peon-ping", "adapters", "codex.sh")
		if !strings.Contains(got, `model = "gpt-5"`) {
			t.Fatalf("config lost existing content: %q", got)
		}
		if !strings.Contains(got, `notify = ["bash", "`+runtimeAdapterPath+`"]`) {
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

		runLaunchScript(t, scriptPath, codexHome, peonDir, nil)

		content, err := os.ReadFile(configPath)
		if err != nil {
			t.Fatalf("read config: %v", err)
		}
		got := string(content)
		wantPrefix := `notify = ["bash", "` + filepath.Join(codexHome, "peon-ping", "adapters", "codex.sh") + `"]` + "\n[notice.model_migrations]\n"
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

		runLaunchScript(t, scriptPath, codexHome, peonDir, nil)

		content, err := os.ReadFile(configPath)
		if err != nil {
			t.Fatalf("read config: %v", err)
		}
		got := string(content)
		runtimeAdapterPath := filepath.Join(codexHome, "peon-ping", "adapters", "codex.sh")
		if strings.Count(got, `notify = ["bash", "`+runtimeAdapterPath+`"]`) != 1 {
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
		output := runLaunchScript(t, scriptPath, codexHome, peonDir, nil)

		if !strings.Contains(output, "peon-ping is enabled but") {
			t.Fatalf("missing warning, got %q", output)
		}
		if _, err := os.Stat(filepath.Join(codexHome, "config.toml")); !os.IsNotExist(err) {
			t.Fatalf("config.toml should not be created when peon.sh is missing")
		}
	})

	t.Run("configures pushover mobile notifications when env vars are set", func(t *testing.T) {
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
		installConfigPath := filepath.Join(peonDir, "config.json")
		initialConfig := `{"default_pack":"peasant"}`
		if err := os.WriteFile(installConfigPath, []byte(initialConfig), 0o644); err != nil {
			t.Fatalf("write peon config: %v", err)
		}

		codexHome := filepath.Join(dir, "codex-home")
		runLaunchScript(t, scriptPath, codexHome, peonDir, map[string]string{
			"PEON_MOBILE_PUSHOVER_USER_KEY":  "user-key",
			"PEON_MOBILE_PUSHOVER_APP_TOKEN": "app-token",
		})

		content, err := os.ReadFile(filepath.Join(codexHome, "peon-ping", "config.json"))
		if err != nil {
			t.Fatalf("read peon config: %v", err)
		}
		got := string(content)
		if !strings.Contains(got, `"default_pack": "peasant"`) {
			t.Fatalf("existing peon config should be preserved: %q", got)
		}
		if !strings.Contains(got, `"service": "pushover"`) {
			t.Fatalf("pushover service missing: %q", got)
		}
		if !strings.Contains(got, `"user_key": "user-key"`) {
			t.Fatalf("pushover user key missing: %q", got)
		}
		if !strings.Contains(got, `"app_token": "app-token"`) {
			t.Fatalf("pushover app token missing: %q", got)
		}
	})

	t.Run("warns when pushover mobile notification config is partial", func(t *testing.T) {
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
		output := runLaunchScript(t, scriptPath, codexHome, peonDir, map[string]string{
			"PEON_MOBILE_PUSHOVER_USER_KEY": "user-key",
		})

		if !strings.Contains(output, "Pushover mobile notifications require both") {
			t.Fatalf("missing partial config warning, got %q", output)
		}
		if _, err := os.Stat(filepath.Join(codexHome, "peon-ping", "config.json")); !os.IsNotExist(err) {
			t.Fatalf("config.json should not be created for partial pushover config")
		}
	})

	t.Run("creates runtime peon dir with staged assets", func(t *testing.T) {
		dir := t.TempDir()
		scriptPath := writeLaunchScript(t, dir)
		peonDir := filepath.Join(dir, "peon")
		peonShPath := filepath.Join(peonDir, "peon.sh")
		adapterPath := filepath.Join(peonDir, "adapters", "codex.sh")
		if err := os.MkdirAll(filepath.Dir(adapterPath), 0o755); err != nil {
			t.Fatalf("mkdir adapters: %v", err)
		}
		if err := os.MkdirAll(filepath.Join(peonDir, "packs"), 0o755); err != nil {
			t.Fatalf("mkdir packs: %v", err)
		}
		if err := os.MkdirAll(filepath.Join(peonDir, "scripts"), 0o755); err != nil {
			t.Fatalf("mkdir scripts: %v", err)
		}
		peonScript := `#!/bin/bash
set -uo pipefail

# --- Platform-aware audio playback ---
play_sound() {
  local file="$1" vol="$2"
  case "$PLATFORM" in
    devcontainer|ssh)
      local rel_path="${file#$PEON_DIR/}"
      local encoded_path
      encoded_path="$rel_path"
      ;;
  esac
}

exit 0
`
		if err := os.WriteFile(peonShPath, []byte(peonScript), 0o755); err != nil {
			t.Fatalf("write peon.sh: %v", err)
		}
		if err := os.WriteFile(adapterPath, []byte("#!/bin/sh\n"), 0o755); err != nil {
			t.Fatalf("write adapter: %v", err)
		}
		configPath := filepath.Join(peonDir, "config.json")
		if err := os.WriteFile(configPath, []byte("{\"default_pack\":\"peasant\"}\n"), 0o644); err != nil {
			t.Fatalf("write config: %v", err)
		}

		codexHome := filepath.Join(dir, "codex-home")
		runLaunchScript(t, scriptPath, codexHome, peonDir, map[string]string{
			"PEON_MOBILE_PUSHOVER_USER_KEY":  "user-key",
			"PEON_MOBILE_PUSHOVER_APP_TOKEN": "app-token",
		})

		runtimePeonDir := filepath.Join(codexHome, "peon-ping")
		runtimeConfig, err := os.ReadFile(filepath.Join(runtimePeonDir, "config.json"))
		if err != nil {
			t.Fatalf("read runtime config: %v", err)
		}
		got := string(runtimeConfig)
		if !strings.Contains(got, `"service": "pushover"`) {
			t.Fatalf("runtime config missing pushover service: %q", got)
		}
		adaptersInfo, err := os.Lstat(filepath.Join(runtimePeonDir, "adapters"))
		if err != nil {
			t.Fatalf("runtime adapters symlink missing: %v", err)
		}
		if adaptersInfo.Mode()&os.ModeSymlink == 0 {
			t.Fatalf("runtime adapters should be a symlink, mode=%v", adaptersInfo.Mode())
		}
		packsInfo, err := os.Lstat(filepath.Join(runtimePeonDir, "packs"))
		if err != nil {
			t.Fatalf("runtime packs symlink missing: %v", err)
		}
		if packsInfo.Mode()&os.ModeSymlink == 0 {
			t.Fatalf("runtime packs should be a symlink, mode=%v", packsInfo.Mode())
		}
		peonInfo, err := os.Lstat(filepath.Join(runtimePeonDir, "peon.sh"))
		if err != nil {
			t.Fatalf("runtime peon.sh missing: %v", err)
		}
		if peonInfo.Mode()&os.ModeSymlink != 0 {
			t.Fatalf("runtime peon.sh should be a regular file, mode=%v", peonInfo.Mode())
		}
		runtimePeon, err := os.ReadFile(filepath.Join(runtimePeonDir, "peon.sh"))
		if err != nil {
			t.Fatalf("read runtime peon.sh: %v", err)
		}
		runtimePeonText := string(runtimePeon)
		if !strings.Contains(runtimePeonText, "relay_relative_path() {") {
			t.Fatalf("runtime peon.sh missing relay path helper: %q", runtimePeonText)
		}
		if !strings.Contains(runtimePeonText, `rel_path="$(relay_relative_path "$file")"`) {
			t.Fatalf("runtime peon.sh missing relay path rewrite: %q", runtimePeonText)
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

func runLaunchScript(t *testing.T, scriptPath, codexHome, peonDir string, extraEnv map[string]string) string {
	t.Helper()

	cmd := exec.Command("sh", scriptPath, "true")
	env := make([]string, 0, len(os.Environ())+len(extraEnv)+3)
	for _, entry := range os.Environ() {
		if strings.HasPrefix(entry, "PEON_MOBILE_PUSHOVER_USER_KEY=") {
			continue
		}
		if strings.HasPrefix(entry, "PEON_MOBILE_PUSHOVER_APP_TOKEN=") {
			continue
		}
		env = append(env, entry)
	}
	cmd.Env = append(env,
		"CODEX_HOME="+codexHome,
		"CLAUDE_PEON_DIR="+peonDir,
		"HOME="+filepath.Dir(codexHome),
	)
	for key, value := range extraEnv {
		cmd.Env = append(cmd.Env, key+"="+value)
	}
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("launch script failed: %v\n%s", err, output)
	}
	return string(output)
}
