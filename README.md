# Moco

Moco is a research experiment management tool designed to ensure reproducibility by tracking Git repository state, capturing command outputs, and documenting execution details.

## Overview

When conducting computational experiments, reproducing results can be challenging due to code changes, different environments, or undocumented parameters. Moco solves this by:

- Creating isolated experiment directories with unique identifiers
- Recording Git metadata (branch, commit hash, status)
- Capturing command outputs (stdout/stderr)
- Generating comprehensive experiment summaries
- Organizing experiments for easy reference and comparison

## Installation

### From Source

```bash
# Clone the repository
git clone https://github.com/bicycle1885/moco.git
cd moco

# Build the binary
go build -o moco

# Optional: Move to your PATH
cp moco /usr/local/bin/
```

## Commands

### Run an Experiment

```bash
moco run [command]
```

This will:
1. Create an experiment directory with timestamp, branch name, and Git commit hash
2. Record Git status and system information
3. Run the command, capturing outputs
4. Generate a summary of the experiment

Options:
- `-f, --force` - Allow experiments with uncommitted Git changes
- `-d, --dir` - Specify base directory for experiment output
- `-n, --no-pushd` - Execute command in current directory
- `-c, --cleanup-on-fail` - Remove experiment directory if command fails
- `--dev` - Development mode (enables --force and --cleanup-on-fail)

### List Experiments

```bash
moco list
moco ls  # Alias
```

Options:
- `-f, --format` - Output format (table, json, csv)
- `-s, --sort` - Sort by (date, branch, status, duration)
- `-r, --reverse` - Reverse sort order
- `-b, --branch` - Filter by branch name
- `--status` - Filter by status (success, failure, running)
- `--since` - Filter by date (e.g., '7d' for last 7 days)
- `-c, --command` - Filter by command pattern (regex)
- `-n, --limit` - Limit number of results

### Show Project Status

```bash
moco status
moco st  # Alias
```

Options:
- `-d, --detail` - Level of detail (minimal, normal, full)
- `-f, --format` - Output format (text, json, markdown)

### Archive Experiments

```bash
moco archive
```

Options:
- `-o, --older-than` - Archive experiments older than duration (e.g., '30d')
- `-s, --status` - Archive by status (success, failure, running, all)
- `-f, --format` - Archive format (zip, tar.gz)
- `-d, --destination` - Archive destination directory
- `--delete` - Remove original directories after archiving
- `--dry-run` - Show what would be archived without executing

## Configuration

Moco can be configured using a `.moco.toml` file in your project directory or in your user config directory.

Example configuration:

```toml
# .moco.toml

[paths]
base_dir = "runs"
summary_file = "Summary.md"
stdout_file = "stdout.log"
stderr_file = "stderr.log"

[run]
default_force = false
default_cleanup_on_fail = false
default_no_pushd = false

[git]
require_clean = true

[archive]
format = "tar.gz"
older_than = "30d"
```

## Example Workflow

```bash
# Run an experiment
moco run python train.py --epochs 100 --batch-size 64

# List recent experiments
moco list --since 7d

# Show detailed status of one experiment
moco status --detail full

# Archive old successful experiments
moco archive --older-than 30d --status success
```

## Experiment Directory Structure

Each experiment is stored in a directory with the following format:
`runs/YYYY-MM-DDTHH:MM:SS.sss_branch_commithash/`

Inside each directory:
- `Summary.md` - Metadata and results
- `stdout.log` - Standard output
- `stderr.log` - Standard error

## Why Use Moco?

- **Reproducibility**: Track exactly what code, command, and environment produced a result
- **Organization**: Keep experiments neatly organized with consistent metadata
- **Transparency**: Document experiment details automatically
- **Efficiency**: Spend less time documenting and more time researching

## License

The MIT License
