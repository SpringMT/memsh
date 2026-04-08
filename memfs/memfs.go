package memfs

import (
	"bytes"
	"errors"
	"fmt"
	"path"
	"sort"
	"strings"
	"sync"
)

var (
	ErrNotFound = errors.New("memfs: path not found")
)

type File struct {
	Path string
	Size int
}

type FS struct {
	mu    sync.RWMutex
	files map[string]*entry
}

type entry struct {
	data []byte
}

func New() *FS {
	return &FS{
		files: make(map[string]*entry),
	}
}

func (fs *FS) Write(pathname string, data []byte) (File, error) {
	clean, err := normalizePath(pathname)
	if err != nil {
		return File{}, err
	}

	fs.mu.Lock()
	defer fs.mu.Unlock()

	current, ok := fs.files[clean]
	if !ok {
		current = &entry{}
		fs.files[clean] = current
	}

	current.data = bytes.Clone(data)

	return current.toFile(clean), nil
}

func (fs *FS) Read(pathname string) ([]byte, File, error) {
	clean, err := normalizePath(pathname)
	if err != nil {
		return nil, File{}, err
	}

	fs.mu.RLock()
	defer fs.mu.RUnlock()

	current, ok := fs.files[clean]
	if !ok {
		return nil, File{}, ErrNotFound
	}

	return bytes.Clone(current.data), current.toFile(clean), nil
}

func (fs *FS) Stat(pathname string) (File, error) {
	clean, err := normalizePath(pathname)
	if err != nil {
		return File{}, err
	}

	fs.mu.RLock()
	defer fs.mu.RUnlock()

	current, ok := fs.files[clean]
	if !ok {
		return File{}, ErrNotFound
	}

	return current.toFile(clean), nil
}

func (fs *FS) Delete(pathname string) error {
	clean, err := normalizePath(pathname)
	if err != nil {
		return err
	}

	fs.mu.Lock()
	defer fs.mu.Unlock()

	if _, ok := fs.files[clean]; !ok {
		return ErrNotFound
	}
	delete(fs.files, clean)
	return nil
}

func (fs *FS) List(prefix string) ([]File, error) {
	cleanPrefix, err := normalizePrefix(prefix)
	if err != nil {
		return nil, err
	}

	fs.mu.RLock()
	defer fs.mu.RUnlock()

	files := make([]File, 0)
	for name, current := range fs.files {
		if cleanPrefix != "/" && !strings.HasPrefix(name, cleanPrefix) {
			continue
		}
		files = append(files, current.toFile(name))
	}

	sort.Slice(files, func(i, j int) bool {
		return files[i].Path < files[j].Path
	})

	return files, nil
}

func normalizePath(raw string) (string, error) {
	if raw == "" {
		return "", fmt.Errorf("memfs: empty path")
	}
	if raw[0] != '/' {
		return "", fmt.Errorf("memfs: path must be absolute: %q", raw)
	}
	clean := path.Clean(raw)
	if clean == "." || clean == ".." || clean == "" || clean[0] != '/' {
		return "", fmt.Errorf("memfs: invalid path: %q", raw)
	}
	return clean, nil
}

func normalizePrefix(raw string) (string, error) {
	if raw == "" {
		return "/", nil
	}
	clean, err := normalizePath(raw)
	if err != nil {
		return "", err
	}
	if clean != "/" {
		clean += "/"
	}
	return clean, nil
}

func (e *entry) toFile(path string) File {
	return File{
		Path: path,
		Size: len(e.data),
	}
}
