package config

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/spf13/viper"
)

const configFileName = "config"

type Config struct {
	Workspace string `mapstructure:"workspace"`
	Email     string `mapstructure:"email"`
	Token     string `mapstructure:"token"`
}

func configDir() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".config", "bitb")
}

func configPath() string {
	return filepath.Join(configDir(), "config.toml")
}

func Load() (*Config, error) {
	path := configPath()
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return nil, fmt.Errorf("not logged in — run: bb auth login")
	}

	v := viper.New()
	v.SetConfigFile(path)
	v.SetConfigType("toml")
	if err := v.ReadInConfig(); err != nil {
		return nil, fmt.Errorf("failed to read config: %w", err)
	}

	var cfg Config
	if err := v.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("failed to parse config: %w", err)
	}
	return &cfg, nil
}

func Save(workspace, email, token string) error {
	dir := configDir()
	if err := os.MkdirAll(dir, 0700); err != nil {
		return fmt.Errorf("failed to create config dir: %w", err)
	}

	path := configPath()
	content := fmt.Sprintf("workspace = %q\nemail    = %q\ntoken    = %q\n", workspace, email, token)
	if err := os.WriteFile(path, []byte(content), 0600); err != nil {
		return fmt.Errorf("failed to write config: %w", err)
	}
	return nil
}

func Remove() error {
	return os.Remove(configPath())
}

func ConfigPath() string {
	return configPath()
}

// DetectRepo parses the current git remote to extract workspace and repo slug.
// Supports SSH (git@bitbucket.org:ws/repo.git) and HTTPS (https://bitbucket.org/ws/repo).
func DetectRepo() (workspace, slug string, ok bool) {
	out, err := exec.Command("git", "remote", "get-url", "origin").Output()
	if err != nil {
		return "", "", false
	}
	return ParseRemoteURL(strings.TrimSpace(string(out)))
}

// ParseRemoteURL extracts workspace and slug from a Bitbucket remote URL.
func ParseRemoteURL(rawURL string) (workspace, slug string, ok bool) {
	var path string
	switch {
	case strings.HasPrefix(rawURL, "git@bitbucket.org:"):
		path = strings.TrimPrefix(rawURL, "git@bitbucket.org:")
		path = strings.TrimSuffix(path, ".git")
	case strings.Contains(rawURL, "bitbucket.org/"):
		parts := strings.SplitN(rawURL, "bitbucket.org/", 2)
		if len(parts) < 2 {
			return "", "", false
		}
		path = strings.TrimSuffix(parts[1], ".git")
	default:
		return "", "", false
	}

	parts := strings.SplitN(path, "/", 2)
	if len(parts) != 2 {
		return "", "", false
	}
	return parts[0], parts[1], true
}

// ResolveRepo resolves workspace and repo from flags, git remote, or config.
func ResolveRepo(wsFlag, repoFlag string, cfg *Config) (workspace, slug string, err error) {
	// repoFlag can be "workspace/slug" or just "slug"
	if repoFlag != "" && strings.Contains(repoFlag, "/") {
		parts := strings.SplitN(repoFlag, "/", 2)
		return parts[0], parts[1], nil
	}

	detectedWS, detectedSlug, detected := DetectRepo()

	workspace = wsFlag
	if workspace == "" {
		if detected {
			workspace = detectedWS
		} else if cfg != nil {
			workspace = cfg.Workspace
		}
	}

	slug = repoFlag
	if slug == "" {
		if detected {
			slug = detectedSlug
		}
	}

	if workspace == "" || slug == "" {
		return "", "", fmt.Errorf("could not determine repository — use --repo workspace/slug or run from a Bitbucket repo directory")
	}
	return workspace, slug, nil
}

// CurrentBranch returns the current git branch name.
func CurrentBranch() string {
	out, err := exec.Command("git", "branch", "--show-current").Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}
