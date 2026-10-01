# `doppel clean`

`clean` scans for duplicate files, selects one file to keep from each duplicate group, and applies an explicitly selected action to the remaining files. It is separate from `find` because it can modify the filesystem. A keep policy is required; deletion is never based on an implicit choice.

## Usage

```sh
doppel clean [options] [directories...]
doppel clean --files [options] <files...>
doppel clean --files-from <file|-> [options]
```

If no directory is supplied, the current working directory is scanned. Directory scans recurse unless `--max-depth` is set. Clean uses the same file discovery, filtering, size, symlink, and hard-link behavior as `find`; see [find.md](./find.md) for additional detail on glob syntax and size formats.

## Scan and input options

| Option | Default | Description |
| --- | --- | --- |
| `-w, --workers <n>` | Number of CPUs | Number of worker goroutines used for hashing. Must be at least 1. |
| `-v, --verbose` | Off | Write detailed scan progress to stderr. |
| `--quiet` | Off | Suppress scan progress and informational messages. Action results are still written. |
| `--files` | Off | Treat positional arguments as an explicit list of regular files. Scan filters and depth limits are bypassed. |
| `--files-from <file>` | None | Read explicit paths from a file, or use `-` for stdin. Scan filters and depth limits are bypassed. |
| `--null` | Off | Read NUL-delimited records from `--files-from` instead of newline-delimited records. |
| `--ignore-empty-paths` | Off | Ignore empty or whitespace-only records in `--files-from`; otherwise they are errors. |
| `--ignore-hardlinks` | Off | Treat paths to the same underlying file as one candidate. By default, hard-linked paths are separate candidates. |
| `--max-depth <n>` | Unlimited | Limit directory traversal. The root and files directly inside it are depth 0; files in a child directory are depth 1. Must be non-negative. |

## Filters

| Option | Default | Description |
| --- | --- | --- |
| `--exclude <patterns>` | None | Comma-separated glob patterns that exclude both files and directories, matching basenames and full walked paths. |
| `--exclude-dirs <patterns>` | None | Comma-separated directory glob patterns, matching directory names or full paths. Matching directories are pruned. |
| `--exclude-files <patterns>` | None | Comma-separated file glob patterns, matching file names or full paths. |
| `--min-size <size>` | No minimum | Ignore files smaller than the given size. |
| `--max-size <size>` | No maximum | Ignore files larger than the given size. |

`--exclude` is mutually exclusive with `--exclude-dirs` and `--exclude-files`. This applies whether the values come from command-line flags or configuration. Glob syntax follows Go's `filepath.Match`; patterns are not regular expressions, `*` does not cross path separators, and shell patterns should be quoted. Comma-separated patterns cannot contain a literal comma.

Negative size values are treated as no limit. If both size bounds are positive, the minimum must not exceed the maximum; equal bounds select only files of that exact size. Decimal (`MB`) and binary (`MiB`) suffixes are supported, case-insensitively.

Explicit-file modes bypass filters and depth, including exclusions and size limits. Paths are still validated: each must exist and be a regular file. Explicit symlink paths are rejected; symlinks discovered during a directory scan are ignored. `--ignore-hardlinks` continues to apply to explicit lists. Exclusion-mode mutual-exclusion validation still runs in explicit-file mode.

## Action and retention options

| Option | Default | Description |
| --- | --- | --- |
| `--keep <policy>` | Required | Select which file to retain from each duplicate group: `newest`, `oldest`, `shortest-path`, or `first`. |
| `--mode <mode>` | `delete` | Action for each non-keeper: `delete`, `trash`, or `replace-with-hardlink`. |
| `--dry-run` | Off | Report selected actions without changing files. Does not require confirmation. |
| `-y, --yes` | Off | Explicitly approve the action and skip interactive confirmation. |

Retention policies:

- `newest`: keep the file with the latest modification time.
- `oldest`: keep the file with the earliest modification time.
- `shortest-path`: keep the file with the shortest cleaned path.
- `first`: keep the lexically first path.

Paths are considered in lexical order to make ties deterministic. Every other file in the duplicate group becomes an action target.

Action modes:

- `delete`: permanently remove each selected duplicate with `os.Remove`.
- `trash`: move each selected duplicate to the user's XDG trash directory on Linux. `XDG_DATA_HOME/Trash` is used when set; otherwise the location is `~/.local/share/Trash`. This mode is unsupported on other platforms and fails rather than falling back to deletion.
- `replace-with-hardlink`: replace each selected duplicate with a hard link to its keeper. The keeper and target must be on a filesystem that supports hard links; cross-filesystem or unsupported operations fail.

Before acting, clean rechecks targets with `Lstat`, requires regular files, and skips mutation when keeper and target already refer to the same underlying file. If any action fails, clean reports the failures and returns an error after processing the selected targets.

## Confirmation and incompatible options

If there are targets and neither `--dry-run` nor `--yes` is set, clean asks for interactive confirmation. When confirmation is required, a non-interactive run must pass `--yes`. Use `--dry-run` first to inspect the keeper/removal choices and action mode.

The following combinations are invalid:

- `--quiet` with `--verbose`.
- `--exclude` with either `--exclude-dirs` or `--exclude-files`.
- `--files-from` with `--files`.
- `--files-from` with positional arguments.
- `--null` without `--files-from`.

`--dry-run` and `--yes` are allowed together, though `--yes` has no effect when no mutation will occur. Exclusion and size flags can appear in explicit-file mode but are ignored there, except that conflicting exclusion modes are still rejected.

## Configuration and global options

The global options are supplied before the subcommand:

| Option | Description |
| --- | --- |
| `--config <path>` | Load a TOML, YAML, or JSON configuration file. `--config=none` or `--config=ignore` bypasses files and environment variables. |
| `--log-level <level>` | Set logging level: `debug`, `info`, `warn`, or `error`. |
| `--log-format <format>` | Set logging format: `text`, `json`, `pretty`, or `discard`. |
| `--log-output <destination>` | Set logging output to `stdout`, `stderr`, `null`, or a file path. |
| `-h, --help` | Display command or application help. |

Clean settings can be configured in the `[clean]` table and with `DOPPEL_CLEAN_*` environment variables. Available keys are `workers`, `verbose`, `quiet`, `ignore_hardlinks`, `exclude`, `max_depth`, `exclude_dirs`, `exclude_files`, `min_size`, `max_size`, `dry_run`, `mode`, and `keep`. Clean-specific scan values override corresponding `[find]` values; `[find]` remains the fallback for shared scan settings. Precedence is command line, environment, configuration file, then built-in defaults.

File-list inputs (`files`, `files_from`, `null`, and `ignore_empty_paths`) are invocation-specific and cannot be set in configuration. `--yes` is also command-line-only so configuration cannot silently approve destructive actions. `dry_run` can safely be enabled as a default; pass `--dry-run=false` to override it for one invocation.

## Examples

Preview keeping the shortest path before deleting anything:

```sh
doppel clean ~/Downloads --keep shortest-path --dry-run
```

Move duplicate files to Linux trash after explicit approval:

```sh
doppel clean ~/Downloads --keep newest --mode trash --yes
```

Replace duplicate paths with hard links to the lexically first keeper:

```sh
doppel clean ~/Downloads --keep first --mode replace-with-hardlink --yes
```

Scan a NUL-delimited explicit file list and preview the selected actions:

```sh
find . -type f -print0 | doppel clean --files-from=- --null --keep first --dry-run
```
