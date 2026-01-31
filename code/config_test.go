package code

import (
	"errors"
	"testing"
	"time"
)

func TestConfig_ValidateRequired_Index(t *testing.T) {
	cfg := Config{
		Docs:   &mockStore{},
		Run:    &mockRunner{},
		Engine: &mockEngine{},
	}
	err := cfg.Validate()
	if err == nil {
		t.Fatal("expected error for nil Index")
	}
	if !errors.Is(err, ErrConfiguration) {
		t.Errorf("expected ErrConfiguration, got %v", err)
	}
	if !containsStr(err.Error(), "Index") {
		t.Errorf("expected error to mention Index, got %q", err.Error())
	}
}

func TestConfig_ValidateRequired_Docs(t *testing.T) {
	cfg := Config{
		Index:  &mockIndex{},
		Run:    &mockRunner{},
		Engine: &mockEngine{},
	}
	err := cfg.Validate()
	if err == nil {
		t.Fatal("expected error for nil Docs")
	}
	if !errors.Is(err, ErrConfiguration) {
		t.Errorf("expected ErrConfiguration, got %v", err)
	}
	if !containsStr(err.Error(), "Docs") {
		t.Errorf("expected error to mention Docs, got %q", err.Error())
	}
}

func TestConfig_ValidateRequired_Run(t *testing.T) {
	cfg := Config{
		Index:  &mockIndex{},
		Docs:   &mockStore{},
		Engine: &mockEngine{},
	}
	err := cfg.Validate()
	if err == nil {
		t.Fatal("expected error for nil Run")
	}
	if !errors.Is(err, ErrConfiguration) {
		t.Errorf("expected ErrConfiguration, got %v", err)
	}
	if !containsStr(err.Error(), "Run") {
		t.Errorf("expected error to mention Run, got %q", err.Error())
	}
}

func TestConfig_ValidateRequired_Engine(t *testing.T) {
	cfg := Config{
		Index: &mockIndex{},
		Docs:  &mockStore{},
		Run:   &mockRunner{},
	}
	err := cfg.Validate()
	if err == nil {
		t.Fatal("expected error for nil Engine")
	}
	if !errors.Is(err, ErrConfiguration) {
		t.Errorf("expected ErrConfiguration, got %v", err)
	}
	if !containsStr(err.Error(), "Engine") {
		t.Errorf("expected error to mention Engine, got %q", err.Error())
	}
}

func TestConfig_ValidateRequired_AllNil(t *testing.T) {
	cfg := Config{}
	err := cfg.Validate()
	if err == nil {
		t.Fatal("expected error for all nil fields")
	}
	if !errors.Is(err, ErrConfiguration) {
		t.Errorf("expected ErrConfiguration, got %v", err)
	}
	// Should mention all missing fields
	errStr := err.Error()
	for _, field := range []string{"Index", "Docs", "Run", "Engine"} {
		if !containsStr(errStr, field) {
			t.Errorf("expected error to mention %s, got %q", field, errStr)
		}
	}
}

func TestConfig_Validate_Success(t *testing.T) {
	cfg := Config{
		Index:  &mockIndex{},
		Docs:   &mockStore{},
		Run:    &mockRunner{},
		Engine: &mockEngine{},
	}
	err := cfg.Validate()
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
}

func TestConfig_DefaultLanguage(t *testing.T) {
	cfg := Config{
		Index:  &mockIndex{},
		Docs:   &mockStore{},
		Run:    &mockRunner{},
		Engine: &mockEngine{},
	}
	cfg.applyDefaults()
	if cfg.DefaultLanguage != "go" {
		t.Errorf("expected DefaultLanguage to be 'go', got %q", cfg.DefaultLanguage)
	}
}

func TestConfig_DefaultLanguage_PreserveExisting(t *testing.T) {
	cfg := Config{
		Index:           &mockIndex{},
		Docs:            &mockStore{},
		Run:             &mockRunner{},
		Engine:          &mockEngine{},
		DefaultLanguage: "javascript",
	}
	cfg.applyDefaults()
	if cfg.DefaultLanguage != "javascript" {
		t.Errorf("expected DefaultLanguage to remain 'javascript', got %q", cfg.DefaultLanguage)
	}
}

func TestConfig_MaxToolCalls_Zero(t *testing.T) {
	cfg := Config{
		Index:        &mockIndex{},
		Docs:         &mockStore{},
		Run:          &mockRunner{},
		Engine:       &mockEngine{},
		MaxToolCalls: 0, // Zero means unlimited
	}
	err := cfg.Validate()
	if err != nil {
		t.Fatalf("expected no error with zero MaxToolCalls, got %v", err)
	}
}

func TestConfig_Logger_Optional(t *testing.T) {
	cfg := Config{
		Index:  &mockIndex{},
		Docs:   &mockStore{},
		Run:    &mockRunner{},
		Engine: &mockEngine{},
		Logger: nil, // nil Logger should be valid
	}
	err := cfg.Validate()
	if err != nil {
		t.Fatalf("expected no error with nil Logger, got %v", err)
	}
}

func TestConfig_WithAllOptions(t *testing.T) {
	logger := &mockLogger{}
	cfg := Config{
		Index:           &mockIndex{},
		Docs:            &mockStore{},
		Run:             &mockRunner{},
		Engine:          &mockEngine{},
		DefaultTimeout:  30 * time.Second,
		DefaultLanguage: "typescript",
		MaxToolCalls:    100,
		Logger:          logger,
	}
	err := cfg.Validate()
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	cfg.applyDefaults()
	// DefaultLanguage should not be overwritten
	if cfg.DefaultLanguage != "typescript" {
		t.Errorf("expected DefaultLanguage 'typescript', got %q", cfg.DefaultLanguage)
	}
}

// containsStr checks if s contains substr
func containsStr(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
