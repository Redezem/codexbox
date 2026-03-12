package cli

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"codexbox/internal/docker"
	"codexbox/internal/image"
	"codexbox/internal/lock"
	"codexbox/internal/project"
	"codexbox/internal/registry"

	"github.com/spf13/cobra"
)

type options struct {
	Engine       string
	ImageTag     string
	ProjectScope string
	Shell        bool
	Cmd          string
	Fresh        bool
	Readonly     bool
	Cpus         string
	Memory       string
	EnvFile      string
	NoGPU        bool
}

const defaultImageTag = "codexbox:latest"

const (
	containerCodexHome         = "/root/.codex"
	containerPeonDir           = "/usr/local/share/claude/hooks/peon-ping"
	launchScriptPath           = "/usr/local/bin/codexbox-launch"
	peonPushoverUserKeyEnvVar  = "PEON_MOBILE_PUSHOVER_USER_KEY"
	peonPushoverAppTokenEnvVar = "PEON_MOBILE_PUSHOVER_APP_TOKEN"
)

func Execute() error {
	root := newRootCmd()
	return root.Execute()
}

func newRootCmd() *cobra.Command {
	opts := options{}
	cmd := &cobra.Command{
		Use:   "codexbox",
		Short: "Run or resume a Codex sandbox for the current project.",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runDefault(cmd, opts)
		},
	}

	cmd.PersistentFlags().StringVar(&opts.Engine, "engine", "docker", "docker|podman")
	cmd.PersistentFlags().StringVar(&opts.ImageTag, "image", defaultImageTag, "image tag")
	cmd.PersistentFlags().StringVar(&opts.ProjectScope, "project-scope", "repo", "repo|dir")
	cmd.PersistentFlags().BoolVar(&opts.Shell, "shell", false, "start a shell instead of Codex")
	cmd.PersistentFlags().StringVar(&opts.Cmd, "cmd", "", "command to run in the container")
	cmd.PersistentFlags().BoolVar(&opts.Fresh, "fresh", false, "recreate the container")
	cmd.PersistentFlags().BoolVar(&opts.Readonly, "readonly", false, "mount workspace read-only")
	cmd.PersistentFlags().StringVar(&opts.Cpus, "cpus", "", "cpu limit")
	cmd.PersistentFlags().StringVar(&opts.Memory, "memory", "", "memory limit")
	cmd.PersistentFlags().StringVar(&opts.EnvFile, "env-file", "", "env file")
	cmd.PersistentFlags().BoolVar(&opts.NoGPU, "no-gpu", false, "disable GPU")

	cmd.AddCommand(newListCmd(opts))
	cmd.AddCommand(newRmCmd(opts))
	cmd.AddCommand(newRebaseCmd(opts))
	cmd.AddCommand(newInitCmd(opts))
	cmd.AddCommand(newDoctorCmd(opts))
	cmd.AddCommand(newConfigCmd(opts))
	cmd.AddCommand(newStatusCmd(opts))
	cmd.AddCommand(newImageCmd(opts))

	return cmd
}

func newStatusCmd(opts options) *cobra.Command {
	return &cobra.Command{
		Use:   "status",
		Short: "Show status of the current project's container",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			engine, err := ensureEngine(opts.Engine)
			if err != nil {
				return err
			}
			cwd, err := os.Getwd()
			if err != nil {
				return err
			}
			pscope := project.Scope(opts.ProjectScope)
			if pscope != project.ScopeRepo && pscope != project.ScopeDir {
				return fmt.Errorf("invalid --project-scope: %s", opts.ProjectScope)
			}
			info, err := project.Detect(pscope, cwd)
			if err != nil {
				return err
			}
			status, err := engine.ContainerStatus(project.ContainerName(info.ID))
			if err != nil {
				return err
			}
			fmt.Fprintln(cmd.OutOrStdout(), status)
			return nil
		},
	}
}

func ensureEngine(name string) (docker.Engine, error) {
	if name != "docker" && name != "podman" {
		return docker.Engine{}, fmt.Errorf("unsupported engine: %s", name)
	}
	return docker.Engine{Binary: name}, nil
}

func withRegistryFileLock(regPath string, fn func() error) error {
	lockPath := regPath + ".lock"
	l, err := lock.Acquire(lockPath)
	if err != nil {
		return err
	}
	defer func() {
		_ = l.Release()
	}()
	return fn()
}

func runDefault(cmd *cobra.Command, opts options) error {
	if opts.Shell && opts.Cmd != "" {
		return errors.New("cannot use --shell and --cmd together")
	}
	engine, err := ensureEngine(opts.Engine)
	if err != nil {
		return err
	}
	imageExists, err := engine.ImageExists(opts.ImageTag)
	if err != nil {
		return err
	}
	if !imageExists {
		return fmt.Errorf("image not found: %s (run `codexbox image build` first)", opts.ImageTag)
	}
	cwd, err := os.Getwd()
	if err != nil {
		return err
	}
	pscope := project.Scope(opts.ProjectScope)
	if pscope != project.ScopeRepo && pscope != project.ScopeDir {
		return fmt.Errorf("invalid --project-scope: %s", opts.ProjectScope)
	}
	info, err := project.Detect(pscope, cwd)
	if err != nil {
		return err
	}
	containerName := project.ContainerName(info.ID)

	regPath, err := registry.DefaultPath()
	if err != nil {
		return err
	}
	lockPath := regPath + ".lock"
	l, err := lock.Acquire(lockPath)
	if err != nil {
		return err
	}
	defer func() {
		if l != nil {
			_ = l.Release()
		}
	}()

	reg, err := registry.Load(regPath)
	if err != nil {
		return err
	}
	if reg.Entries == nil {
		reg.Entries = map[string]registry.Entry{}
	}

	exists, err := engine.ContainerExists(containerName)
	if err != nil {
		return err
	}
	if opts.Fresh && exists {
		if err := engine.RemoveContainer(containerName); err != nil {
			return err
		}
		exists = false
	}

	if !exists {
		entry, err := createContainer(engine, opts, info)
		if err != nil {
			return err
		}
		reg.Entries[info.ID] = entry
		if err := registry.Save(regPath, reg); err != nil {
			return err
		}
	}

	if err := engine.StartContainer(containerName); err != nil {
		return err
	}
	now := time.Now().UTC()
	entry, ok := reg.Entries[info.ID]
	if !ok {
		entry = registry.Entry{
			ID:        info.ID,
			Path:      info.Root,
			ImageTag:  opts.ImageTag,
			CreatedAt: now,
		}
	}
	entry.LastUsed = now
	reg.Entries[info.ID] = entry
	if err := registry.Save(regPath, reg); err != nil {
		return err
	}
	if err := l.Release(); err != nil {
		return err
	}
	l = nil

	execCmd := buildExecCommand(opts)

	if err := engine.ExecInteractive(containerName, containerEnv(), execCmd); err != nil {
		_ = engine.StopContainer(containerName)
		return err
	}
	return engine.StopContainer(containerName)
}

func buildExecCommand(opts options) []string {
	switch {
	case opts.Shell:
		return []string{"bash"}
	case opts.Cmd != "":
		return []string{"sh", "-lc", opts.Cmd}
	default:
		return []string{launchScriptPath, "codex", "--dangerously-bypass-approvals-and-sandbox"}
	}
}

func createContainer(engine docker.Engine, opts options, info project.Info) (registry.Entry, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return registry.Entry{}, err
	}
	mounts := []docker.Mount{
		{Source: info.Root, Target: "/workspace", Readonly: opts.Readonly},
		{Source: project.VolumeName(info.ID, "go"), Target: "/go/pkg/mod"},
		{Source: project.VolumeName(info.ID, "cargo"), Target: "/root/.cargo"},
		{Source: project.VolumeName(info.ID, "npm"), Target: "/root/.npm"},
		{Source: project.VolumeName(info.ID, "pip"), Target: "/root/.cache/pip"},
		{Source: filepath.Join(home, ".codex"), Target: "/root/.codex"},
	}
	labels := map[string]string{
		"com.codexbox.project_id": info.ID,
		"com.codexbox.path":       info.Root,
		"com.codexbox.image_tag":  opts.ImageTag,
		"com.codexbox.created_at": time.Now().UTC().Format(time.RFC3339),
	}
	env := containerEnv()
	create := docker.CreateOpts{
		Name:     project.ContainerName(info.ID),
		Image:    opts.ImageTag,
		Workdir:  "/workspace",
		Mounts:   mounts,
		Labels:   labels,
		Env:      env,
		EnvFile:  opts.EnvFile,
		Cpus:     opts.Cpus,
		Memory:   opts.Memory,
		Readonly: opts.Readonly,
		Init:     true,
		Cmd:      []string{"sleep", "infinity"},
	}
	if err := engine.CreateContainer(create); err != nil {
		return registry.Entry{}, err
	}
	return registry.Entry{
		ID:        info.ID,
		Path:      info.Root,
		ImageTag:  opts.ImageTag,
		CreatedAt: time.Now().UTC(),
		LastUsed:  time.Now().UTC(),
	}, nil
}

func containerEnv() map[string]string {
	return map[string]string{
		"OPENAI_API_KEY":           os.Getenv("OPENAI_API_KEY"),
		"OPENAI_BASE_URL":          os.Getenv("OPENAI_BASE_URL"),
		peonPushoverUserKeyEnvVar:  os.Getenv(peonPushoverUserKeyEnvVar),
		peonPushoverAppTokenEnvVar: os.Getenv(peonPushoverAppTokenEnvVar),
		"REMOTE_CONTAINERS":        "true",
		"CODEXBOX":                 "true",
		"CLAUDE_PEON_DIR":          containerPeonDir,
		"CODEX_HOME":               containerCodexHome,
	}
}

func newListCmd(opts options) *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List project containers",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			engine, err := ensureEngine(opts.Engine)
			if err != nil {
				return err
			}
			regPath, err := registry.DefaultPath()
			if err != nil {
				return err
			}
			var reg registry.Registry
			if err := withRegistryFileLock(regPath, func() error {
				loaded, err := registry.Load(regPath)
				if err != nil {
					return err
				}
				reg = loaded
				return nil
			}); err != nil {
				return err
			}
			if len(reg.Entries) == 0 {
				fmt.Fprintln(cmd.OutOrStdout(), "no projects")
				return nil
			}
			fmt.Fprintln(cmd.OutOrStdout(), "id\tpath\timage\tstatus\tlast_used")
			for _, entry := range reg.Entries {
				status, err := engine.ContainerStatus(project.ContainerName(entry.ID))
				if err != nil {
					status = "unknown"
				}
				lastUsed := "-"
				if !entry.LastUsed.IsZero() {
					lastUsed = entry.LastUsed.UTC().Format(time.RFC3339)
				}
				fmt.Fprintf(cmd.OutOrStdout(), "%s\t%s\t%s\t%s\t%s\n", entry.ID, entry.Path, entry.ImageTag, status, lastUsed)
			}
			return nil
		},
	}
}

func newRmCmd(opts options) *cobra.Command {
	return &cobra.Command{
		Use:   "rm <project>",
		Short: "Remove a project container",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			engine, err := ensureEngine(opts.Engine)
			if err != nil {
				return err
			}
			id := strings.TrimSpace(args[0])
			name := project.ContainerName(id)
			_ = engine.RemoveContainer(name)
			regPath, err := registry.DefaultPath()
			if err != nil {
				return err
			}
			return withRegistryFileLock(regPath, func() error {
				reg, err := registry.Load(regPath)
				if err != nil {
					return err
				}
				delete(reg.Entries, id)
				return registry.Save(regPath, reg)
			})
		},
	}
}

func newRebaseCmd(opts options) *cobra.Command {
	return &cobra.Command{
		Use:   "rebase [project]",
		Short: "Recreate project container using latest image",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			engine, err := ensureEngine(opts.Engine)
			if err != nil {
				return err
			}
			id := ""
			if len(args) > 0 {
				id = strings.TrimSpace(args[0])
			} else {
				cwd, err := os.Getwd()
				if err != nil {
					return err
				}
				pscope := project.Scope(opts.ProjectScope)
				if pscope != project.ScopeRepo && pscope != project.ScopeDir {
					return fmt.Errorf("invalid --project-scope: %s", opts.ProjectScope)
				}
				info, err := project.Detect(pscope, cwd)
				if err != nil {
					return err
				}
				id = info.ID
			}
			name := project.ContainerName(id)
			_ = engine.RemoveContainer(name)

			regPath, err := registry.DefaultPath()
			if err != nil {
				return err
			}
			var entry registry.Entry
			if err := withRegistryFileLock(regPath, func() error {
				reg, err := registry.Load(regPath)
				if err != nil {
					return err
				}
				e, ok := reg.Entries[id]
				if !ok {
					return fmt.Errorf("project not found in registry: %s", id)
				}
				entry = e
				return nil
			}); err != nil {
				return err
			}

			info := project.Info{ID: id, Root: entry.Path}
			newEntry, err := createContainer(engine, opts, info)
			if err != nil {
				return err
			}
			newEntry.LastUsed = time.Now().UTC()
			if err := withRegistryFileLock(regPath, func() error {
				reg, err := registry.Load(regPath)
				if err != nil {
					return err
				}
				if reg.Entries == nil {
					reg.Entries = map[string]registry.Entry{}
				}
				reg.Entries[id] = newEntry
				return registry.Save(regPath, reg)
			}); err != nil {
				return err
			}
			fmt.Fprintln(cmd.OutOrStdout(), "rebased container", name)
			return nil
		},
	}
}

func newInitCmd(opts options) *cobra.Command {
	return &cobra.Command{
		Use:   "init",
		Short: "Add .codex to .gitignore",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			cwd, err := os.Getwd()
			if err != nil {
				return err
			}
			info, err := project.Detect(project.ScopeRepo, cwd)
			if err != nil {
				return err
			}
			root := info.Root
			path := filepath.Join(root, ".gitignore")
			content := ""
			if data, err := os.ReadFile(path); err == nil {
				content = string(data)
			}
			if strings.Contains(content, ".codex") {
				return nil
			}
			if content != "" && !strings.HasSuffix(content, "\n") {
				content += "\n"
			}
			content += ".codex\n"
			return os.WriteFile(path, []byte(content), 0o644)
		},
	}
}

func newDoctorCmd(opts options) *cobra.Command {
	return &cobra.Command{
		Use:   "doctor",
		Short: "Check docker + environment",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			engine, err := ensureEngine(opts.Engine)
			if err != nil {
				return err
			}
			fmt.Fprintln(cmd.OutOrStdout(), "checking engine:", engine.Binary)
			return engine.Version()
		},
	}
}

func newConfigCmd(opts options) *cobra.Command {
	return &cobra.Command{
		Use:   "config",
		Short: "Show configuration paths",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			regPath, err := registry.DefaultPath()
			if err != nil {
				return err
			}
			fmt.Fprintln(cmd.OutOrStdout(), "registry:", regPath)
			return nil
		},
	}
}

func newImageCmd(opts options) *cobra.Command {
	img := &cobra.Command{
		Use:   "image",
		Short: "Manage base image",
	}
	img.AddCommand(&cobra.Command{
		Use:   "build",
		Short: "Build base image",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			engine, err := ensureEngine(opts.Engine)
			if err != nil {
				return err
			}
			cwd, err := os.Getwd()
			if err != nil {
				return err
			}
			return image.Build(engine, opts.ImageTag, cwd)
		},
	})
	img.AddCommand(&cobra.Command{
		Use:   "update",
		Short: "Update and rebuild base image",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			engine, err := ensureEngine(opts.Engine)
			if err != nil {
				return err
			}
			cwd, err := os.Getwd()
			if err != nil {
				return err
			}
			return image.Update(engine, opts.ImageTag, cwd)
		},
	})
	return img
}
