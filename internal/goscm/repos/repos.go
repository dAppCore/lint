package repos

import (
	"os"
	"path/filepath"
	"sort"
	"strings"

	coreio "dappco.re/go/io"
	"gopkg.in/yaml.v3"
)

const (
	reposReposYaml9e36f4 = "repos.yaml"
)

type Registry struct {
	Version  int              `yaml:"version"`
	Org      string           `yaml:"org"`
	BasePath string           `yaml:"base_path"`
	Repos    map[string]*Repo `yaml:"repos"`
	Defaults RegistryDefaults `yaml:"defaults"`
	medium   coreio.Medium    `yaml:"-"`
}

type RegistryDefaults struct {
	CI      string `yaml:"ci"`
	License string `yaml:"license"`
	Branch  string `yaml:"branch"`
}

type Repo struct {
	Name        string    `yaml:"-"`
	Type        string    `yaml:"type"`
	DependsOn   []string  `yaml:"depends_on"`
	Description string    `yaml:"description"`
	Docs        bool      `yaml:"docs"`
	CI          string    `yaml:"ci"`
	Domain      string    `yaml:"domain,omitempty"`
	Clone       *bool     `yaml:"clone,omitempty"`
	Path        string    `yaml:"path,omitempty"`
	registry    *Registry `yaml:"-"`
}

func LoadRegistry(m coreio.Medium, path string) (*Registry, error) {
	content, err := m.Read(path)
	if err != nil {
		return nil, err
	}
	var reg Registry
	if err := yaml.Unmarshal([]byte(content), &reg); err != nil {
		return nil, err
	}
	reg.medium = m
	reg.BasePath = expandPath(reg.BasePath)
	for name, repo := range reg.Repos {
		repo.Name = name
		if repo.Path == "" {
			repo.Path = filepath.Join(reg.BasePath, name)
		} else {
			repo.Path = expandPath(repo.Path)
		}
		if repo.CI == "" {
			repo.CI = reg.Defaults.CI
		}
		repo.registry = &reg
	}
	return &reg, nil
}

func FindRegistry(m coreio.Medium) (string, error) {
	dir, err := os.Getwd()
	if err != nil {
		return "", err
	}
	for {
		for _, candidate := range []string{
			filepath.Join(dir, reposReposYaml9e36f4),
			filepath.Join(dir, ".core", reposReposYaml9e36f4),
		} {
			if m.Exists(candidate) {
				return candidate, nil
			}
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	home, err := os.UserHomeDir()
	if err == nil {
		for _, candidate := range []string{
			filepath.Join(home, "Code", "host-uk", ".core", reposReposYaml9e36f4),
			filepath.Join(home, "Code", "host-uk", reposReposYaml9e36f4),
			filepath.Join(home, ".config", "core", reposReposYaml9e36f4),
		} {
			if m.Exists(candidate) {
				return candidate, nil
			}
		}
	}
	return "", os.ErrNotExist
}

func (r *Registry) List() []*Repo {
	repos := make([]*Repo, 0, len(r.Repos))
	for _, repo := range r.Repos {
		repos = append(repos, repo)
	}
	sort.Slice(repos, func(i, j int) bool {
		return repos[i].Name < repos[j].Name
	})
	return repos
}

func (r *Registry) Get(name string) (*Repo, bool) {
	repo, ok := r.Repos[name]
	return repo, ok
}

func (r *Registry) ByType(kind string) []*Repo {
	var repos []*Repo
	for _, repo := range r.Repos {
		if repo.Type == kind {
			repos = append(repos, repo)
		}
	}
	sort.Slice(repos, func(i, j int) bool {
		return repos[i].Name < repos[j].Name
	})
	return repos
}

func (repo *Repo) Exists() bool {
	return repo.getMedium().IsDir(repo.Path)
}

func (repo *Repo) IsGitRepo() bool {
	return repo.getMedium().IsDir(filepath.Join(repo.Path, ".git"))
}

func (repo *Repo) getMedium() coreio.Medium {
	if repo.registry != nil && repo.registry.medium != nil {
		return repo.registry.medium
	}
	return coreio.Local
}

func expandPath(path string) string {
	if strings.HasPrefix(path, "~/") {
		home, err := os.UserHomeDir()
		if err != nil {
			return path
		}
		return filepath.Join(home, path[2:])
	}
	return path
}
