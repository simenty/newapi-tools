package instance

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"github.com/simenty/newapi-tools/internal/apperr"
)

// Store manages instance metadata with file-based persistence.
// All operations are goroutine-safe via mutex.
type Store struct {
	path string
	mu   sync.Mutex
}

// NewStore creates a Store backed by the given file path.
func NewStore(path string) *Store {
	return &Store{path: path}
}

// DefaultStorePath returns the default path for instance metadata.
func DefaultStorePath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return "instances.json"
	}
	return filepath.Join(home, ".config", "newapi-tools", "instances.json")
}

// Load reads all instances from the store file.
// Returns empty slice (not nil) if the file does not exist.
func (s *Store) Load() ([]Instance, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	data, err := os.ReadFile(s.path)
	if err != nil {
		if os.IsNotExist(err) {
			return []Instance{}, nil
		}
		return nil, apperr.Wrap(apperr.CodeConfigLoad, "", err)
	}

	var instances []Instance
	if err := json.Unmarshal(data, &instances); err != nil {
		return nil, apperr.Wrap(apperr.CodeConfigLoad, "", fmt.Errorf("invalid instances.json: %w", err))
	}
	return instances, nil
}

// Save writes all instances to the store file using atomic write (temp + rename).
func (s *Store) Save(instances []Instance) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	data, err := json.MarshalIndent(instances, "", "  ")
	if err != nil {
		return apperr.Wrap(apperr.CodeConfigLoad, "", fmt.Errorf("marshal instances: %w", err))
	}

	// Ensure parent directory exists
	if err := os.MkdirAll(filepath.Dir(s.path), 0755); err != nil {
		return apperr.Wrap(apperr.CodeConfigLoad, "", fmt.Errorf("create instances dir: %w", err))
	}

	// Atomic write: temp file then rename
	tmpPath := s.path + ".tmp"
	if err := os.WriteFile(tmpPath, data, 0600); err != nil {
		return apperr.Wrap(apperr.CodeConfigLoad, "", fmt.Errorf("write instances tmp: %w", err))
	}

	if err := os.Rename(tmpPath, s.path); err != nil {
		os.Remove(tmpPath) // cleanup temp file
		return apperr.Wrap(apperr.CodeConfigLoad, "", fmt.Errorf("rename instances file: %w", err))
	}

	return nil
}

// Add appends a new instance after validating uniqueness of name and port.
func (s *Store) Add(inst Instance) error {
	instances, err := s.Load()
	if err != nil {
		return err
	}

	// Check name uniqueness
	for _, existing := range instances {
		if existing.Name == inst.Name {
			return apperr.New(apperr.CodeInstanceExists, fmt.Sprintf("实例 '%s' 已存在", inst.Name), "", nil)
		}
		// Check port conflict
		if existing.Port == inst.Port {
			return apperr.New(apperr.CodeInstanceExists, fmt.Sprintf("端口 %d 已被实例 '%s' 占用", inst.Port, existing.Name), "", nil)
		}
	}

	instances = append(instances, inst)
	return s.Save(instances)
}

// Remove deletes an instance by name. Returns error if the instance is active.
func (s *Store) Remove(name string) error {
	instances, err := s.Load()
	if err != nil {
		return err
	}

	found := false
	newInstances := make([]Instance, 0, len(instances))
	for _, inst := range instances {
		if inst.Name == name {
			found = true
			if inst.Active {
				return apperr.New(apperr.CodeInstanceActive, fmt.Sprintf("实例 '%s' 为当前活跃实例，无法删除", name), "", nil)
			}
			continue // skip this instance (delete it)
		}
		newInstances = append(newInstances, inst)
	}

	if !found {
		return apperr.New(apperr.CodeInstanceNotFound, fmt.Sprintf("实例 '%s' 不存在", name), "", nil)
	}

	return s.Save(newInstances)
}

// Get returns a single instance by name.
func (s *Store) Get(name string) (*Instance, error) {
	instances, err := s.Load()
	if err != nil {
		return nil, err
	}

	for i := range instances {
		if instances[i].Name == name {
			return &instances[i], nil
		}
	}

	return nil, apperr.New(apperr.CodeInstanceNotFound, fmt.Sprintf("实例 '%s' 不存在", name), "", nil)
}

// SetActive marks the given instance as active and deactivates all others.
func (s *Store) SetActive(name string) error {
	instances, err := s.Load()
	if err != nil {
		return err
	}

	found := false
	for i := range instances {
		if instances[i].Name == name {
			instances[i].Active = true
			found = true
		} else {
			instances[i].Active = false
		}
	}

	if !found {
		return apperr.New(apperr.CodeInstanceNotFound, fmt.Sprintf("实例 '%s' 不存在", name), "", nil)
	}

	return s.Save(instances)
}

// GetActive returns the currently active instance, or nil if none is active.
func (s *Store) GetActive() (*Instance, error) {
	instances, err := s.Load()
	if err != nil {
		return nil, err
	}

	for i := range instances {
		if instances[i].Active {
			return &instances[i], nil
		}
	}

	return nil, nil // no active instance
}

// List returns all instances.
func (s *Store) List() ([]Instance, error) {
	return s.Load()
}
