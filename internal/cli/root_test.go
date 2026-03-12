package cli

import "testing"

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
