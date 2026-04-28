package io

import (
	"errors"
	goio "io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

type Medium interface {
	Read(path string) (string, error)
	Write(path, content string) error
	WriteMode(path, content string, mode fs.FileMode) error
	EnsureDir(path string) error
	IsFile(path string) bool
	Delete(path string) error
	DeleteAll(path string) error
	Rename(oldPath, newPath string) error
	List(path string) ([]fs.DirEntry, error)
	Stat(path string) (fs.FileInfo, error)
	Open(path string) (fs.File, error)
	Create(path string) (goio.WriteCloser, error)
	Append(path string) (goio.WriteCloser, error)
	ReadStream(path string) (goio.ReadCloser, error)
	WriteStream(path string) (goio.WriteCloser, error)
	Exists(path string) bool
	IsDir(path string) bool
}

type localMedium struct {
	root string
}

var Local Medium = localMedium{root: string(filepath.Separator)}

func NewSandboxed(root string) (Medium, error) {
	if root == "" {
		root = "."
	}
	absolute, err := filepath.Abs(root)
	if err != nil {
		return nil, err
	}
	return localMedium{root: absolute}, nil
}

func (medium localMedium) resolve(path string) string {
	if medium.root == "" || medium.root == string(filepath.Separator) {
		return filepath.Clean(path)
	}
	clean := filepath.Clean(string(filepath.Separator) + path)
	relative := strings.TrimPrefix(clean, string(filepath.Separator))
	return filepath.Join(medium.root, relative)
}

func (medium localMedium) Read(path string) (string, error) {
	data, err := os.ReadFile(medium.resolve(path))
	if err != nil {
		return "", err
	}
	return string(data), nil
}

func (medium localMedium) Write(path, content string) error {
	return medium.WriteMode(path, content, 0o644)
}

func (medium localMedium) WriteMode(path, content string, mode fs.FileMode) error {
	resolved := medium.resolve(path)
	if err := os.MkdirAll(filepath.Dir(resolved), 0o755); err != nil {
		return err
	}
	return os.WriteFile(resolved, []byte(content), mode)
}

func (medium localMedium) EnsureDir(path string) error {
	return os.MkdirAll(medium.resolve(path), 0o755)
}

func (medium localMedium) IsFile(path string) bool {
	info, err := os.Stat(medium.resolve(path))
	return err == nil && !info.IsDir()
}

func (medium localMedium) Delete(path string) error {
	return os.Remove(medium.resolve(path))
}

func (medium localMedium) DeleteAll(path string) error {
	return os.RemoveAll(medium.resolve(path))
}

func (medium localMedium) Rename(oldPath, newPath string) error {
	return os.Rename(medium.resolve(oldPath), medium.resolve(newPath))
}

func (medium localMedium) List(path string) ([]fs.DirEntry, error) {
	return os.ReadDir(medium.resolve(path))
}

func (medium localMedium) Stat(path string) (fs.FileInfo, error) {
	return os.Stat(medium.resolve(path))
}

func (medium localMedium) Open(path string) (fs.File, error) {
	return os.Open(medium.resolve(path))
}

func (medium localMedium) Create(path string) (goio.WriteCloser, error) {
	resolved := medium.resolve(path)
	if err := os.MkdirAll(filepath.Dir(resolved), 0o755); err != nil {
		return nil, err
	}
	return os.Create(resolved)
}

func (medium localMedium) Append(path string) (goio.WriteCloser, error) {
	resolved := medium.resolve(path)
	if err := os.MkdirAll(filepath.Dir(resolved), 0o755); err != nil {
		return nil, err
	}
	return os.OpenFile(resolved, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
}

func (medium localMedium) ReadStream(path string) (goio.ReadCloser, error) {
	return os.Open(medium.resolve(path))
}

func (medium localMedium) WriteStream(path string) (goio.WriteCloser, error) {
	return medium.Create(path)
}

func (medium localMedium) Exists(path string) bool {
	_, err := os.Stat(medium.resolve(path))
	return err == nil || !errors.Is(err, fs.ErrNotExist)
}

func (medium localMedium) IsDir(path string) bool {
	info, err := os.Stat(medium.resolve(path))
	return err == nil && info.IsDir()
}
