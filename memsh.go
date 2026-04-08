package memsh

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"

	"github.com/SpringMT/memsh/internal/broker"
	"github.com/SpringMT/memsh/memfs"
)

type File struct {
	Path    string
	Content []byte
}

type Session struct {
	id     string
	fs     *memfs.FS
	broker *broker.Broker

	mu       sync.Mutex
	loaded   bool
	executed bool
	closed   bool
}

type Manager struct {
	mu       sync.RWMutex
	nextID   uint64
	sessions map[string]*Session
}

func NewManager() *Manager {
	return &Manager{
		sessions: make(map[string]*Session),
	}
}

func (m *Manager) Open() *Session {
	id := fmt.Sprintf("sess-%d", atomic.AddUint64(&m.nextID, 1))
	fs := memfs.New()
	s := &Session{
		id:     id,
		fs:     fs,
		broker: broker.New(fs),
	}

	m.mu.Lock()
	defer m.mu.Unlock()
	m.sessions[id] = s
	return s
}

func (m *Manager) Get(id string) (*Session, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	s, ok := m.sessions[id]
	return s, ok
}

func (m *Manager) Close(id string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if s, ok := m.sessions[id]; ok {
		s.markClosed()
		delete(m.sessions, id)
	}
}

func (s *Session) ID() string {
	return s.id
}

func (s *Session) Load(files []File) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.closed {
		return fmt.Errorf("session %s is closed", s.id)
	}
	if s.loaded {
		return fmt.Errorf("session %s is already loaded", s.id)
	}
	if len(files) == 0 {
		return fmt.Errorf("session %s requires at least one input file", s.id)
	}

	for _, file := range files {
		if !isInputPath(file.Path) {
			return fmt.Errorf("session %s only accepts /input paths, got %q", s.id, file.Path)
		}
		if _, err := s.fs.Write(file.Path, file.Content); err != nil {
			return err
		}
	}

	s.loaded = true
	return nil
}

func (s *Session) Execute(ctx context.Context, input string) (broker.Result, error) {
	s.mu.Lock()
	if s.closed {
		s.mu.Unlock()
		return broker.Result{}, fmt.Errorf("session %s is closed", s.id)
	}
	if !s.loaded {
		s.mu.Unlock()
		return broker.Result{}, fmt.Errorf("session %s is not loaded", s.id)
	}
	if s.executed {
		s.mu.Unlock()
		return broker.Result{}, fmt.Errorf("session %s has already executed", s.id)
	}
	s.executed = true
	s.mu.Unlock()

	return s.broker.ExecuteDSL(ctx, input)
}

func (s *Session) Read(path string) ([]byte, memfs.File, error) {
	if !isReadablePath(path) {
		return nil, memfs.File{}, fmt.Errorf("session %s path is outside readable namespaces: %q", s.id, path)
	}
	return s.fs.Read(path)
}

func (s *Session) List(prefix string) ([]memfs.File, error) {
	if prefix != "" && !isReadablePath(prefix) && prefix != "/" {
		return nil, fmt.Errorf("session %s prefix is outside readable namespaces: %q", s.id, prefix)
	}
	return s.fs.List(prefix)
}

func (s *Session) Close() {
	s.markClosed()
}

func (s *Session) markClosed() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.closed = true
}

func isInputPath(p string) bool {
	return p == "/input" || len(p) > len("/input/") && p[:len("/input/")] == "/input/"
}

func isReadablePath(p string) bool {
	return p == "/" ||
		p == "/input" || len(p) > len("/input/") && p[:len("/input/")] == "/input/" ||
		p == "/work" || len(p) > len("/work/") && p[:len("/work/")] == "/work/" ||
		p == "/output" || len(p) > len("/output/") && p[:len("/output/")] == "/output/"
}
