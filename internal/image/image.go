package image

import (
	"os"
	"path/filepath"

	"codexbox/internal/docker"
)

func Build(engine docker.Engine, tag, contextDir string) error {
	return buildInternal(engine, tag, false, false)
}

func Update(engine docker.Engine, tag, contextDir string) error {
	return buildInternal(engine, tag, true, true)
}

func buildInternal(engine docker.Engine, tag string, pull, noCache bool) error {
	tmpDir, err := os.MkdirTemp("", "codexbox-image-")
	if err != nil {
		return err
	}
	defer os.RemoveAll(tmpDir)

	dockerfilePath := filepath.Join(tmpDir, "Dockerfile")
	if err := os.WriteFile(dockerfilePath, []byte(dockerfile), 0o644); err != nil {
		return err
	}
	launchScriptPath := filepath.Join(tmpDir, "codexbox-launch.sh")
	if err := os.WriteFile(launchScriptPath, []byte(launchScript), 0o755); err != nil {
		return err
	}
	return engine.BuildImage(tag, tmpDir, pull, noCache)
}
