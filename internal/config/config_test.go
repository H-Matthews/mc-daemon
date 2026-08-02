package config_test

import (
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"testing"

	"mc-daemon/internal/config"
)

// Helper function to create a silent logger for unit testing
func testLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

func TestDefaultConfig(t *testing.T) {
	cfg := config.Default()

	if err := cfg.Validate(); err != nil {
		t.Fatalf("Default() config failed validation: %v", err)
	}

	if cfg.Java.Executable != "java" {
		t.Errorf("expected default java executable 'java', got '%s'", cfg.Java.Executable)
	}

	if cfg.Paths.InstallDir != "./data" {
		t.Errorf("expected default data dir './data', got '%s'", cfg.Paths.InstallDir)
	}
}

func TestConfigValidation(t *testing.T) {
	tests := []struct {
		name    string
		cfg     config.Config
		wantErr bool
	}{
		{
			name:    "valid configuration",
			cfg:     config.Default(),
			wantErr: false,
		},
		{
			name: "missing java executable",
			cfg: config.Config{
				Java:  config.JavaConfig{Executable: ""},
				Paths: config.PathConfig{InstallDir: "./data", JarFile: "server.jar"},
			},
			wantErr: true,
		},
		{
			name: "missing install directory",
			cfg: config.Config{
				Java:  config.JavaConfig{Executable: "java"},
				Paths: config.PathConfig{InstallDir: "", JarFile: "server.jar"},
			},
			wantErr: true,
		},
		{
			name: "missing jar file",
			cfg: config.Config{
				Java:  config.JavaConfig{Executable: "java"},
				Paths: config.PathConfig{InstallDir: "./data", JarFile: ""},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.cfg.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestLoad(t *testing.T) {
	// Create a temporary directory for test files (cleaned up automatically)
	tempDir := t.TempDir()

	// 1. Create a valid YAML test file
	yamlPath := filepath.Join(tempDir, "config.yaml")
	yamlData := []byte(`
java:
  executable: "/usr/bin/java"
  extra_args: ["-Xmx4G"]
paths:
  install_dir: "/var/mc"
  jar_file: "paper.jar"
`)
	if err := os.WriteFile(yamlPath, yamlData, 0644); err != nil {
		t.Fatalf("failed to write test YAML file: %v", err)
	}

	// 2. Create a valid JSON test file
	jsonPath := filepath.Join(tempDir, "config.json")
	jsonData := []byte(`{
		"java": { "executable": "java17", "extra_args": [] },
		"paths": { "install_dir": "/tmp/mc", "jar_file": "spigot.jar" }
	}`)
	if err := os.WriteFile(jsonPath, jsonData, 0644); err != nil {
		t.Fatalf("failed to write test JSON file: %v", err)
	}

	// 3. Create an invalid syntax file
	invalidPath := filepath.Join(tempDir, "invalid.yaml")
	if err := os.WriteFile(invalidPath, []byte("invalid: [yaml: content"), 0644); err != nil {
		t.Fatalf("failed to write invalid test file: %v", err)
	}

	// 4. Create an unsupported extension file
	unsupportedPath := filepath.Join(tempDir, "config.xml")
	if err := os.WriteFile(unsupportedPath, []byte("<config></config>"), 0644); err != nil {
		t.Fatalf("failed to write unsupported test file: %v", err)
	}

	logger := testLogger()

	tests := []struct {
		name         string
		filePath     string
		wantErr      bool
		checkDataDir string
	}{
		{
			name:         "empty path returns default config",
			filePath:     "",
			wantErr:      false,
			checkDataDir: "./data",
		},
		{
			name:         "valid YAML load",
			filePath:     yamlPath,
			wantErr:      false,
			checkDataDir: "/var/mc",
		},
		{
			name:         "valid JSON load",
			filePath:     jsonPath,
			wantErr:      false,
			checkDataDir: "/tmp/mc",
		},
		{
			name:     "file does not exist",
			filePath: filepath.Join(tempDir, "nonexistent.yaml"),
			wantErr:  true,
		},
		{
			name:     "invalid syntax in file",
			filePath: invalidPath,
			wantErr:  true,
		},
		{
			name:     "unsupported file extension",
			filePath: unsupportedPath,
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg, err := config.Load(tt.filePath, logger)

			if (err != nil) != tt.wantErr {
				t.Fatalf("Load() error = %v, wantErr %v", err, tt.wantErr)
			}

			if !tt.wantErr && cfg.Paths.InstallDir != tt.checkDataDir {
				t.Errorf("expected DataDir = %s, got %s", tt.checkDataDir, cfg.Paths.InstallDir)
			}
		})
	}
}

func TestLoad_Extended(t *testing.T) {
	tempDir := t.TempDir()

	// Create a directory to test the info.IsDir() check
	subDir := filepath.Join(tempDir, "testdir.yaml")
	if err := os.Mkdir(subDir, 0755); err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}

	// Create a file that fails struct validation post-parse
	invalidSchemaPath := filepath.Join(tempDir, "bad_schema.yaml")
	invalidSchemaData := []byte(`
java:
  executable: "" # Fails validation!
paths:
  data_dir: "./data"
  jar_file: "server.jar"
`)
	if err := os.WriteFile(invalidSchemaPath, invalidSchemaData, 0644); err != nil {
		t.Fatalf("failed to write bad schema file: %v", err)
	}

	logger := testLogger()

	tests := []struct {
		name     string
		filePath string
		wantErr  bool
	}{
		{
			name:     "path is a directory instead of a file",
			filePath: subDir,
			wantErr:  true,
		},
		{
			name:     "valid YAML syntax but invalid schema values",
			filePath: invalidSchemaPath,
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := config.Load(tt.filePath, logger)
			if (err != nil) != tt.wantErr {
				t.Errorf("Load() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestConfigStringer(t *testing.T) {
	cfg := config.Default()
	str := cfg.String()
	if str == "" {
		t.Error("expected non-empty string output from String()")
	}
}
