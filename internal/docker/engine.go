package docker

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strings"
)

type Engine struct {
	Binary string
}

type Mount struct {
	Source   string
	Target   string
	Readonly bool
}

type CreateOpts struct {
	Name     string
	Image    string
	Workdir  string
	Mounts   []Mount
	Labels   map[string]string
	Env      map[string]string
	EnvFile  string
	Cpus     string
	Memory   string
	Readonly bool
	Init     bool
	Cmd      []string
}

func (e Engine) run(args ...string) error {
	cmd := exec.Command(e.Binary, args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin
	return cmd.Run()
}

func (e Engine) runOutput(args ...string) (string, error) {
	cmd := exec.Command(e.Binary, args...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		if stderr.Len() > 0 {
			return "", fmt.Errorf("%w: %s", err, strings.TrimSpace(stderr.String()))
		}
		return "", err
	}
	return stdout.String(), nil
}

func (e Engine) Version() error {
	return e.run("version")
}

func (e Engine) ContainerExists(name string) (bool, error) {
	out, err := e.runOutput("ps", "-a", "--filter", "name=^/"+name+"$", "--format", "{{.Names}}")
	if err != nil {
		return false, err
	}
	for _, line := range strings.Split(out, "\n") {
		if strings.TrimSpace(line) == name {
			return true, nil
		}
	}
	return false, nil
}

func (e Engine) ContainerStatus(name string) (string, error) {
	out, err := e.runOutput("ps", "-a", "--filter", "name=^/"+name+"$", "--format", "{{.Status}}")
	if err != nil {
		return "", err
	}
	for _, line := range strings.Split(out, "\n") {
		status := strings.TrimSpace(line)
		if status != "" {
			return status, nil
		}
	}
	return "missing", nil
}

func (e Engine) ImageExists(tag string) (bool, error) {
	out, err := e.runOutput("image", "inspect", tag, "--format", "{{.Id}}")
	if err != nil {
		return false, nil
	}
	return strings.TrimSpace(out) != "", nil
}

func (e Engine) CreateContainer(opts CreateOpts) error {
	args := []string{"create", "--name", opts.Name}
	if opts.Init {
		args = append(args, "--init")
	}
	if opts.Workdir != "" {
		args = append(args, "-w", opts.Workdir)
	}
	if opts.Readonly {
		args = append(args, "--read-only")
	}
	if opts.Cpus != "" {
		args = append(args, "--cpus", opts.Cpus)
	}
	if opts.Memory != "" {
		args = append(args, "--memory", opts.Memory)
	}
	if opts.EnvFile != "" {
		args = append(args, "--env-file", opts.EnvFile)
	}
	for k, v := range opts.Env {
		if v == "" {
			continue
		}
		args = append(args, "-e", k+"="+v)
	}
	for k, v := range opts.Labels {
		args = append(args, "--label", k+"="+v)
	}
	for _, m := range opts.Mounts {
		vol := m.Source + ":" + m.Target
		if m.Readonly {
			vol += ":ro"
		}
		args = append(args, "-v", vol)
	}
	if runtime.GOOS == "linux" {
		uid := os.Getuid()
		gid := os.Getgid()
		args = append(args, "--user", fmt.Sprintf("%d:%d", uid, gid))
	}
	args = append(args, opts.Image)
	args = append(args, opts.Cmd...)
	return e.run(args...)
}

func (e Engine) StartContainer(name string) error {
	return e.run("start", name)
}

func (e Engine) StopContainer(name string) error {
	return e.run("stop", name)
}

func (e Engine) RemoveContainer(name string) error {
	return e.run("rm", "-f", name)
}

func (e Engine) ExecInteractive(name string, cmd []string) error {
	args := append([]string{"exec", "-it", name}, cmd...)
	return e.run(args...)
}

func (e Engine) BuildImage(tag, contextDir string, pull bool) error {
	args := []string{"build", "-t", tag}
	if pull {
		args = append(args, "--pull")
	}
	args = append(args, contextDir)
	return e.run(args...)
}
