package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestParseRemoteURL_SSH(t *testing.T) {
	ws, slug, ok := ParseRemoteURL("git@bitbucket.org:myworkspace/myrepo.git")
	if !ok || ws != "myworkspace" || slug != "myrepo" {
		t.Errorf("SSH = (%q, %q, %v)", ws, slug, ok)
	}
}

func TestParseRemoteURL_HTTPS(t *testing.T) {
	ws, slug, ok := ParseRemoteURL("https://bitbucket.org/myworkspace/myrepo")
	if !ok || ws != "myworkspace" || slug != "myrepo" {
		t.Errorf("HTTPS = (%q, %q, %v)", ws, slug, ok)
	}
}

func TestParseRemoteURL_HTTPSWithGit(t *testing.T) {
	ws, slug, ok := ParseRemoteURL("https://bitbucket.org/ws/repo.git")
	if !ok || ws != "ws" || slug != "repo" {
		t.Errorf("HTTPS .git = (%q, %q, %v)", ws, slug, ok)
	}
}

func TestParseRemoteURL_Invalid(t *testing.T) {
	cases := []string{
		"https://github.com/user/repo",
		"git@github.com:user/repo.git",
		"not-a-url",
		"git@bitbucket.org:onlyone",
	}
	for _, c := range cases {
		_, _, ok := ParseRemoteURL(c)
		if ok {
			t.Errorf("ParseRemoteURL(%q) should return false", c)
		}
	}
}

func TestResolveRepo_ExplicitRepoFlag(t *testing.T) {
	ws, slug, err := ResolveRepo("", "acme/project", nil)
	if err != nil || ws != "acme" || slug != "project" {
		t.Errorf("explicit flag = (%q, %q, %v)", ws, slug, err)
	}
}

func TestResolveRepo_SlugWithWSFlag(t *testing.T) {
	ws, slug, err := ResolveRepo("myws", "myrepo", nil)
	if err != nil || ws != "myws" || slug != "myrepo" {
		t.Errorf("slug+ws = (%q, %q, %v)", ws, slug, err)
	}
}

func TestResolveRepo_FallbackToConfig(t *testing.T) {
	cfg := &Config{Workspace: "cfgws"}
	// No flags, no git — only ws from config, slug still missing → error
	_, _, err := ResolveRepo("", "", cfg)
	if err == nil {
		t.Error("expected error when slug cannot be determined")
	}
}

func TestResolveRepo_ErrorWhenEmpty(t *testing.T) {
	_, _, err := ResolveRepo("", "", nil)
	if err == nil {
		t.Error("expected error with no flags and no config")
	}
}

func TestSaveAndLoad(t *testing.T) {
	// Use temp dir to avoid touching real config
	tmpDir := t.TempDir()
	origHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpDir)
	defer os.Setenv("HOME", origHome)

	// Ensure config dir matches what configDir() will compute
	dir := filepath.Join(tmpDir, ".config", "bitb")
	os.MkdirAll(dir, 0700)

	err := Save("testws", "user@test.com", "secret-token")
	if err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if cfg.Workspace != "testws" || cfg.Email != "user@test.com" || cfg.Token != "secret-token" {
		t.Errorf("Load() = %+v", cfg)
	}
}

func TestRemove_NotExist(t *testing.T) {
	tmpDir := t.TempDir()
	origHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpDir)
	defer os.Setenv("HOME", origHome)

	err := Remove()
	if err == nil {
		t.Skip("Remove on non-existent file may succeed on some OS")
	}
	if !os.IsNotExist(err) {
		t.Errorf("Remove() = %v, want os.IsNotExist", err)
	}
}
