package config

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"

	"github.com/pelletier/go-toml/v2"
)

// Config represents the complete application configuration
type Config struct {
	Drives    DrivesConfig    `toml:"drives"`
	Paths     PathsConfig     `toml:"paths"`
	CDRipping CDRippingConfig `toml:"cd_ripping"`
	Execution ExecutionConfig `toml:"execution"`
	Tools     ToolsConfig     `toml:"tools"`
	UI        UIConfig        `toml:"ui"`
	Container ContainerConfig `toml:"container"`
}

// DrivesConfig contains optical drive settings
type DrivesConfig struct {
	CDDrive    string   `toml:"cd_drive"`
	AutoDetect bool     `toml:"auto_detect"`
	Available  []string `toml:"available"`
}

// PathsConfig contains directory and file paths
type PathsConfig struct {
	Music   string `toml:"music"`
	Movies  string `toml:"movies"`
	Config  string `toml:"config"`
	LogFile string `toml:"log_file"`
}

// CDRippingConfig contains CD ripping specific settings
type CDRippingConfig struct {
	RetryCount   int    `toml:"retry_count"`
	RetryDelay   int    `toml:"retry_delay"`
	InitialWait  int    `toml:"initial_wait"`
	AutoEject    bool   `toml:"auto_eject"`
	OutputFormat string `toml:"output_format"`
	CDDBMethod   string `toml:"cddb_method"`
}

// ExecutionConfig contains execution preferences
type ExecutionConfig struct {
	PreferredBackend string `toml:"preferred_backend"`
	VerboseLogging   bool   `toml:"verbose_logging"`
}

// ToolsConfig contains paths to external tools
type ToolsConfig struct {
	AbcdePath    string `toml:"abcde_path"`
	CDDiscidPath string `toml:"cd_discid_path"`
	MakeMKVPath  string `toml:"makemkv_path"`
}

// UIConfig contains user interface settings
type UIConfig struct {
	Theme       string `toml:"theme"`
	RefreshRate int    `toml:"refresh_rate"`
}

// ContainerConfig contains Docker/container settings
type ContainerConfig struct {
	Image      string `toml:"image"`
	PullPolicy string `toml:"pull_policy"`
	Enabled    bool   `toml:"enabled"`
}

// DefaultConfig returns a configuration with sensible defaults
func DefaultConfig() *Config {
	homeDir, _ := os.UserHomeDir()

	return &Config{
		Drives: DrivesConfig{
			CDDrive:    "/dev/sr0",
			AutoDetect: true,
			Available:  []string{"/dev/sr0", "/dev/sr1", "/dev/cdrom"},
		},
		Paths: PathsConfig{
			Music:   "/mnt/nas/media/music",
			Movies:  "/mnt/nas/media/movies",
			Config:  filepath.Join(homeDir, ".config", "media-ripper"),
			LogFile: filepath.Join(homeDir, "cd-ripper.log"),
		},
		CDRipping: CDRippingConfig{
			RetryCount:   3,
			RetryDelay:   5,
			InitialWait:  10,
			AutoEject:    true,
			OutputFormat: "flac",
			CDDBMethod:   "musicbrainz",
		},
		Execution: ExecutionConfig{
			PreferredBackend: "native",
			VerboseLogging:   true,
		},
		Tools: ToolsConfig{
			AbcdePath:    "",
			CDDiscidPath: "",
			MakeMKVPath:  "",
		},
		UI: UIConfig{
			Theme:       "default",
			RefreshRate: 100,
		},
		Container: ContainerConfig{
			Image:      "media-ripper:latest",
			PullPolicy: "if_not_present",
			Enabled:    false,
		},
	}
}

// Load reads configuration from a TOML file
func Load(configPath string) (*Config, error) {
	config := DefaultConfig()

	data, err := os.ReadFile(configPath)
	if err != nil {
		if os.IsNotExist(err) {
			// Return default config if file doesn't exist
			return config, nil
		}
		return nil, err
	}

	err = toml.Unmarshal(data, config)
	if err != nil {
		return nil, err
	}

	return config, nil
}

// Save writes the configuration to a TOML file
func (c *Config) Save(configPath string) error {
	// Ensure the directory exists
	dir := filepath.Dir(configPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	data, err := toml.Marshal(c)
	if err != nil {
		return err
	}

	return os.WriteFile(configPath, data, 0644)
}

// GetConfigPath returns the default configuration file path
func GetConfigPath() string {
	homeDir, _ := os.UserHomeDir()
	return filepath.Join(homeDir, ".config", "media-ripper", "config.toml")
}

// ValidationError represents a configuration validation error
type ValidationError struct {
	Field   string
	Value   any
	Message string
}

func (e ValidationError) Error() string {
	return fmt.Sprintf(
		"config validation failed for %s: %s (value: %v)",
		e.Field,
		e.Message,
		e.Value,
	)
}

// ValidationErrors represents multiple validation errors
type ValidationErrors []ValidationError

func (e ValidationErrors) Error() string {
	if len(e) == 0 {
		return "no validation errors"
	}
	if len(e) == 1 {
		return e[0].Error()
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("%d validation errors:\n", len(e)))
	for i, err := range e {
		sb.WriteString(fmt.Sprintf("  %d. %s\n", i+1, err.Error()))
	}
	return sb.String()
}

// Validate performs comprehensive validation on the configuration
func (c *Config) Validate() error {
	var errors ValidationErrors

	// Validate drives
	if err := c.validateDrives(); err != nil {
		if ve, ok := err.(ValidationErrors); ok {
			errors = append(errors, ve...)
		} else {
			errors = append(errors, ValidationError{"drives", nil, err.Error()})
		}
	}

	// Validate paths
	if err := c.validatePaths(); err != nil {
		if ve, ok := err.(ValidationErrors); ok {
			errors = append(errors, ve...)
		} else {
			errors = append(errors, ValidationError{"paths", nil, err.Error()})
		}
	}

	// Validate CD ripping settings
	if err := c.validateCDRipping(); err != nil {
		if ve, ok := err.(ValidationErrors); ok {
			errors = append(errors, ve...)
		} else {
			errors = append(errors, ValidationError{"cd_ripping", nil, err.Error()})
		}
	}

	// Validate execution settings
	if err := c.validateExecution(); err != nil {
		if ve, ok := err.(ValidationErrors); ok {
			errors = append(errors, ve...)
		} else {
			errors = append(errors, ValidationError{"execution", nil, err.Error()})
		}
	}

	// Validate tools
	if err := c.validateTools(); err != nil {
		if ve, ok := err.(ValidationErrors); ok {
			errors = append(errors, ve...)
		} else {
			errors = append(errors, ValidationError{"tools", nil, err.Error()})
		}
	}

	// Validate UI settings
	if err := c.validateUI(); err != nil {
		if ve, ok := err.(ValidationErrors); ok {
			errors = append(errors, ve...)
		} else {
			errors = append(errors, ValidationError{"ui", nil, err.Error()})
		}
	}

	if len(errors) > 0 {
		return errors
	}
	return nil
}

func (c *Config) validateDrives() error {
	var errors ValidationErrors

	// Validate CD drive path
	if c.Drives.CDDrive == "" {
		errors = append(
			errors,
			ValidationError{"drives.cd_drive", c.Drives.CDDrive, "cannot be empty"},
		)
	} else if !strings.HasPrefix(c.Drives.CDDrive, "/dev/") {
		errors = append(errors, ValidationError{"drives.cd_drive", c.Drives.CDDrive, "must be a device path starting with /dev/"})
	}

	// Check if the drive exists (if not auto-detecting)
	if !c.Drives.AutoDetect {
		if _, err := os.Stat(c.Drives.CDDrive); os.IsNotExist(err) {
			errors = append(
				errors,
				ValidationError{"drives.cd_drive", c.Drives.CDDrive, "device does not exist"},
			)
		}
	}

	// Validate available drives list
	for i, drive := range c.Drives.Available {
		if drive == "" {
			errors = append(
				errors,
				ValidationError{fmt.Sprintf("drives.available[%d]", i), drive, "cannot be empty"},
			)
		} else if !strings.HasPrefix(drive, "/dev/") {
			errors = append(errors, ValidationError{fmt.Sprintf("drives.available[%d]", i), drive, "must be a device path starting with /dev/"})
		}
	}

	if len(errors) > 0 {
		return errors
	}
	return nil
}

func (c *Config) validatePaths() error {
	var errors ValidationErrors

	// Expand and validate music directory
	if c.Paths.Music == "" {
		errors = append(errors, ValidationError{"paths.music", c.Paths.Music, "cannot be empty"})
	} else {
		expanded := expandPath(c.Paths.Music)
		if !filepath.IsAbs(expanded) {
			errors = append(errors, ValidationError{"paths.music", c.Paths.Music, "must be an absolute path"})
		}
		c.Paths.Music = expanded
	}

	// Expand and validate movies directory
	if c.Paths.Movies == "" {
		errors = append(errors, ValidationError{"paths.movies", c.Paths.Movies, "cannot be empty"})
	} else {
		expanded := expandPath(c.Paths.Movies)
		if !filepath.IsAbs(expanded) {
			errors = append(errors, ValidationError{"paths.movies", c.Paths.Movies, "must be an absolute path"})
		}
		c.Paths.Movies = expanded
	}

	// Expand and validate config directory
	if c.Paths.Config == "" {
		errors = append(errors, ValidationError{"paths.config", c.Paths.Config, "cannot be empty"})
	} else {
		expanded := expandPath(c.Paths.Config)
		if !filepath.IsAbs(expanded) {
			errors = append(errors, ValidationError{"paths.config", c.Paths.Config, "must be an absolute path"})
		}
		c.Paths.Config = expanded
	}

	// Expand and validate log file
	if c.Paths.LogFile == "" {
		errors = append(
			errors,
			ValidationError{"paths.log_file", c.Paths.LogFile, "cannot be empty"},
		)
	} else {
		expanded := expandPath(c.Paths.LogFile)
		if !filepath.IsAbs(expanded) {
			errors = append(errors, ValidationError{"paths.log_file", c.Paths.LogFile, "must be an absolute path"})
		}
		c.Paths.LogFile = expanded
	}

	if len(errors) > 0 {
		return errors
	}
	return nil
}

func (c *Config) validateCDRipping() error {
	var errors ValidationErrors

	// Validate retry count
	if c.CDRipping.RetryCount < 0 {
		errors = append(
			errors,
			ValidationError{"cd_ripping.retry_count", c.CDRipping.RetryCount, "cannot be negative"},
		)
	} else if c.CDRipping.RetryCount > 10 {
		errors = append(errors, ValidationError{"cd_ripping.retry_count", c.CDRipping.RetryCount, "cannot exceed 10"})
	}

	// Validate retry delay
	if c.CDRipping.RetryDelay < 0 {
		errors = append(
			errors,
			ValidationError{"cd_ripping.retry_delay", c.CDRipping.RetryDelay, "cannot be negative"},
		)
	} else if c.CDRipping.RetryDelay > 60 {
		errors = append(errors, ValidationError{"cd_ripping.retry_delay", c.CDRipping.RetryDelay, "cannot exceed 60 seconds"})
	}

	// Validate initial wait
	if c.CDRipping.InitialWait < 0 {
		errors = append(
			errors,
			ValidationError{
				"cd_ripping.initial_wait",
				c.CDRipping.InitialWait,
				"cannot be negative",
			},
		)
	} else if c.CDRipping.InitialWait > 120 {
		errors = append(errors, ValidationError{"cd_ripping.initial_wait", c.CDRipping.InitialWait, "cannot exceed 120 seconds"})
	}

	// Validate output format
	validFormats := []string{"flac", "mp3", "ogg", "wav"}
	if !slices.Contains(validFormats, c.CDRipping.OutputFormat) {
		errors = append(
			errors,
			ValidationError{
				"cd_ripping.output_format",
				c.CDRipping.OutputFormat,
				fmt.Sprintf("must be one of: %s", strings.Join(validFormats, ", ")),
			},
		)
	}

	// Validate CDDB method
	validMethods := []string{"musicbrainz", "cddb", "none"}
	if !slices.Contains(validMethods, c.CDRipping.CDDBMethod) {
		errors = append(
			errors,
			ValidationError{
				"cd_ripping.cddb_method",
				c.CDRipping.CDDBMethod,
				fmt.Sprintf("must be one of: %s", strings.Join(validMethods, ", ")),
			},
		)
	}

	if len(errors) > 0 {
		return errors
	}
	return nil
}

func (c *Config) validateExecution() error {
	var errors ValidationErrors

	// Validate preferred backend
	validBackends := []string{"native", "container"}
	if !slices.Contains(validBackends, c.Execution.PreferredBackend) {
		errors = append(
			errors,
			ValidationError{
				"execution.preferred_backend",
				c.Execution.PreferredBackend,
				fmt.Sprintf("must be one of: %s", strings.Join(validBackends, ", ")),
			},
		)
	}

	if len(errors) > 0 {
		return errors
	}
	return nil
}

func (c *Config) validateTools() error {
	var errors ValidationErrors

	// Auto-detect tools if paths are empty
	if c.Tools.AbcdePath == "" {
		if path, err := exec.LookPath("abcde"); err == nil {
			c.Tools.AbcdePath = path
		}
	}

	if c.Tools.CDDiscidPath == "" {
		if path, err := exec.LookPath("cd-discid"); err == nil {
			c.Tools.CDDiscidPath = path
		}
	}

	if c.Tools.MakeMKVPath == "" {
		if path, err := exec.LookPath("makemkvcon"); err == nil {
			c.Tools.MakeMKVPath = path
		}
	}

	// Validate tool paths if specified
	if c.Tools.AbcdePath != "" {
		if _, err := os.Stat(c.Tools.AbcdePath); os.IsNotExist(err) {
			errors = append(
				errors,
				ValidationError{"tools.abcde_path", c.Tools.AbcdePath, "file does not exist"},
			)
		} else if !isExecutable(c.Tools.AbcdePath) {
			errors = append(errors, ValidationError{"tools.abcde_path", c.Tools.AbcdePath, "file is not executable"})
		}
	}

	if c.Tools.CDDiscidPath != "" {
		if _, err := os.Stat(c.Tools.CDDiscidPath); os.IsNotExist(err) {
			errors = append(
				errors,
				ValidationError{
					"tools.cd_discid_path",
					c.Tools.CDDiscidPath,
					"file does not exist",
				},
			)
		} else if !isExecutable(c.Tools.CDDiscidPath) {
			errors = append(errors, ValidationError{"tools.cd_discid_path", c.Tools.CDDiscidPath, "file is not executable"})
		}
	}

	if c.Tools.MakeMKVPath != "" {
		if _, err := os.Stat(c.Tools.MakeMKVPath); os.IsNotExist(err) {
			errors = append(
				errors,
				ValidationError{"tools.makemkv_path", c.Tools.MakeMKVPath, "file does not exist"},
			)
		} else if !isExecutable(c.Tools.MakeMKVPath) {
			errors = append(errors, ValidationError{"tools.makemkv_path", c.Tools.MakeMKVPath, "file is not executable"})
		}
	}

	if len(errors) > 0 {
		return errors
	}
	return nil
}

func (c *Config) validateUI() error {
	var errors ValidationErrors

	// Validate theme
	if c.UI.Theme == "" {
		errors = append(errors, ValidationError{"ui.theme", c.UI.Theme, "cannot be empty"})
	}

	// Validate refresh rate
	if c.UI.RefreshRate < 50 {
		errors = append(
			errors,
			ValidationError{"ui.refresh_rate", c.UI.RefreshRate, "cannot be less than 50ms"},
		)
	} else if c.UI.RefreshRate > 1000 {
		errors = append(errors, ValidationError{"ui.refresh_rate", c.UI.RefreshRate, "cannot exceed 1000ms"})
	}

	if len(errors) > 0 {
		return errors
	}
	return nil
}

// expandPath expands ~ to home directory
func expandPath(path string) string {
	if !strings.HasPrefix(path, "~/") {
		return path
	}

	homeDir, err := os.UserHomeDir()
	if err != nil {
		return path
	}

	return filepath.Join(homeDir, path[2:])
}

// isExecutable checks if a file is executable
func isExecutable(path string) bool {
	info, err := os.Stat(path)
	if err != nil {
		return false
	}
	return info.Mode()&0111 != 0
}

// LoadAndValidate loads configuration from file and validates it
func LoadAndValidate(configPath string) (*Config, error) {
	config, err := Load(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to load config: %w", err)
	}

	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("config validation failed: %w", err)
	}

	return config, nil
}

