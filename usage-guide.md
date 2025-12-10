Skill Porter – brief usage guide

## Overview

Skill Porter is a CLI and cross-platform helper that converts between:

* Claude Code skills (SKILL.md + .claude-plugin/marketplace.json)
* Gemini CLI extensions (gemini-extension.json + GEMINI.md)

It can also create universal projects that support both platforms and optionally wire up forks and PRs for dual-platform support.

---

## Prerequisites

* Node.js ≥ 18 (enforced via `engines.node` in package.json)
* Git for repo/PR features
* GitHub CLI `gh` for `create-pr` (must be installed and authenticated)

---

## Install & Run

From a clone of the repository:

```bash
git clone https://github.com/jduncan-rva/skill-porter
cd skill-porter
npm install
```

Run as a CLI:

```bash
node src/cli.js --help
# or, after linking/installing as a binary:
skill-porter --help
```

Binary name: `skill-porter` (defined in package.json `"bin"`)

---

## Core Commands

All commands accept a path to a skill/extension directory (containing SKILL.md or gemini-extension.json).

### 1. Analyze platform type

```bash
skill-porter analyze ./my-skill
```

* Detects whether the directory is `claude`, `gemini`, `universal`, or `unknown`.
* Prints platform, confidence, and which key files were found (SKILL.md, gemini-extension.json, etc.).

### 2. Convert between platforms

```bash
# Claude → Gemini
skill-porter convert ./my-claude-skill --to gemini

# Gemini → Claude
skill-porter convert ./my-gemini-extension --to claude
```

Options:

* `--to <platform>`: `claude` or `gemini` (default: `gemini`)
* `--output <path>`: write converted files into another directory
* `--no-validate`: skip post-conversion validation

What happens:

* Platform detection (Claude / Gemini / Universal)
* Metadata conversion:

  * YAML frontmatter ↔ JSON manifest
  * `allowed-tools` ↔ `excludeTools`
  * env-vars ↔ `settings` schema
* MCP servers adjusted:

  * Claude paths → `${extensionPath}/…` for Gemini
  * Gemini `${extensionPath}` → relative paths for Claude
* Generates:

  * For Gemini target: `gemini-extension.json`, `GEMINI.md`, `commands/*.toml`, `docs/GEMINI_ARCHITECTURE.md`, `shared/` docs
  * For Claude target: `SKILL.md`, `.claude-plugin/marketplace.json`, `.claude/commands/*.md`, `shared/` docs

On success, the CLI prints “Next steps” with install commands for the target platform.

### 3. Validate a skill/extension

```bash
skill-porter validate ./my-project
skill-porter validate ./my-project --platform claude
skill-porter validate ./my-project --platform gemini
skill-porter validate ./my-project --platform universal
```

Checks:

* Required files (SKILL.md, gemini-extension.json, etc.)
* YAML/JSON correctness
* Basic frontmatter and manifest constraints
* MCP server config sanity (e.g., `${extensionPath}` usage)
* `settings` and `excludeTools` structure

### 4. Create a universal project (both platforms)

```bash
skill-porter universal ./my-project
# or specify an output directory
skill-porter universal ./my-project --output ./my-universal-project
```

Behavior:

* Detects current platform
* Generates missing side (Claude or Gemini) into `output` while keeping original
* Returns a directory that has both SKILL.md and gemini-extension.json (plus shared docs).

---

## Advanced Commands

### 5. Generate a PR adding cross-platform support

```bash
skill-porter create-pr ./my-repo --to gemini
skill-porter create-pr ./my-repo --to claude
```

Options:

* `--to <platform>`: target platform to add (`gemini` or `claude`)
* `--base <branch>`: base branch (default `main`)
* `--remote <name>`: git remote (default `origin`)
* `--draft`: open PR as draft

Flow:

* Runs conversion into the repo working tree
* Creates or reuses branch `skill-porter/add-dual-platform-support`
* Commits generated files
* Pushes branch and opens a GitHub PR via `gh pr create` with a prefilled body.

### 6. Fork-style dual-platform setup

```bash
skill-porter fork ./my-project --location ./fork-dir
# optionally:
skill-porter fork ./my-project --location ./fork-dir --url https://github.com/user/repo.git
```

Behavior:

* Copies or clones into `--location`
* Ensures both Claude and Gemini configs exist (invokes converter if needed)
* Tries to symlink the fork into `~/.claude/skills/<name>`
* Prints a Gemini install command like `gemini extensions install <forkPath>`.

---

## Typical workflows

* Port a Claude skill to Gemini:

  ```bash
  skill-porter convert ./my-claude-skill --to gemini
  # then:
  gemini extensions install ./my-claude-skill
  ```

* Port a Gemini extension to Claude:

  ```bash
  skill-porter convert ./my-gemini-extension --to claude
  # then:
  cp -r ./my-gemini-extension ~/.claude/skills/
  ```

* Turn a single-platform repo into a universal Claude+Gemini project:

  ```bash
  skill-porter universal ./my-project --output ./my-project-universal
  skill-porter validate ./my-project-universal --platform universal
  ```
