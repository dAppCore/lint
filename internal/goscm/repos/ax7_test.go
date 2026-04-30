package repos

import (
	stderrors "errors"
	"os"
	"path/filepath"

	core "dappco.re/go"
	coreio "dappco.re/go/io"
)

const (
	ax7TestReposApi34e3c4    = "repos/api"
	ax7TestReposApiGit64cb4f = "repos/api/.git"
	ax7TestReposYaml56a034   = "repos.yaml"
)

func ax7RegistryMedium(t *core.T) coreio.Medium {
	t.Helper()
	medium, err := coreio.NewSandboxed(t.TempDir())
	core.RequireNoError(t, err)
	return medium
}

func ax7RegistryYAML() string {
	return "version: 1\norg: test\nbase_path: repos\ndefaults:\n  ci: github\nrepos:\n  api:\n    type: service\n  docs:\n    type: docs\n"
}

func TestRepos_LoadRegistry_Good(t *core.T) {
	medium := ax7RegistryMedium(t)
	core.RequireNoError(t, medium.Write(ax7TestReposYaml56a034, ax7RegistryYAML()))
	registry, err := LoadRegistry(medium, ax7TestReposYaml56a034)
	core.AssertNoError(t, err)
	core.AssertEqual(t, "test", registry.Org)
	core.AssertLen(t, registry.Repos, 2)
}

func TestRepos_LoadRegistry_Bad(t *core.T) {
	medium := ax7RegistryMedium(t)
	registry, err := LoadRegistry(medium, "missing.yaml")
	core.AssertError(t, err)
	core.AssertNil(t, registry)
}

func TestRepos_LoadRegistry_Ugly(t *core.T) {
	medium := ax7RegistryMedium(t)
	core.RequireNoError(t, medium.Write(ax7TestReposYaml56a034, "version: 1\nbase_path: ~/Code\nrepos:\n  api:\n    path: custom/api\n"))
	registry, err := LoadRegistry(medium, ax7TestReposYaml56a034)
	core.AssertNoError(t, err)
	core.AssertContains(t, registry.Repos["api"].Path, filepath.Join("custom", "api"))
}

func TestRepos_FindRegistry_Good(t *core.T) {
	dir := t.TempDir()
	core.RequireNoError(t, os.WriteFile(filepath.Join(dir, ax7TestReposYaml56a034), []byte(ax7RegistryYAML()), 0o644))
	old, err := os.Getwd()
	core.RequireNoError(t, err)
	t.Cleanup(func() { core.RequireNoError(t, os.Chdir(old)) })
	core.RequireNoError(t, os.Chdir(dir))
	path, err := FindRegistry(coreio.Local)
	core.AssertNoError(t, err)
	core.AssertContains(t, path, ax7TestReposYaml56a034)
}

func TestRepos_FindRegistry_Bad(t *core.T) {
	dir := t.TempDir()
	t.Setenv("HOME", t.TempDir())
	old, err := os.Getwd()
	core.RequireNoError(t, err)
	t.Cleanup(func() { core.RequireNoError(t, os.Chdir(old)) })
	core.RequireNoError(t, os.Chdir(dir))
	path, err := FindRegistry(coreio.Local)
	core.AssertErrorIs(t, err, os.ErrNotExist)
	core.AssertEqual(t, "", path)
}

func TestRepos_FindRegistry_Ugly(t *core.T) {
	dir := t.TempDir()
	core.RequireNoError(t, os.MkdirAll(filepath.Join(dir, ".core"), 0o755))
	core.RequireNoError(t, os.WriteFile(filepath.Join(dir, ".core", ax7TestReposYaml56a034), []byte(ax7RegistryYAML()), 0o644))
	old, err := os.Getwd()
	core.RequireNoError(t, err)
	t.Cleanup(func() { core.RequireNoError(t, os.Chdir(old)) })
	core.RequireNoError(t, os.Chdir(dir))
	path, err := FindRegistry(coreio.Local)
	core.AssertNoError(t, err)
	core.AssertContains(t, path, filepath.Join(".core", ax7TestReposYaml56a034))
}

func TestRepos_Registry_List_Good(t *core.T) {
	registry := &Registry{Repos: map[string]*Repo{"z": {Name: "z"}, "a": {Name: "a"}}}
	repos := registry.List()
	core.AssertLen(t, repos, 2)
	core.AssertEqual(t, "a", repos[0].Name)
}

func TestRepos_Registry_List_Bad(t *core.T) {
	registry := &Registry{Repos: map[string]*Repo{}}
	repos := registry.List()
	core.AssertEmpty(t, repos)
	core.AssertNotNil(t, repos)
}

func TestRepos_Registry_List_Ugly(t *core.T) {
	registry := &Registry{Repos: nil}
	repos := registry.List()
	core.AssertEmpty(t, repos)
	core.AssertNotNil(t, repos)
}

func TestRepos_Registry_Get_Good(t *core.T) {
	registry := &Registry{Repos: map[string]*Repo{"api": {Name: "api"}}}
	repo, ok := registry.Get("api")
	core.AssertTrue(t, ok)
	core.AssertEqual(t, "api", repo.Name)
}

func TestRepos_Registry_Get_Bad(t *core.T) {
	registry := &Registry{Repos: map[string]*Repo{"api": {Name: "api"}}}
	repo, ok := registry.Get("missing")
	core.AssertFalse(t, ok)
	core.AssertNil(t, repo)
}

func TestRepos_Registry_Get_Ugly(t *core.T) {
	registry := &Registry{Repos: map[string]*Repo{"nil": nil}}
	repo, ok := registry.Get("nil")
	core.AssertTrue(t, ok)
	core.AssertNil(t, repo)
}

func TestRepos_Registry_ByType_Good(t *core.T) {
	registry := &Registry{Repos: map[string]*Repo{"z": {Name: "z", Type: "service"}, "a": {Name: "a", Type: "service"}}}
	repos := registry.ByType("service")
	core.AssertLen(t, repos, 2)
	core.AssertEqual(t, "a", repos[0].Name)
}

func TestRepos_Registry_ByType_Bad(t *core.T) {
	registry := &Registry{Repos: map[string]*Repo{"api": {Name: "api", Type: "service"}}}
	repos := registry.ByType("docs")
	core.AssertEmpty(t, repos)
	core.AssertNil(t, repos)
}

func TestRepos_Registry_ByType_Ugly(t *core.T) {
	registry := &Registry{Repos: map[string]*Repo{"api": {Name: "api", Type: ""}}}
	repos := registry.ByType("")
	core.AssertLen(t, repos, 1)
	core.AssertEqual(t, "api", repos[0].Name)
}

func TestRepos_Repo_Exists_Good(t *core.T) {
	medium := ax7RegistryMedium(t)
	core.RequireNoError(t, medium.EnsureDir(ax7TestReposApi34e3c4))
	repo := &Repo{Path: ax7TestReposApi34e3c4, registry: &Registry{medium: medium}}
	got := repo.Exists()
	core.AssertTrue(t, got)
	core.AssertTrue(t, medium.IsDir(ax7TestReposApi34e3c4))
}

func TestRepos_Repo_Exists_Bad(t *core.T) {
	medium := ax7RegistryMedium(t)
	repo := &Repo{Path: "repos/missing", registry: &Registry{medium: medium}}
	got := repo.Exists()
	core.AssertFalse(t, got)
	core.AssertFalse(t, medium.IsDir("repos/missing"))
}

func TestRepos_Repo_Exists_Ugly(t *core.T) {
	dir := t.TempDir()
	repo := &Repo{Path: dir}
	got := repo.Exists()
	core.AssertTrue(t, got)
	core.AssertFalse(t, stderrors.Is(os.ErrNotExist, os.ErrPermission))
}

func TestRepos_Repo_IsGitRepo_Good(t *core.T) {
	medium := ax7RegistryMedium(t)
	core.RequireNoError(t, medium.EnsureDir(ax7TestReposApiGit64cb4f))
	repo := &Repo{Path: ax7TestReposApi34e3c4, registry: &Registry{medium: medium}}
	got := repo.IsGitRepo()
	core.AssertTrue(t, got)
	core.AssertTrue(t, medium.IsDir(ax7TestReposApiGit64cb4f))
}

func TestRepos_Repo_IsGitRepo_Bad(t *core.T) {
	medium := ax7RegistryMedium(t)
	core.RequireNoError(t, medium.EnsureDir(ax7TestReposApi34e3c4))
	repo := &Repo{Path: ax7TestReposApi34e3c4, registry: &Registry{medium: medium}}
	got := repo.IsGitRepo()
	core.AssertFalse(t, got)
	core.AssertFalse(t, medium.IsDir(ax7TestReposApiGit64cb4f))
}

func TestRepos_Repo_IsGitRepo_Ugly(t *core.T) {
	medium := ax7RegistryMedium(t)
	core.RequireNoError(t, medium.Write(ax7TestReposApiGit64cb4f, "file-not-dir"))
	repo := &Repo{Path: ax7TestReposApi34e3c4, registry: &Registry{medium: medium}}
	got := repo.IsGitRepo()
	core.AssertFalse(t, got)
	core.AssertTrue(t, medium.IsFile(ax7TestReposApiGit64cb4f))
}
