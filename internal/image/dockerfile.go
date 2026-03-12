package image

import _ "embed"

//go:embed assets/Dockerfile
var dockerfile string

//go:embed assets/codexbox-launch.sh
var launchScript string
