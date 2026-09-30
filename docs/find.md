# `doppel find`

`find` scans directories or an explicit list of regular files and reports groups of files with identical contents. Directory scans apply the configured filters before hashing. If no directory is supplied, the current working directory is scanned.

The aliases are `doppel search` and `doppel f`. Running `doppel` with no subcommand also runs `find`.

## Usage

```sh
doppel find [options] [directories...]
doppel find --files [options] <files...>
doppel find --files-from <file|-> [options]
```

Directory scans walk each supplied root recursively, skip symlinks, group candidate files by size, and compare their hashes. The scan is read-only.

## Options

### Scan and input options

| Option | Default | Description |
| --- | --- | --- |
| `-w, --workers <n>` | Number of CPUs | Number of worker goroutines used for hashing. Must be at least 1. |
| `-v, --verbose` | Off | Write detailed scan and hashing progress to stderr. |
| `--quiet` | Off | Suppress progress and informational messages; the report is still written. |
| `--files` | Off | Treat positional arguments as an explicit list of regular files. All scan filters and depth limits are bypassed. |
| `--files-from <file>` | None | Read explicit paths from a file. Use `-` to read from stdin. This also bypasses scan filters and depth limits. |
| `--null` | Off | Read NUL-delimited records from `--files-from` instead of newline-delimited records. |
| `--ignore-empty-paths` | Off | Ignore empty or whitespace-only records in `--files-from`; otherwise an empty record is an error. |
| `--ignore-hardlinks` | Off | Treat paths to the same underlying file as one candidate. By default, hard-linked paths are separate candidates. |
| `--max-depth <n>` | Unlimited | Limit directory traversal. The root and files directly inside it are depth 0; files in a child directory are depth 1. Must be non-negative. |

### Filters

| Option | Default | Description |
| --- | --- | --- |
| `--exclude <patterns>` | None | Comma-separated glob patterns that exclude both files and directories. Each pattern is matched against an entry's basename and its full walked path. |
| `--exclude-dirs <patterns>` | None | Comma-separated glob patterns for directory names or full paths. Alias: `--skip-dirs`. Matching directories are pruned from traversal. |
| `--exclude-files <patterns>` | None | Comma-separated glob patterns for file names or full paths. Alias: `--skip-files`. |
| `--min-size <size>` | No minimum | Ignore files smaller than the given size. |
| `--max-size <size>` | No maximum | Ignore files larger than the given size. |
| `--show-filters` | Off | Display active filters and exit without scanning a directory tree. This does not display filters in explicit-file mode. |

Glob patterns use Go's `filepath.Match` syntax, not regular expressions. `*` matches characters within one path component; it does not recursively cross directory separators. Quote patterns so the shell does not expand them before `doppel` receives them. Patterns are comma-separated, so a literal comma cannot be represented within one pattern.

`--exclude` is mutually exclusive with `--exclude-dirs` and `--exclude-files`. This applies to values loaded from configuration as well as command-line values. Choose the unified option or the specialized glob options for one invocation.

Size suffixes are case-insensitive. Plain numbers are bytes; decimal units include `KB`, `MB`, `GB`, and `TB`, and binary units include `KiB`, `MiB`, `GiB`, and `TiB`. A negative size is treated as no limit. When both positive limits are set, the minimum cannot exceed the maximum; equal limits include only files of exactly that size.

### Output options

| Option | Default | Description |
| --- | --- | --- |
| `--output-format <format>` | `pretty` | Format the report as `pretty`, `json`, `jsonl`, or `yaml`. `jsonl` emits one duplicate-group object per line. |
| `--output-file <destination>` | stdout | Write the report to a file. The values `stdout` and `stderr` select those streams. Parent directories are created as needed. |
| `--sort <mode>` | `path` | Order duplicate groups by `path`, `size`, `wasted-space`, or `count`. Paths within every group are sorted lexically. |
| `--reverse` | Off | Reverse the selected sort order. |
| `--paths-only` | Off | Emit every path in each duplicate group instead of a formatted report. |
| `--print0` | Off | Terminate each path with NUL instead of newline. |
| `--fail-on-duplicates` | Off | Return a nonzero status if at least one duplicate group is found. Output is still written. |

`--print0` requires `--paths-only`. `--paths-only` requires the effective output format to be `pretty`; it cannot be combined with a non-pretty `--output-format` or a non-pretty output format loaded from configuration. `--quiet` cannot be combined with `--verbose` (including verbose mode enabled through configuration).

## Explicit file lists

`--files` accepts positional files, while `--files-from` reads one path per line by default. `--files-from=-` reads from stdin. Every path must exist and refer to a regular file. Invalid paths fail the command instead of being skipped. Explicit symlink paths are rejected; symlinks encountered during directory scans are ignored.

For `--files-from`, records are trimmed. Empty records fail unless `--ignore-empty-paths` is set. With `--null`, records are NUL-delimited, which permits newline characters inside a path.

At least one file is required in explicit-file mode.

The following combinations are invalid:

- `--files` with `--files-from`
- `--files-from` with positional arguments
- `--null` without `--files-from`
- `--files-from` without a path or `-`

`--null` is specifically an input delimiter option. For NUL-delimited output, use `--paths-only --print0`.

Filters and depth options are bypassed in explicit-file mode, including `--exclude`, `--exclude-dirs`, `--exclude-files`, `--min-size`, `--max-size`, and `--max-depth`. `--ignore-hardlinks` still applies to explicit files. The mutual-exclusion validation for exclusion options still applies even when explicit-file mode is selected.

## Common global options

These application-level options are supplied before the subcommand:

| Option | Description |
| --- | --- |
| `--config <path>` | Load a TOML, YAML, or JSON configuration file. `--config=none` or `--config=ignore` bypasses configuration files and environment variables. |
| `--log-level <level>` | Set logging level: `debug`, `info`, `warn`, or `error`. |
| `--log-format <format>` | Set logging format: `text`, `json`, `pretty`, or `discard`. |
| `--log-output <destination>` | Set logging output to `stdout`, `stderr`, `null`, or a file path. |
| `-h, --help` | Display command or application help. |

Configuration values can provide find defaults. Precedence is command line, environment, configuration file, then built-in defaults. Find settings use the `[find]` table in configuration files and the `DOPPEL_FIND_` environment prefix where supported. Examples include `exclude`, `max_depth`, `exclude_dirs`, `exclude_files`, `min_size`, `max_size`, `workers`, and `ignore_hardlinks`.

## Examples

Scan the current directory, or specific roots:

```sh
doppel find
doppel find ~/Downloads ~/Documents
```

Limit recursion to the root and its direct child directories:

```sh
doppel find --max-depth 1 ~/Downloads
```

Use one unified exclusion mode, or the specialized glob mode:

```sh
doppel find --exclude '*.log,node_modules' ~/Downloads
doppel find --exclude-dirs '.git,node_modules' --exclude-files '*.tmp,*.log' ~/Downloads
```

Read file names safely from another command:

```sh
find . -type f -print0 | doppel find --files-from=- --null
```

Write machine-readable output, or stream NUL-terminated duplicate paths:

```sh
doppel find --quiet --output-format jsonl ~/Downloads
doppel find --quiet --paths-only --print0 ~/Downloads | xargs -0 -n1 printf '%s\\n'
```
