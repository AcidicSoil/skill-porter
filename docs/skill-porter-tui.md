# Skill Porter TUI

A Text User Interface (TUI) for the Skill Porter tool, built with Bubble Tea.

## Overview

Skill Porter TUI allows you to interactively scan for, review, and convert skills between Claude Code and Gemini CLI formats. It provides a visual interface to manage bulk conversions and monitor progress.

## Installation

```bash
# From source
go install ./cmd/skill-porter-tui
```

## Usage

Run the tool from your terminal:

```bash
skill-porter-tui [flags]
```

### Flags

| Flag | Description | Default |
|------|-------------|---------|
| `--root <path>` | Root directory to scan for skills | Current directory |
| `--recursive` | Scan directories recursively | `true` |
| `--target <gemini|claude|auto>` | Default conversion target | `auto` |
| `--out <path>` | Base directory for output | In-place |
| `--auto` | Enable auto-convert mode (convert all pending immediately) | `false` |
| `--debug` | Enable debug logging to debug.log | `false` |

## Keybindings

| Key | Action |
|-----|--------|
| `↑` / `k` | Move cursor up |
| `↓` / `j` | Move cursor down |
| `c` | Convert selected skill (using default/auto target) |
| `g` | Force convert selected skill to **Gemini** |
| `a` | Force convert selected skill to **Claude** |
| `A` | Auto-convert all pending skills |
| `r` | Rescan directory |
| `q` / `ctrl+c` | Quit |

## Interface

The interface is split into two main sections:

1.  **Skill List (Left)**: Displays discovered skills, their current platform, and conversion status.
2.  **Details Panel (Right)**: Shows detailed information for the selected skill, including paths and conversion logs/errors.

### Status Indicators

- **Pending**: Ready for conversion (Grey)
- **Running**: Conversion in progress (Orange)
- **Success**: Conversion completed successfully (Green)
- **Failed**: Conversion failed (Red)

## Troubleshooting

Logs are written to `debug.log` in the current directory. Use `--debug` for verbose output.
