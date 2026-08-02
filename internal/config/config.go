package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3" // Third Party Dep for YAML parsing
)

// --- Struct Definitions ---

type JavaConfig struct {
	Executable string   `json:"executable" yaml:"executable"`
	ExtraArgs  []string `json:"extra_args" yaml:"extra_args"`
}

type PathConfig struct {
	InstallDir string `json:"install_dir" yaml:"install_dir"`
	JarFile    string `json:"jar_file" yaml:"jar_file"`
}

type Config struct {
	Java  JavaConfig `json:"java" yaml:"java"`
	Paths PathConfig `json:"paths" yaml:"paths"`
}

// --- Factory Functions Definitions ---

// Default returns a base Config populated with safe baseline values
func Default() Config {
	return Config{
		Java: JavaConfig{
			Executable: "java",
			ExtraArgs:  []string{"-Xms1024M", "-Xmx2048M"},
		},
		Paths: PathConfig{
			InstallDir: "./data",
			JarFile:    "server.jar",
		},
	}
}

// Load reads and parses a configuration file from disk.
// If filePath is empty, it returns the Default Configuration
func Load(filePath string, logger *slog.Logger) (Config, error) {
	// Create a child logger with the component attribute bound
	log := logger.With("component", "config")

	if len(filePath) == 0 {
		log.Info("no config file specified, using default configuration")
		cfg := Default()
		return cfg, cfg.Validate()
	}

	log.Debug("checking config file status info", "path", filePath)
	fileInfo, err := os.Stat(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return Config{}, fmt.Errorf("config file not found: %s", filePath)
		}

		return Config{}, fmt.Errorf("error accessing config file: %w", err)
	}

	if fileInfo.IsDir() {
		return Config{}, fmt.Errorf("config path is a directory, not a file: %s", filePath)
	}

	data, err := os.ReadFile(filePath)
	if err != nil {
		return Config{}, fmt.Errorf("failed to read config file: %w", err)
	}

	cfg := Default()

	// Switch based on file extension
	fileExtension := strings.ToLower(filepath.Ext(filePath))
	switch fileExtension {
	case ".yaml", ".yml":
		if err := yaml.Unmarshal(data, &cfg); err != nil {
			return Config{}, fmt.Errorf("failed to parse YAML config: %w", err)
		}
	case ".json":
		if err := json.Unmarshal(data, &cfg); err != nil {
			return Config{}, fmt.Errorf("failed to parse JSON config: %w", err)
		}
	default:
		return Config{}, fmt.Errorf("unsupported config file extension: %s (only .json, .yaml, or .yml)", fileExtension)
	}

	if err := cfg.Validate(); err != nil {
		return Config{}, fmt.Errorf("invalid configuration: %w", err)
	}

	log.Info("configuration successfully loaded", "path", filePath)
	return cfg, nil
}

// --- Instance Method Definitions ---

func (j JavaConfig) Validate() error {
	if j.Executable == "" {
		return errors.New(("java executable path cannot be empty"))
	}

	return nil
}

func (p PathConfig) Validate() error {
	if p.InstallDir == "" {
		return errors.New("paths.install_dir cannot be empty")
	}
	if p.JarFile == "" {
		return errors.New("paths.jar_file cannot be empty")
	}

	return nil
}

// Validate cascades checks to sub configs
func (c Config) Validate() error {
	if err := c.Java.Validate(); err != nil {
		return fmt.Errorf("Invalid java config: %w", err)
	}

	if err := c.Paths.Validate(); err != nil {
		return fmt.Errorf("invalid paths config: %w", err)
	}

	return nil
}

// String implements fmt.Stringer interface for clean printing
func (c Config) String() string {
	return fmt.Sprintf(
		"Config [Java Exec: %s | Args: %v | InstallDir: %s | Jarfile: %s]",
		c.Java.Executable,
		c.Java.ExtraArgs,
		c.Paths.InstallDir,
		c.Paths.JarFile,
	)
}
