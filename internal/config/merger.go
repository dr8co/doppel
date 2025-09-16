package config

// defaultMerger provides deep merging of configurations.
type defaultMerger struct{}

// Merge merges two configurations.
// The base values are only overwritten if the override values are non-empty.
func (m *defaultMerger) Merge(base, override *Config) *Config {
	result := *base // Copy base

	// Merge log config
	if override.Log.Level != "" {
		result.Log.Level = override.Log.Level
	}
	if override.Log.Format != "" {
		result.Log.Format = override.Log.Format
	}
	if override.Log.Output != "" {
		result.Log.Output = override.Log.Output
	}

	// Merge find config
	if override.Find.Workers != 0 {
		result.Find.Workers = override.Find.Workers
	}
	if override.Find.Verbose {
		result.Find.Verbose = override.Find.Verbose
	}
	if override.Find.ExcludeDirs != "" {
		result.Find.ExcludeDirs = override.Find.ExcludeDirs
	}
	if override.Find.ExcludeFiles != "" {
		result.Find.ExcludeFiles = override.Find.ExcludeFiles
	}
	if override.Find.ExcludeDirRegex != "" {
		result.Find.ExcludeDirRegex = override.Find.ExcludeDirRegex
	}
	if override.Find.ExcludeFileRegex != "" {
		result.Find.ExcludeFileRegex = override.Find.ExcludeFileRegex
	}
	if override.Find.MinSize != "" {
		result.Find.MinSize = override.Find.MinSize
	}
	if override.Find.MaxSize != "" {
		result.Find.MaxSize = override.Find.MaxSize
	}
	if override.Find.ShowFilters {
		result.Find.ShowFilters = override.Find.ShowFilters
	}
	if override.Find.OutputFormat != "" {
		result.Find.OutputFormat = override.Find.OutputFormat
	}
	if override.Find.OutputFile != "" {
		result.Find.OutputFile = override.Find.OutputFile
	}

	// Merge preset config
	if override.Preset.Workers != 0 {
		result.Preset.Workers = override.Preset.Workers
	}
	if override.Preset.Verbose {
		result.Preset.Verbose = override.Preset.Verbose
	}
	if override.Preset.ShowFilters {
		result.Preset.ShowFilters = override.Preset.ShowFilters
	}
	if override.Preset.OutputFormat != "" {
		result.Preset.OutputFormat = override.Preset.OutputFormat
	}
	if override.Preset.OutputFile != "" {
		result.Preset.OutputFile = override.Preset.OutputFile
	}

	return &result
}
