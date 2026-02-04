package project

import (
	"crypto/sha256"
	"encoding/hex"
	"os/exec"
	"path/filepath"
	"strings"
)

type Scope string

const (
	ScopeRepo Scope = "repo"
	ScopeDir  Scope = "dir"
)

type Info struct {
	ID        string
	Root      string
	GitRoot   string
	GitRemote string
}

func Detect(scope Scope, cwd string) (Info, error) {
	root := cwd
	gitRoot, gitRemote := "", ""
	if scope == ScopeRepo {
		if gr, err := gitTopLevel(cwd); err == nil && gr != "" {
			root = gr
			gitRoot = gr
			gitRemote = gitOriginURL(gr)
		}
	}
	root = filepath.Clean(root)
	id := stableID(root, gitRemote)
	return Info{ID: id, Root: root, GitRoot: gitRoot, GitRemote: gitRemote}, nil
}

func stableID(root, remote string) string {
	seed := root
	if remote != "" {
		seed = seed + "|" + remote
	}
	h := sha256.Sum256([]byte(seed))
	return hex.EncodeToString(h[:8])
}

func gitTopLevel(dir string) (string, error) {
	cmd := exec.Command("git", "-C", dir, "rev-parse", "--show-toplevel")
	out, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(out)), nil
}

func gitOriginURL(dir string) string {
	cmd := exec.Command("git", "-C", dir, "config", "--get", "remote.origin.url")
	out, err := cmd.Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}

func ContainerName(projectID string) string {
	return "codexbox-" + projectID
}
