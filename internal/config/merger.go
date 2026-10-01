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
	mergeFindConfig(&result, override)

	// Merge clean config
	mergeCleanConfig(&result, override)

	return &result
}

func mergeFindConfig(result, override *Config) {
	if override.Find.Exclude != "" {
		result.Find.Exclude = override.Find.Exclude
	}
	if override.Find.MaxDepth != nil {
		maxDepth := *override.Find.MaxDepth
		result.Find.MaxDepth = &maxDepth
	}
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
	if override.Find.MinSize != "" {
		result.Find.MinSize = override.Find.MinSize
	}
	if override.Find.MaxSize != "" {
		result.Find.MaxSize = override.Find.MaxSize
	}
	if override.Find.Sort != "" {
		result.Find.Sort = override.Find.Sort
	}
	if override.Find.SortReverse {
		result.Find.SortReverse = true
	}
	if override.Find.PathsOnly {
		result.Find.PathsOnly = true
	}
	if override.Find.Print0 {
		result.Find.Print0 = true
	}
	if override.Find.Quiet {
		result.Find.Quiet = true
	}
	if override.Find.FailOnDuplicates {
		result.Find.FailOnDuplicates = true
	}
	if override.Find.ShowFilters {
		result.Find.ShowFilters = override.Find.ShowFilters
	}
	if override.Find.IgnoreHardlinks {
		result.Find.IgnoreHardlinks = override.Find.IgnoreHardlinks
	}
	if override.Find.OutputFormat != "" {
		result.Find.OutputFormat = override.Find.OutputFormat
	}
	if override.Find.OutputFile != "" {
		result.Find.OutputFile = override.Find.OutputFile
	}
}

func mergeCleanConfig(result, override *Config) {
	if override.Clean.Workers != 0 {
		result.Clean.Workers = override.Clean.Workers
	}
	if override.Clean.Verbose {
		result.Clean.Verbose = true
	}
	if override.Clean.Quiet {
		result.Clean.Quiet = true
	}
	if override.Clean.Exclude != "" {
		result.Clean.Exclude = override.Clean.Exclude
	}
	if override.Clean.MaxDepth != nil {
		maxDepth := *override.Clean.MaxDepth
		result.Clean.MaxDepth = &maxDepth
	}
	if override.Clean.ExcludeDirs != "" {
		result.Clean.ExcludeDirs = override.Clean.ExcludeDirs
	}
	if override.Clean.ExcludeFiles != "" {
		result.Clean.ExcludeFiles = override.Clean.ExcludeFiles
	}
	if override.Clean.MinSize != "" {
		result.Clean.MinSize = override.Clean.MinSize
	}
	if override.Clean.MaxSize != "" {
		result.Clean.MaxSize = override.Clean.MaxSize
	}
	if override.Clean.IgnoreHardlinks {
		result.Clean.IgnoreHardlinks = true
	}
	if override.Clean.DryRun {
		result.Clean.DryRun = true
	}
	if override.Clean.Mode != "" {
		result.Clean.Mode = override.Clean.Mode
	}
	if override.Clean.Keep != "" {
		result.Clean.Keep = override.Clean.Keep
	}
}
