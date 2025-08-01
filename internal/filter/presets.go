package filter

// GetPresetConfig returns a predefined [Config] based on the preset name
func GetPresetConfig(preset string) *Config {
	switch preset {
	case "dev":
		return &Config{
			ExcludeDirs:  []string{"node_modules", ".git", "build", "dist", "target", "__pycache__", ".vscode", ".idea", "vendor"},
			ExcludeFiles: []string{"*.tmp", "*.log", "*.swp", "*.swo", "*~", ".DS_Store", "Thumbs.db", "*.pyc", "*.pyo"},
			MinSize:      100, // Skip very small files
		}
	case "media":
		return &Config{
			ExcludeDirs: []string{".git", "__pycache__", "node_modules"},
			MinSize:     10240, // 10KB minimum for media files
		}
	case "docs":
		return &Config{
			ExcludeDirs:  []string{".git", "__pycache__", "node_modules", "build", "dist"},
			ExcludeFiles: []string{"*.tmp", "*.log", "*.swp", "*~"},
			MinSize:      1024, // 1KB minimum
		}
	case "clean":
		return &Config{
			ExcludeDirs:  []string{".git", "__pycache__", "node_modules", ".cache", "tmp", "temp"},
			ExcludeFiles: []string{"*.tmp", "*.log", "*.cache", "*.swp", "*~", ".DS_Store", "Thumbs.db"},
		}
	default:
		return &Config{}
	}
}
