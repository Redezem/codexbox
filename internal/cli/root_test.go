package cli

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"codexbox/internal/project"
)

func TestBuildExecCommand(t *testing.T) {
	t.Run("default uses launch wrapper", func(t *testing.T) {
		got := buildExecCommand(options{})
		want := []string{launchScriptPath, "codex", "--dangerously-bypass-approvals-and-sandbox"}
		if len(got) != len(want) {
			t.Fatalf("len mismatch: got %d want %d", len(got), len(want))
		}
		for i := range want {
			if got[i] != want[i] {
				t.Fatalf("arg %d mismatch: got %q want %q", i, got[i], want[i])
			}
		}
	})

	t.Run("shell bypasses wrapper", func(t *testing.T) {
		got := buildExecCommand(options{Shell: true})
		if len(got) != 1 || got[0] != "bash" {
			t.Fatalf("unexpected shell command: %#v", got)
		}
	})

	t.Run("cmd bypasses wrapper", func(t *testing.T) {
		got := buildExecCommand(options{Cmd: "echo hi"})
		want := []string{"sh", "-lc", "echo hi"}
		if len(got) != len(want) {
			t.Fatalf("len mismatch: got %d want %d", len(got), len(want))
		}
		for i := range want {
			if got[i] != want[i] {
				t.Fatalf("arg %d mismatch: got %q want %q", i, got[i], want[i])
			}
		}
	})
}

func TestContainerEnvIncludesRuntimeFlags(t *testing.T) {
	env := containerEnv()

	if env["REMOTE_CONTAINERS"] != "true" {
		t.Fatalf("REMOTE_CONTAINERS = %q", env["REMOTE_CONTAINERS"])
	}
	if env["CODEXBOX"] != "true" {
		t.Fatalf("CODEXBOX = %q", env["CODEXBOX"])
	}
	if env["CLAUDE_PEON_DIR"] != containerPeonDir {
		t.Fatalf("CLAUDE_PEON_DIR = %q", env["CLAUDE_PEON_DIR"])
	}
	if env["CODEX_HOME"] != containerCodexHome {
		t.Fatalf("CODEX_HOME = %q", env["CODEX_HOME"])
	}
}

func TestContainerEnvIncludesOptionalPeonMobilePushoverVars(t *testing.T) {
	t.Setenv(peonPushoverUserKeyEnvVar, "user-key")
	t.Setenv(peonPushoverAppTokenEnvVar, "app-token")

	env := containerEnv()

	if env[peonPushoverUserKeyEnvVar] != "user-key" {
		t.Fatalf("%s = %q", peonPushoverUserKeyEnvVar, env[peonPushoverUserKeyEnvVar])
	}
	if env[peonPushoverAppTokenEnvVar] != "app-token" {
		t.Fatalf("%s = %q", peonPushoverAppTokenEnvVar, env[peonPushoverAppTokenEnvVar])
	}
}

func TestContainerMountsIncludeDockerSocketWhenRequested(t *testing.T) {
	info := project.Info{ID: "proj-123", Root: "/workspace-src"}
	mounts := containerMounts(info, "/home/tester", false, true)

	foundSocket := false
	for _, m := range mounts {
		if m.Source == hostDockerSocketPath && m.Target == hostDockerSocketPath {
			foundSocket = true
			break
		}
	}
	if !foundSocket {
		t.Fatalf("expected docker socket mount in %#v", mounts)
	}
}

func TestContainerMountsOmitDockerSocketWhenUnavailable(t *testing.T) {
	info := project.Info{ID: "proj-123", Root: "/workspace-src"}
	mounts := containerMounts(info, "/home/tester", false, false)

	for _, m := range mounts {
		if m.Source == hostDockerSocketPath || m.Target == hostDockerSocketPath {
			t.Fatalf("did not expect docker socket mount in %#v", mounts)
		}
	}
}

func TestPathExists(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "exists.txt")
	if err := os.WriteFile(file, []byte("x"), 0o644); err != nil {
		t.Fatalf("write test file: %v", err)
	}

	if !pathExists(file) {
		t.Fatalf("expected existing path to return true")
	}
	if pathExists(filepath.Join(dir, "missing.txt")) {
		t.Fatalf("expected missing path to return false")
	}
}

func TestImageIDsDiffer(t *testing.T) {
	tests := []struct {
		name          string
		containerID   string
		latestImageID string
		want          bool
	}{
		{name: "different ids", containerID: "sha256:old", latestImageID: "sha256:new", want: true},
		{name: "same ids", containerID: "sha256:same", latestImageID: "sha256:same", want: false},
		{name: "missing container id", containerID: "", latestImageID: "sha256:new", want: false},
		{name: "missing latest id", containerID: "sha256:old", latestImageID: "", want: false},
		{name: "trims whitespace", containerID: " sha256:old ", latestImageID: "\tsha256:new\n", want: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := imageIDsDiffer(tt.containerID, tt.latestImageID); got != tt.want {
				t.Fatalf("imageIDsDiffer(%q, %q) = %v, want %v", tt.containerID, tt.latestImageID, got, tt.want)
			}
		})
	}
}

func TestParsePromptDefaultYes(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  bool
		ok    bool
	}{
		{name: "empty defaults yes", input: "", want: true, ok: true},
		{name: "whitespace defaults yes", input: "   ", want: true, ok: true},
		{name: "lower y", input: "y", want: true, ok: true},
		{name: "upper yes", input: "YES", want: true, ok: true},
		{name: "lower n", input: "n", want: false, ok: true},
		{name: "mixed no", input: "No", want: false, ok: true},
		{name: "invalid", input: "maybe", want: false, ok: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := parsePromptDefaultYes(tt.input)
			if got != tt.want || ok != tt.ok {
				t.Fatalf("parsePromptDefaultYes(%q) = (%v, %v), want (%v, %v)", tt.input, got, ok, tt.want, tt.ok)
			}
		})
	}
}

func TestPromptForImageRebase(t *testing.T) {
	tests := []struct {
		name          string
		input         string
		want          bool
		wantPrompts   int
		wantErrSubstr string
	}{
		{name: "enter defaults yes", input: "\n", want: true, wantPrompts: 1},
		{name: "explicit yes", input: "y\n", want: true, wantPrompts: 1},
		{name: "explicit no", input: "n\n", want: false, wantPrompts: 1},
		{name: "invalid then no", input: "maybe\nn\n", want: false, wantPrompts: 2},
		{name: "invalid eof errors", input: "maybe", wantPrompts: 1, wantErrSubstr: "invalid prompt response"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var out bytes.Buffer
			got, err := promptForImageRebase(strings.NewReader(tt.input), &out)
			if tt.wantErrSubstr != "" {
				if err == nil || !strings.Contains(err.Error(), tt.wantErrSubstr) {
					t.Fatalf("promptForImageRebase error = %v, want substring %q", err, tt.wantErrSubstr)
				}
				return
			}
			if err != nil {
				t.Fatalf("promptForImageRebase error = %v", err)
			}
			if got != tt.want {
				t.Fatalf("promptForImageRebase(%q) = %v, want %v", tt.input, got, tt.want)
			}
			if prompts := strings.Count(out.String(), "A new Codexbox image exists, do you want to rebase? (Y/n) "); prompts != tt.wantPrompts {
				t.Fatalf("prompt count = %d, want %d; output=%q", prompts, tt.wantPrompts, out.String())
			}
		})
	}
}

func TestIsInteractiveSession(t *testing.T) {
	if isInteractiveSession(strings.NewReader(""), &bytes.Buffer{}) {
		t.Fatalf("expected non-file streams to be treated as non-interactive")
	}
}
