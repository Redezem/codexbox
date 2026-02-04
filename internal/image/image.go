package image

import (
	"codexbox/internal/docker"
)

func Build(engine docker.Engine, tag, contextDir string) error {
	return engine.BuildImage(tag, contextDir, false)
}

func Update(engine docker.Engine, tag, contextDir string) error {
	return engine.BuildImage(tag, contextDir, true)
}
