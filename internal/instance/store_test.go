// NewAPI Tools - Instance store tests
package instance

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/simenty/newapi-tools/internal/apperr"
)

// helper: create a temp file path for the store
func tempStorePath(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	return filepath.Join(dir, "instances.json")
}

// helper: create a Store with a temp file
func newTestStore(t *testing.T) *Store {
	t.Helper()
	return NewStore(tempStorePath(t))
}

// helper: create a sample instance
func sampleInstance(name string, port int, active bool) Instance {
	inst := NewInstance(name, "/opt/newapi", port, "calciumion/new-api:latest")
	inst.Active = active
	return *inst
}

// --- NewStore / DefaultStorePath ---

func TestNewStore(t *testing.T) {
	path := "/tmp/test-instances.json"
	s := NewStore(path)
	if s == nil {
		t.Fatal("NewStore returned nil")
	}
	if s.path != path {
		t.Errorf("expected path %q, got %q", path, s.path)
	}
}

func TestDefaultStorePath(t *testing.T) {
	p := DefaultStorePath()
	if p == "" {
		t.Error("DefaultStorePath returned empty string")
	}
	// Should contain "newapi-tools" somewhere in the path
	if !contains(p, "newapi-tools") {
		t.Errorf("DefaultStorePath() = %q, expected to contain 'newapi-tools'", p)
	}
}

func contains(s, sub string) bool {
	return len(s) >= len(sub) && (s == sub || len(sub) == 0 || containsSubstr(s, sub))
}

func containsSubstr(s, sub string) bool {
	for i := 0; i <= len(s)-len(sub); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}

// --- Load ---

func TestLoadNonExistentFile(t *testing.T) {
	s := newTestStore(t)
	instances, err := s.Load()
	if err != nil {
		t.Fatalf("Load() returned error for non-existent file: %v", err)
	}
	if instances == nil {
		t.Error("Load() returned nil, expected empty slice")
	}
	if len(instances) != 0 {
		t.Errorf("Load() returned %d instances, expected 0", len(instances))
	}
}

func TestLoadEmptyFile(t *testing.T) {
	s := newTestStore(t)
	// Write an empty JSON array to the file
	if err := os.MkdirAll(filepath.Dir(s.path), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(s.path, []byte("[]"), 0644); err != nil {
		t.Fatal(err)
	}

	instances, err := s.Load()
	if err != nil {
		t.Fatalf("Load() returned error for empty file: %v", err)
	}
	if len(instances) != 0 {
		t.Errorf("Load() returned %d instances, expected 0", len(instances))
	}
}

func TestLoadInvalidJSON(t *testing.T) {
	s := newTestStore(t)
	if err := os.MkdirAll(filepath.Dir(s.path), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(s.path, []byte("not json"), 0644); err != nil {
		t.Fatal(err)
	}

	_, err := s.Load()
	if err == nil {
		t.Error("Load() should return error for invalid JSON")
	}
}

// --- Add + Load round-trip ---

func TestAddAndLoad(t *testing.T) {
	s := newTestStore(t)
	inst := sampleInstance("test-inst", 3000, false)

	if err := s.Add(inst); err != nil {
		t.Fatalf("Add() returned error: %v", err)
	}

	instances, err := s.Load()
	if err != nil {
		t.Fatalf("Load() returned error: %v", err)
	}
	if len(instances) != 1 {
		t.Fatalf("expected 1 instance, got %d", len(instances))
	}
	if instances[0].Name != "test-inst" {
		t.Errorf("expected name 'test-inst', got %q", instances[0].Name)
	}
	if instances[0].Port != 3000 {
		t.Errorf("expected port 3000, got %d", instances[0].Port)
	}
	if instances[0].ComposeProject != "newapi-test-inst" {
		t.Errorf("expected compose_project 'newapi-test-inst', got %q", instances[0].ComposeProject)
	}
}

// --- Add duplicate name ---

func TestAddDuplicateName(t *testing.T) {
	s := newTestStore(t)
	inst1 := sampleInstance("dup", 3000, false)
	inst2 := sampleInstance("dup", 3001, false)

	if err := s.Add(inst1); err != nil {
		t.Fatalf("first Add() returned error: %v", err)
	}

	err := s.Add(inst2)
	if err == nil {
		t.Error("Add() should return error for duplicate name")
	}

	var appErr *apperr.AppError
	if !errors.As(err, &appErr) {
		t.Fatalf("expected AppError, got %T", err)
	}
	if appErr.Code != apperr.CodeInstanceExists {
		t.Errorf("expected code %q, got %q", apperr.CodeInstanceExists, appErr.Code)
	}
}

// --- Add port conflict ---

func TestAddPortConflict(t *testing.T) {
	s := newTestStore(t)
	inst1 := sampleInstance("inst-a", 3000, false)
	inst2 := sampleInstance("inst-b", 3000, false) // same port, different name

	if err := s.Add(inst1); err != nil {
		t.Fatalf("first Add() returned error: %v", err)
	}

	err := s.Add(inst2)
	if err == nil {
		t.Error("Add() should return error for port conflict")
	}

	var appErr *apperr.AppError
	if !errors.As(err, &appErr) {
		t.Fatalf("expected AppError, got %T", err)
	}
	if appErr.Code != apperr.CodeInstanceExists {
		t.Errorf("expected code %q, got %q", apperr.CodeInstanceExists, appErr.Code)
	}
}

// --- Remove ---

func TestRemoveNormal(t *testing.T) {
	s := newTestStore(t)
	inst1 := sampleInstance("keep", 3000, false)
	inst2 := sampleInstance("remove", 3001, false)

	if err := s.Add(inst1); err != nil {
		t.Fatal(err)
	}
	if err := s.Add(inst2); err != nil {
		t.Fatal(err)
	}

	if err := s.Remove("remove"); err != nil {
		t.Fatalf("Remove() returned error: %v", err)
	}

	instances, err := s.Load()
	if err != nil {
		t.Fatal(err)
	}
	if len(instances) != 1 {
		t.Fatalf("expected 1 instance after remove, got %d", len(instances))
	}
	if instances[0].Name != "keep" {
		t.Errorf("expected remaining instance 'keep', got %q", instances[0].Name)
	}
}

// --- Remove active instance ---

func TestRemoveActiveInstance(t *testing.T) {
	s := newTestStore(t)
	inst := sampleInstance("active-inst", 3000, true)

	if err := s.Add(inst); err != nil {
		t.Fatal(err)
	}

	err := s.Remove("active-inst")
	if err == nil {
		t.Error("Remove() should return error for active instance")
	}

	var appErr *apperr.AppError
	if !errors.As(err, &appErr) {
		t.Fatalf("expected AppError, got %T", err)
	}
	if appErr.Code != apperr.CodeInstanceActive {
		t.Errorf("expected code %q, got %q", apperr.CodeInstanceActive, appErr.Code)
	}
}

// --- Remove non-existent instance ---

func TestRemoveNotFound(t *testing.T) {
	s := newTestStore(t)

	err := s.Remove("nonexistent")
	if err == nil {
		t.Error("Remove() should return error for non-existent instance")
	}

	var appErr *apperr.AppError
	if !errors.As(err, &appErr) {
		t.Fatalf("expected AppError, got %T", err)
	}
	if appErr.Code != apperr.CodeInstanceNotFound {
		t.Errorf("expected code %q, got %q", apperr.CodeInstanceNotFound, appErr.Code)
	}
}

// --- Get ---

func TestGet(t *testing.T) {
	s := newTestStore(t)
	inst := sampleInstance("findme", 3000, false)

	if err := s.Add(inst); err != nil {
		t.Fatal(err)
	}

	found, err := s.Get("findme")
	if err != nil {
		t.Fatalf("Get() returned error: %v", err)
	}
	if found.Name != "findme" {
		t.Errorf("expected name 'findme', got %q", found.Name)
	}
}

func TestGetNotFound(t *testing.T) {
	s := newTestStore(t)

	_, err := s.Get("nonexistent")
	if err == nil {
		t.Error("Get() should return error for non-existent instance")
	}

	var appErr *apperr.AppError
	if !errors.As(err, &appErr) {
		t.Fatalf("expected AppError, got %T", err)
	}
	if appErr.Code != apperr.CodeInstanceNotFound {
		t.Errorf("expected code %q, got %q", apperr.CodeInstanceNotFound, appErr.Code)
	}
}

// --- SetActive ---

func TestSetActive(t *testing.T) {
	s := newTestStore(t)
	inst1 := sampleInstance("inst-a", 3000, true)
	inst2 := sampleInstance("inst-b", 3001, false)

	if err := s.Add(inst1); err != nil {
		t.Fatal(err)
	}
	if err := s.Add(inst2); err != nil {
		t.Fatal(err)
	}

	// Switch active to inst-b
	if err := s.SetActive("inst-b"); err != nil {
		t.Fatalf("SetActive() returned error: %v", err)
	}

	instances, err := s.Load()
	if err != nil {
		t.Fatal(err)
	}

	for _, inst := range instances {
		if inst.Name == "inst-a" && inst.Active {
			t.Error("inst-a should no longer be active")
		}
		if inst.Name == "inst-b" && !inst.Active {
			t.Error("inst-b should be active")
		}
	}
}

func TestSetActiveNotFound(t *testing.T) {
	s := newTestStore(t)

	err := s.SetActive("nonexistent")
	if err == nil {
		t.Error("SetActive() should return error for non-existent instance")
	}

	var appErr *apperr.AppError
	if !errors.As(err, &appErr) {
		t.Fatalf("expected AppError, got %T", err)
	}
	if appErr.Code != apperr.CodeInstanceNotFound {
		t.Errorf("expected code %q, got %q", apperr.CodeInstanceNotFound, appErr.Code)
	}
}

// --- GetActive ---

func TestGetActiveNone(t *testing.T) {
	s := newTestStore(t)
	inst := sampleInstance("no-active", 3000, false)

	if err := s.Add(inst); err != nil {
		t.Fatal(err)
	}

	active, err := s.GetActive()
	if err != nil {
		t.Fatalf("GetActive() returned error: %v", err)
	}
	if active != nil {
		t.Errorf("expected nil, got instance %q", active.Name)
	}
}

func TestGetActiveExists(t *testing.T) {
	s := newTestStore(t)
	inst1 := sampleInstance("inactive", 3000, false)
	inst2 := sampleInstance("active", 3001, true)

	if err := s.Add(inst1); err != nil {
		t.Fatal(err)
	}
	if err := s.Add(inst2); err != nil {
		t.Fatal(err)
	}

	active, err := s.GetActive()
	if err != nil {
		t.Fatalf("GetActive() returned error: %v", err)
	}
	if active == nil {
		t.Fatal("expected active instance, got nil")
	}
	if active.Name != "active" {
		t.Errorf("expected active instance 'active', got %q", active.Name)
	}
}

func TestGetActiveEmptyStore(t *testing.T) {
	s := newTestStore(t)

	active, err := s.GetActive()
	if err != nil {
		t.Fatalf("GetActive() returned error: %v", err)
	}
	if active != nil {
		t.Errorf("expected nil for empty store, got instance %q", active.Name)
	}
}

// --- List ---

func TestList(t *testing.T) {
	s := newTestStore(t)
	inst1 := sampleInstance("list-a", 3000, false)
	inst2 := sampleInstance("list-b", 3001, false)

	if err := s.Add(inst1); err != nil {
		t.Fatal(err)
	}
	if err := s.Add(inst2); err != nil {
		t.Fatal(err)
	}

	instances, err := s.List()
	if err != nil {
		t.Fatalf("List() returned error: %v", err)
	}
	if len(instances) != 2 {
		t.Errorf("expected 2 instances, got %d", len(instances))
	}
}

// --- Atomic write verification ---

func TestAtomicWrite(t *testing.T) {
	s := newTestStore(t)
	inst := sampleInstance("atomic", 3000, false)

	if err := s.Add(inst); err != nil {
		t.Fatalf("Add() returned error: %v", err)
	}

	// Verify the file exists
	if _, err := os.Stat(s.path); os.IsNotExist(err) {
		t.Fatal("instances.json file should exist after Add()")
	}

	// Verify the content is valid JSON
	data, err := os.ReadFile(s.path)
	if err != nil {
		t.Fatalf("failed to read instances.json: %v", err)
	}

	var instances []Instance
	if err := json.Unmarshal(data, &instances); err != nil {
		t.Fatalf("instances.json contains invalid JSON: %v", err)
	}

	if len(instances) != 1 {
		t.Errorf("expected 1 instance in file, got %d", len(instances))
	}
	if instances[0].Name != "atomic" {
		t.Errorf("expected instance name 'atomic', got %q", instances[0].Name)
	}

	// Verify no temp file is left behind
	tmpPath := s.path + ".tmp"
	if _, err := os.Stat(tmpPath); !os.IsNotExist(err) {
		t.Error("temp file should not exist after successful save")
	}
}

// --- NewInstance ---

func TestNewInstance(t *testing.T) {
	inst := NewInstance("myinst", "/opt/myinst", 4000, "calciumion/new-api:latest")
	if inst.Name != "myinst" {
		t.Errorf("expected name 'myinst', got %q", inst.Name)
	}
	if inst.Home != "/opt/myinst" {
		t.Errorf("expected home '/opt/myinst', got %q", inst.Home)
	}
	if inst.Port != 4000 {
		t.Errorf("expected port 4000, got %d", inst.Port)
	}
	if inst.DockerImage != "calciumion/new-api:latest" {
		t.Errorf("expected docker_image 'calciumion/new-api:latest', got %q", inst.DockerImage)
	}
	if inst.ComposeProject != "newapi-myinst" {
		t.Errorf("expected compose_project 'newapi-myinst', got %q", inst.ComposeProject)
	}
	if inst.Active {
		t.Error("new instance should not be active by default")
	}
	if inst.CreatedAt == "" {
		t.Error("created_at should not be empty")
	}
}
