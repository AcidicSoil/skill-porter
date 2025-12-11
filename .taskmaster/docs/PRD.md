1. Overview
   ===========

Problem Statement
The existing `convert-all-skills.sh` script provides a gum-based interactive flow to recursively discover skills and invoke `skill-porter convert`, but the UX is linear, text-heavy, and lacks a coherent TUI layout, progress overview, or easy error inspection. It is difficult to see all skills, their statuses, and results in one place, and the behavior is tied tightly to interactive shell prompts.

Target Users

* Internal maintainers of the `skill-porter` tool who frequently convert many skills between Claude and Gemini formats.
* External advanced users working with large trees of skills/extensions who need repeatable, inspectable conversion runs.
* CI / tooling authors who may later want to script or wrap the TUI for semi-automated batch conversions.

Success Metrics

* Functional parity: all flows supported by `convert-all-skills.sh` (recursive vs single, per-skill target selection, auto-convert remaining, skip, quit, summary) are available or clearly replaced by equivalent TUI flows.
* UX improvement: users can see a structured list of discovered skills, each with status (pending, running, success, skipped, failed), and a persistent summary bar (success/skipped/failed/total).
* Reliability: conversion failure modes are clearly surfaced per-skill (exit codes and error snippets) and via process exit code (non-zero if any failures).
* Adoption: for internal workflows, the TUI becomes the default interactive entrypoint; the existing script remains usable and unchanged in surface area.
* Quality: no “god modules,” no ad-hoc “utils,” clear module boundaries, and code structured for long-term maintenance.

2. Capability Tree (Functional Decomposition)
   =============================================

### Capability: Configuration & Startup

Covers how users launch the TUI, set scan parameters, and wire flags/env into the application model.

#### Feature: CLI Flag and Environment Parsing [MVP]

* **Description**: Parse CLI flags and environment variables into a validated configuration struct for the TUI.
* **Inputs**: Command-line args (`--root`, `--recursive/--no-recursive`, `--target`, `--out-base`), environment variables (e.g. `SKILL_PORTER_ROOT`, `SKILL_PORTER_TARGET`).
* **Outputs**: `AppConfig` struct with fields: `ScanRoot`, `RecursiveMode`, `DefaultTarget`, `OutBaseDir`, `AutoConvertMode` (off by default).
* **Behavior**:

  * Merge env defaults, CLI flags, and internal defaults (`./skills`, `gemini` target, `./converted-<target>-skills`) using the same defaults as `convert-all-skills.sh`.
  * Validate directories and normalize paths (absolute or clean relative).
  * On validation error, print clear error and exit with non-zero code.

#### Feature: Initial Configuration Screen [MVP]

* **Description**: Bubble Tea screen to confirm or adjust initial configuration before scanning.
* **Inputs**: `AppConfig` from flag/env parsing.
* **Outputs**: Possibly-updated `AppConfig` (e.g. user edits scan root, toggles recursive flag, selects default target, changes output base).
* **Behavior**:

  * Render header with title and brief context (“Skill Porter Conversion”).
  * Render editable fields: scan root, recursive yes/no, default target (Gemini/Claude), output base dir.
  * Provide key hints for editing (e.g. `Tab` to move fields, `Enter` to accept).
  * Confirm configuration before triggering discovery; on cancel, exit cleanly.

#### Feature: Configuration Persistence (Optional / non-MVP)

* **Description**: Optional ability to persist last-used configuration to a small config file.
* **Inputs**: Final `AppConfig` at end of a successful run.
* **Outputs**: On-disk config file (e.g. `$XDG_CONFIG_HOME/skill-porter-tui/config.json`), plus persisted defaults on next run.
* **Behavior**:

  * On startup, attempt to load previous config; fall back to built-in defaults if missing/invalid.
  * Never block startup on config errors; log and continue with defaults.

### Capability: Skill Discovery

Covers finding skill directories under a root, matching the semantics of `convert-all-skills.sh`.

#### Feature: Recursive Skill Discovery [MVP]

* **Description**: Recursively scan a root directory for skill directories containing `SKILL.md` or `gemini-extension.json`.
* **Inputs**: `ScanRoot` (directory), `RecursiveMode` (bool).
* **Outputs**: List of `SkillDir` records: `{ Path, Name, HasSkillMd, HasGeminiManifest }`.
* **Behavior**:

  * If `RecursiveMode = true`, walk directories and collect files whose names are `SKILL.md` or `gemini-extension.json`, de-duplicating by directory like the existing `SEEN_DIRS` logic.
  * If `RecursiveMode = false`, treat `ScanRoot` as a single candidate directory and check for the same files.
  * If no skills found, surface a TUI message and allow user to change configuration or exit.
  * Emit a Bubble Tea message `SkillsDiscoveredMsg` containing the list.

#### Feature: Skill Metadata Extraction [MVP]

* **Description**: Augment `SkillDir` records with basic metadata for display.
* **Inputs**: List of `SkillDir` paths.
* **Outputs**: Enhanced `SkillDir` with derived display name (directory basename) and platform type (Claude/Gemini/Universal/Unknown) using existing Node detector semantics as reference.
* **Behavior**:

  * Infer name as directory basename.
  * Optionally (non-blocking), shell out to `skill-porter analyze` or reuse detection logic via a small utility to identify platform type and confidence.
  * Ensure failures in metadata detection don’t block listing; mark platform as Unknown on error.

### Capability: Conversion Orchestration

Covers scheduling and executing conversions using `skill-porter convert`, tracking per-skill status, and computing summaries.

#### Feature: Conversion Command Builder [MVP]

* **Description**: Build the exact `skill-porter convert` command for a given skill, target, and output base.
* **Inputs**: `SkillDir.Path`, `targetPlatform` (`gemini` or `claude`), `OutBaseDir`, optional flags (`--no-validate` etc).
* **Outputs**: Struct representing command invocation: `{ Cmd: "skill-porter", Args: ["convert", path, "--to", target, "--output", outPath, ...] }`.
* **Behavior**:

  * Compose output directory as `${OutBaseDir}/${Name}-${target}` as in the script.
  * Support optional settings (e.g. validate on/off) via config.
  * Provide a pure function usable by tests.

#### Feature: Conversion Execution Engine [MVP]

* **Description**: Execute conversions sequentially (for MVP) and update skill statuses in the model.
* **Inputs**: Command spec from Command Builder, skill identifier, target, current summary counts.
* **Outputs**:

  * `SkillConvertedMsg` on success (status: `success`, outputPath).
  * `ConversionErrorMsg` on failure (status: `failed`, error summary, exit code).
  * Updated summary counters (success/skipped/failed).
* **Behavior**:

  * Start conversions in a Go goroutine to avoid blocking the Bubble Tea event loop.
  * Capture stdout/stderr for error context (trim to reasonable length for UI).
  * Determine success from exit code (`0` → success; non-zero → failure).
  * For MVP, run conversions one at a time; later phases may add bounded concurrency.

#### Feature: Summary Aggregation & Exit Semantics [MVP]

* **Description**: Maintain counters mirroring the Bash script (success, skipped, failed, found) and map them to exit codes.
* **Inputs**: Stream of per-skill status changes.
* **Outputs**: Summary struct `{ Succeeded, Skipped, Failed, Total }`, plus final program exit code.
* **Behavior**:

  * Increment counters on state transitions to `success`, `skipped`, `failed`.
  * Render summary in footer and final “Summary” panel.
  * On exit, return non-zero exit code if `Failed > 0`; zero otherwise.
  * Support early exit with partial summary when user chooses to quit.

### Capability: TUI Presentation & Interaction

Bubble Tea model/view/update logic with Lip Gloss styling.

#### Feature: Skill List View with Statuses [MVP]

* **Description**: Main panel listing discovered skills with per-skill status.
* **Inputs**: List of `SkillDir` and each skill’s `ConversionStatus` (`pending`, `running`, `success`, `skipped`, `failed`).
* **Outputs**: Rendered Bubble Tea view (rows with name, platform, status glyph).
* **Behavior**:

  * Use Lip Gloss to style rows (borders, padding, colors per status).
  * Support scrolling for large lists; visual pointer for selected row.
  * Mark running conversions distinctly (e.g. spinner or special glyph).
  * Reflect updates in near-real-time as conversion messages arrive.

#### Feature: Skill Detail Panel [MVP]

* **Description**: Secondary panel showing details for the selected skill.
* **Inputs**: Selected `SkillDir`, latest conversion result (output path, exit code, last error line).
* **Outputs**: Detail panel view for Bubble Tea.
* **Behavior**:

  * Show path, detected platform, default target, last operation (with timestamp or simple label), last error snippet if failed, and output directory.
  * Update as new conversion completes for this skill.
  * Provide enough info for user to re-run conversion manually if needed.

#### Feature: Footer Status & Key Hints [MVP]

* **Description**: Footer strip with live progress and keybindings.
* **Inputs**: Summary counts, current operation, available actions in current state.
* **Outputs**: Single-line footer view.
* **Behavior**:

  * Display “Pending: X | Running: Y | Success: Z | Skipped: S | Failed: F”.
  * Show key hints (`↑/↓` or `j/k` to move, `c` convert, `g` gemini, `a` claude, `s` skip, `q` quit, `r` rescan).
  * Fade or disable hints when actions are not applicable (e.g. no skills loaded).

#### Feature: Keyboard Action Handling & Mode Management [MVP]

* **Description**: Map key presses to per-skill actions and app-level commands.
* **Inputs**: Key events from Bubble Tea, current selection, config, model state (pending/running/summary).
* **Outputs**: State transitions: start conversion, mark skip, set auto-convert mode, toggle recursive, quit, rescan.
* **Behavior**:

  * `c`: convert selected skill with current default target.
  * `g` / `a`: convert selected skill as gemini/claude, overriding default.
  * `s`: mark selected skill as skipped; increment skipped counter.
  * `r`: re-run discovery with same config; reset statuses and summary.
  * `q`: if any pending conversions, ask for confirmation (see gum interop); otherwise exit.
  * Implement an “auto-convert remaining” mode equivalent to the script’s “Convert ALL remaining (target=<target>)”.

### Capability: Gum Interoperability

Use gum where it still adds UX value.

#### Feature: Confirmation Prompts via Gum [Non-MVP, optional]

* **Description**: Integrate gum CLI prompts for certain disruptive confirms (quit with pending, bulk operations), while primary UX stays in Bubble Tea.
* **Inputs**: Confirmation intents (e.g. “Quit while 5 skills are still pending?”).
* **Outputs**: Boolean decision from gum, mapped back into the Bubble Tea model.
* **Behavior**:

  * Spawn `gum confirm` in a separate process when user triggers a destructive action from TUI.
  * Pause Bubble Tea interaction while gum prompt is active; resume after completion.
  * If gum is unavailable, fall back to internal TUI confirmation dialog.

### Capability: Logging & Diagnostics

#### Feature: Structured Logging [MVP]

* **Description**: Emit minimal structured logs for conversions for debugging and CI usage.
* **Inputs**: Conversion start/finish events, errors, configuration.
* **Outputs**: Lines written to stderr or dedicated log stream (e.g. JSONL or tagged text).
* **Behavior**:

  * Log at least: skill path, target, output dir, status, exit code, duration.
  * Keep logging minimal and machine-parseable; no TUI control sequences.

#### Feature: Debug Mode [Non-MVP]

* **Description**: Provide `--debug` flag to increase verbosity for troubleshooting.
* **Inputs**: CLI `--debug`, internal errors.
* **Outputs**: Additional log lines (e.g. raw `skill-porter` command, env info), optional on-screen debug panel.
* **Behavior**:

  * When `--debug` is set, include full command and working directory in logs.
  * Never leak secrets (no environment dumps containing credentials).

3. Repository Structure + Module Definitions
   ===========================================

Assumptions:

* Go module lives at the repo root or under a new `tui/` subdirectory.
* We follow “one file = one primary responsibility” and avoid cross-layer mixing.

### Repository Structure (new/changed parts only)

```text
project-root/
├── cmd/
│   └── skill-porter-tui/
│       └── main.go                    # Entry point (CLI + Bubble Tea bootstrap)
├── internal/
│   └── skillportertui/
│       ├── config/
│       │   ├── config.go              # AppConfig type, defaults, validation
│       │   └── flags.go               # CLI/env parsing → AppConfig
│       ├── domain/
│       │   ├── types.go               # SkillDir, ConversionStatus, Summary, enums
│       │   └── messages.go            # Bubble Tea message types
│       ├── discovery/
│       │   ├── discovery.go           # Recursive/single-skill scanning
│       │   └── discovery_test.go
│       ├── conversion/
│       │   ├── command_builder.go     # Build skill-porter CLI commands
│       │   ├── executor.go            # Run commands, capture output
│       │   └── conversion_test.go
│       ├── ui/
│       │   ├── model.go               # Bubble Tea model state + Update
│       │   ├── view.go                # Bubble Tea views (list, detail, footer)
│       │   ├── keymap.go              # Keybindings definitions
│       │   ├── theme.go               # Lip Gloss styles
│       │   └── ui_test.go
│       ├── guminterop/
│       │   └── confirm.go             # Optional gum-based confirms
│       └── logging/
│           └── logger.go              # Structured logging helpers
├── docs/
│   ├── skill-porter-tui.md            # Usage, flags, screenshots (new)
│   └── tasks/
│       └── todo/
│           └── 01-gum-wrapper-for-skill-porter.md  # This plan/PRD
└── scripts/
    └── convert-all-skills.sh          # Existing script, unchanged surface area
```

### Module Definitions

#### Module: `internal/skillportertui/config`

* **Maps to capability**: Configuration & Startup.
* **Responsibility**: Define configuration schema and translate flags/env into `AppConfig`.
* **File structure**:

  ```text
  config/
  ├── config.go   # AppConfig definition, defaults, validation functions
  └── flags.go    # CLI/env parsing using `flag` or similar
  ```

* **Exports**:

  * `type AppConfig` – holds `ScanRoot`, `RecursiveMode`, `DefaultTarget`, `OutBaseDir`, `AutoConvertMode`.
  * `func DefaultConfig() AppConfig` – produces defaults.
  * `func LoadFromFlagsAndEnv() (AppConfig, error)` – parse flags/env into config.
  * `func Validate(AppConfig) error` – ensures directories/targets are valid.

#### Module: `internal/skillportertui/domain`

* **Maps to capability**: Skill Discovery, Conversion Orchestration, TUI Presentation.
* **Responsibility**: Core domain types and Bubble Tea messages.
* **File structure**:

  ```text
  domain/
  ├── types.go     # SkillDir, ConversionStatus, Summary, ConversionTarget
  └── messages.go  # Bubble Tea message types and constructors
  ```

* **Exports**:

  * `type SkillDir` – `{ Path, Name, HasSkillMd, HasGeminiManifest, Platform }`.
  * `type ConversionTarget` – enum-like (`Gemini`, `Claude`).
  * `type ConversionStatus` – enum-like (`Pending`, `Running`, `Success`, `Skipped`, `Failed`).
  * `type Summary` – counters struct.
  * `type SkillsDiscoveredMsg`, `SkillConvertedMsg`, `ConversionErrorMsg`, `ConfigUpdatedMsg`, `QuitMsg`.

#### Module: `internal/skillportertui/discovery`

* **Maps to capability**: Skill Discovery.
* **Responsibility**: Scan filesystem for skills.
* **File structure**:

  ```text
  discovery/
  ├── discovery.go
  └── discovery_test.go
  ```

* **Exports**:

  * `func DiscoverSkills(root string, recursive bool) ([]domain.SkillDir, error)` – core scanning function.
  * `func EnhanceWithMetadata([]domain.SkillDir) []domain.SkillDir` – optional platform detection hook.

#### Module: `internal/skillportertui/conversion`

* **Maps to capability**: Conversion Orchestration, Logging & Diagnostics.
* **Responsibility**: Build and execute `skill-porter` conversions, emitting domain messages.
* **File structure**:

  ```text
  conversion/
  ├── command_builder.go
  ├── executor.go
  └── conversion_test.go
  ```

* **Exports**:

  * `func BuildConvertCommand(skill domain.SkillDir, target domain.ConversionTarget, outBase string) (cmd string, args []string, outDir string)` – pure builder.
  * `func ExecuteConversion(cmd string, args []string) (exitCode int, stdout, stderr string, err error)` – process execution.
  * `func StartConversionAsync(...)` – orchestration helper that wraps `ExecuteConversion` and sends Bubble Tea messages via a channel or callback.

#### Module: `internal/skillportertui/ui`

* **Maps to capability**: TUI Presentation & Interaction.
* **Responsibility**: Bubble Tea model, update, and views; Lip Gloss theming; keybindings.
* **File structure**:

  ```text
  ui/
  ├── model.go    # Bubble Tea Model, Init, Update
  ├── view.go     # View composition: header, list, detail, footer
  ├── keymap.go   # Key → action mapping
  ├── theme.go    # Lip Gloss styles
  └── ui_test.go
  ```

* **Exports**:

  * `type Model` – holds config, skills, selection index, statuses, summary.
  * `func NewModel(config AppConfig) Model` – constructor.
  * `func (m Model) Init() tea.Cmd` – discovery kick-off.
  * `func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd)` – state transitions.
  * `func (m Model) View() string` – TUI rendering.
  * `var KeyMap` – central keybinding definition.

#### Module: `internal/skillportertui/guminterop`

* **Maps to capability**: Gum Interoperability.
* **Responsibility**: Optional bridging to gum for confirmation prompts.
* **File structure**:

  ```text
  guminterop/
  └── confirm.go
  ```

* **Exports**:

  * `func ConfirmWithGum(prompt string) (bool, error)` – run `gum confirm` if available; fallback logic if not.

#### Module: `internal/skillportertui/logging`

* **Maps to capability**: Logging & Diagnostics.
* **Responsibility**: Minimal structured logging API.
* **File structure**:

  ```text
  logging/
  └── logger.go
  ```

* **Exports**:

  * `func Info(event string, fields map[string]any)` – informational logs.
  * `func Error(event string, err error, fields map[string]any)` – error logs.

#### Module: `cmd/skill-porter-tui`

* **Maps to capability**: Configuration & Startup (CLI), TUI Presentation (entry).
* **Responsibility**: Wire CLI → config → Bubble Tea program; handle process exit code.
* **File structure**:

  ```text
  cmd/skill-porter-tui/
  └── main.go
  ```

* **Exports**: None (main package).

  * `func main()` – parse flags/env, init Model, run `tea.NewProgram`, use summary/failure count to set `os.Exit` code.

4. Dependency Chain
   ===================

### Foundation Layer (Phase 0)

No dependencies.

* **Module: config**

  * Provides `AppConfig` and configuration loading/validation.
* **Module: domain**

  * Provides core types and Bubble Tea messages.
* **Module: logging**

  * Provides common logging helpers.

### Discovery & Conversion Infrastructure Layer (Phase 1)

Depends on foundation.

* **Module: discovery**

  * Depends on: `[config, domain]`.
  * Uses `AppConfig.ScanRoot` and `RecursiveMode` to produce `[]SkillDir`.
* **Module: conversion**

  * Depends on: `[domain, logging]`.
  * Uses domain types for skill/target, logging helpers for events.

### TUI Layer (Phase 2)

Depends on discovery & conversion infrastructure.

* **Module: ui**

  * Depends on: `[config, domain, discovery, conversion, logging]`.
  * Uses discovery to find skills, conversion to launch work, logging for debug, config for header and defaults.
* **Module: guminterop** (optional)

  * Depends on: `[logging]`.
  * Called from `ui` for specific prompts; has no influence on lower layers.

### Entrypoint Layer (Phase 3)

Depends on TUI and foundation.

* **Module: cmd/skill-porter-tui**

  * Depends on: `[config, ui, logging]`.
  * Orchestrates initial config load then runs Bubble Tea program; sets exit code based on model summary.

No cycles:

* Foundation has no deps.
* discovery/conversion/logging only depend on foundation.
* ui depends “downwards” only.
* cmd only depends downwards on ui and foundation.
* guminterop is optional and only called from ui, with no reverse imports.

5. Development Phases
   =====================

### Phase 0: Foundation & Configuration

**Goal**: Establish configuration schema, domain types, and logging, with a compilable stub CLI.

**Entry Criteria**: Existing repo compiles as-is; no Go TUI yet.

**Tasks**:

* [ ] Implement `config` module (AppConfig + flags/env parsing)

  * Depends on: none.
  * Acceptance criteria:

    * `go test` for `config` passes.
    * Running `skill-porter-tui --root /tmp/foo --no-recursive --target claude --out-base ./out` prints parsed config and exits without panic.
  * Test strategy: unit tests for precedence (env < flags) and validation (missing directory error).

* [ ] Implement `domain` module (core types + messages)

  * Depends on: none.
  * Acceptance criteria:

    * Types defined for `SkillDir`, `ConversionTarget`, `ConversionStatus`, `Summary`.
    * Messages defined as distinct Go types; basic compile-time test ensures they implement `tea.Msg` if Bubble Tea already included.
  * Test strategy: simple compile-time tests and basic constructor tests.

* [ ] Implement `logging` module

  * Depends on: none.
  * Acceptance criteria:

    * `Info`/`Error` available and used by a tiny stub main to log events.
    * Logs are human-readable and do not include ANSI control codes.
  * Test strategy: tests asserting output formatting patterns.

**Exit Criteria**: Running a stub `skill-porter-tui` prints parsed config and exits cleanly; core Go types and logging utilities exist and are tested.

**Delivers**: Validated configuration and domain types ready for discovery and conversion modules.

---

### Phase 1: Discovery & Conversion Infrastructure

**Goal**: Provide filesystem scanning and conversion execution primitives independent of UI.

**Entry Criteria**: Phase 0 complete.

**Tasks**:

* [ ] Implement `discovery.DiscoverSkills` and tests

  * Depends on: `[config, domain]`.
  * Acceptance criteria:

    * Given a synthetic directory tree with `SKILL.md` and `gemini-extension.json` files, discovery returns unique `SkillDir` objects in recursive mode and correctly handles non-recursive mode.
    * Empty tree → no skills, no panic, clear error or empty slice.
  * Test strategy: unit tests constructing temp directories; verify deduplication and file matching.

* [ ] Implement `conversion.BuildConvertCommand`

  * Depends on: `[domain]`.
  * Acceptance criteria:

    * For a skill named `foo-skill` with target `gemini` and out base `/tmp/out`, the output directory matches `/tmp/out/foo-skill-gemini` matching `convert-all-skills.sh` semantics.
    * Command and args arrays are stable and testable (no reliance on global state).
  * Test strategy: pure function unit tests across combinations of targets and paths.

* [ ] Implement `conversion.ExecuteConversion` with process spawning

  * Depends on: `[logging]`.
  * Acceptance criteria:

    * For a dummy `true` command, returns exitCode `0`.
    * For a dummy `false` command, returns non-zero exitCode and error.
    * Captures stdout/stderr reliably.
  * Test strategy: unit tests for exit code and output capture; skip on platforms where `true/false` not available by gating.

* [ ] Design and implement minimal Summary maintenance helper

  * Depends on: `[domain]`.
  * Acceptance criteria:

    * Given a sequence of statuses, resulting Summary matches expected counts.
  * Test strategy: unit tests for transitions.

**Exit Criteria**: Non-UI Go code supports scanning and running arbitrary commands with correct semantics and summary aggregation.

**Delivers**: CLI can be wired to run a simple non-interactive discovery + conversion run sequentially (e.g., convert first skill only) for debugging.

---

### Phase 2: Basic TUI (MVP)

**Goal**: Deliver an end-to-end, single-binary TUI that discovers skills, converts them sequentially, and shows statuses and a summary.

**Entry Criteria**: Phase 1 complete.

**Tasks**:

* [ ] Implement Bubble Tea `Model` and Update loop

  * Depends on: `[config, domain, discovery, conversion, logging]`.
  * Acceptance criteria:

    * On Init, runs discovery and populates a list of skills (or shows “no skills found”).
    * On key actions (`c` on a selected skill), triggers conversion and updates status from `pending → running → success/failed`.
  * Test strategy: unit-like tests calling `Update` with synthetic messages; ensure state transitions correct and summary matches.

* [ ] Implement `view.go` with list, detail, footer panels using Lip Gloss

  * Depends on: `[domain]`.
  * Acceptance criteria:

    * Skills render as rows with at least three columns: name, platform, status.
    * Selected row visually distinct; statuses have different styling for success/failed/skipped.
    * Footer shows summary counts and key hints.
  * Test strategy: snapshot tests of `View()` strings for small model states; manual inspection for readability.

* [ ] Wire `main.go` to run Bubble Tea program and set exit code

  * Depends on: `[config, ui, logging]`.
  * Acceptance criteria:

    * Running `skill-porter-tui` from a directory with sample skills shows TUI, allows converting at least one skill, and exits with non-zero status when any conversion fails.
  * Test strategy: manual e2e run plus basic automated test that runs program in a pseudo-TTY for a trivial case (if feasible).

**Exit Criteria**: A user can run `skill-porter-tui`, inspect discovered skills, convert them via keyboard, and see a final summary, with exit codes reflecting success/failure.

**Delivers**: MVP TUI wrapper meeting core success criteria.

---

### Phase 3: Full Action Parity with Bash Script

**Goal**: Reach behavioral parity with `convert-all-skills.sh` choices (auto-convert remaining, per-skill target overrides, skip, quit-with-summary).

**Entry Criteria**: Phase 2 complete.

**Tasks**:

* [ ] Implement per-skill target override actions (`g`, `a`)

  * Depends on: `[ui, conversion]`.
  * Acceptance criteria:

    * `g` converts selected skill as gemini even if default is claude; `a` does the reverse.
    * Summary correctly tracks results.
  * Test strategy: `Update` tests; manual run with targets.

* [ ] Implement auto-convert remaining mode

  * Depends on: `[ui, conversion]`.
  * Acceptance criteria:

    * When user chooses “convert all remaining with default target,” all pending skills are queued and processed sequentially without further confirmation (matching `AUTO_CONVERT_MODE` semantics).
    * UI clearly indicates auto mode in header or detail panel.
  * Test strategy: `Update` tests for auto mode flag; manual run with multiple skills.

* [ ] Implement rescan (`r`) behavior

  * Depends on: `[ui, discovery]`.
  * Acceptance criteria:

    * Pressing `r` resets skill list and statuses based on a fresh discovery; summary counters reset.
  * Test strategy: `Update` tests verifying old skills replaced and counters reset.

* [ ] Implement quit behavior with partial summary

  * Depends on: `[ui, logging]`.
  * Acceptance criteria:

    * `q` exits with current summary; if conversions are pending or running, a confirmation path is enforced (using TUI or guminterop once available).
  * Test strategy: `Update` tests for quit messages; manual run.

**Exit Criteria**: Feature set matches or supersedes all usage flows from `convert-all-skills.sh` (excluding its line-based gum UI).

**Delivers**: Fully capable TUI that replicates script behavior with better ergonomics.

---

### Phase 4: Gum Interop & Diagnostics

**Goal**: Integrate gum where it adds value and enrich diagnostics for troubleshooting.

**Entry Criteria**: Phase 3 complete.

**Tasks**:

* [ ] Implement `guminterop.ConfirmWithGum` and wire to quit/auto actions

  * Depends on: `[logging]`.
  * Acceptance criteria:

    * If `gum` is installed, disruptive actions (quit with pending, switch to auto-convert) use gum confirm with clear messages.
    * If gum is absent, a TUI-based confirm is used; no behavior regression.
  * Test strategy: tests that simulate `gum` presence/absence via PATH; manual run.

* [ ] Implement `--debug` flag and extended logging

  * Depends on: `[config, logging]`.
  * Acceptance criteria:

    * When `--debug` is set, logs include command args and durations; when unset, logs remain minimal.
  * Test strategy: unit tests verifying log fields for debug vs non-debug.

**Exit Criteria**: Gum used selectively where helpful; logging supports debugging issues without overwhelming standard output.

**Delivers**: Polished integration, better debuggability.

---

### Phase 5: Documentation & Hardening

**Goal**: Ship documentation and robust tests.

**Entry Criteria**: Phases 0–4 complete.

**Tasks**:

* [ ] Write `docs/skill-porter-tui.md`

  * Depends on: `[cmd/skill-porter-tui, ui]`.
  * Acceptance criteria:

    * Document flags, UI layout, keyboard shortcuts, and relationship to `convert-all-skills.sh`.
    * Include at least one screenshot of TUI.
  * Test strategy: documentation review.

* [ ] Update `docs/tasks/todo/01-gum-wrapper-for-skill-porter.md` with final status and notes

  * Depends on: the entire implementation.
  * Acceptance criteria:

    * This PRD and implementation notes are captured; task status clearly indicated.

* [ ] Expand test coverage for edge cases

  * Depends on: all modules.
  * Acceptance criteria:

    * Unit tests cover discovery, command building, state transitions, and summary logic across edge cases (no skills, mixed successes/failures, interrupted runs).
    * Target coverage thresholds met (see Test Strategy).

**Exit Criteria**: Documentation available; tests stable; CLI ready for regular use.

**Delivers**: Production-ready TUI wrapper with supporting docs and test suite.

6. User Experience
   ==================

Personas

* **Tooling engineer**: Works on multiple skills/extensions; wants a fast, inspectable batch conversion flow.
* **Skill author**: Maintains 1–3 skills; occasionally runs conversions; cares about clarity and minimal friction.

Key Flows

1. **Basic batch conversion**

   * Run `skill-porter-tui`.
   * Adjust root, recursive mode, target, output base if desired.
   * See list of discovered skills; press `c` to convert each or enable auto-convert remaining.
   * Watch statuses update; inspect any failures in the detail panel.
   * On completion, summary panel shows counts; exit.

2. **Target override for specific skills**

   * Same as above, but for a particular skill, press `g` or `a` to override target before converting.
   * Detail panel shows target used and output path.

3. **Failure inspection**

   * If a conversion fails, status shows `failed`.
   * Selecting that skill shows exit code and error snippet in detail panel.
   * User can re-run conversion (e.g. after fixing issues) by pressing `c`.

4. **Rescanning after file changes**

   * User modifies skill tree on disk.
   * Press `r` to rescan; list updates; statuses reset.

UI/UX Notes

* Use a clear header with current default target, scan mode (recursive/single), and root path.
* Skill list is the main focus; detail panel and footer are secondary but always visible.
* Lip Gloss styling:

  * Consistent theme with distinct colors for success (green-ish), failed (red-ish), pending (dim).
  * Bordered panels (header, list, detail) with padding; avoid clutter.
* Avoid nested modal flows; keep interactions single-key and reversible.
* Ensure keyboard navigation and reading experience work well on both small and large terminals.

7. Technical Architecture
   =========================

System Components

* **skill-porter-tui binary (Go)**

  * Main entry point; CLI + Bubble Tea program.
  * Uses `os/exec` to call `skill-porter` CLI for conversions.

* **Discovery engine (Go)**

  * Walks filesystem based on config; identifies skills by presence of `SKILL.md` or `gemini-extension.json`.

* **Conversion engine (Go)**

  * Builds and runs `skill-porter convert <dir> --to <target> --output <path>`.
  * Interprets exit codes and output.

* **TUI engine (Go + Bubble Tea + Lip Gloss)**

  * Owns application state, key handling, and views.
  * Communicates with discovery/conversion via Bubble Tea messages and commands.

* **Optional gum interop (Go + gum CLI)**

  * Used for confirm prompts for destructive/bulk actions if available.

Data Models

* `AppConfig`: root string, recursive bool, default target enum, output base string, debug flag, auto-convert flag.
* `SkillDir`: path string, name string, bools for `HasSkillMd`/`HasGeminiManifest`, optional platform string.
* `ConversionStatus`: `pending | running | success | skipped | failed`.
* `Summary`: integer counters.
* Bubble Tea messages: strongly typed structs carrying IDs and payloads.

Technology Stack

* Go (version as per repo standard).
* `github.com/charmbracelet/bubbletea` for TUI event loop.
* `github.com/charmbracelet/lipgloss` for styling.
* `os/exec` for `skill-porter` and optional `gum`.
* Existing Node-based `skill-porter` CLI as the conversion engine; no changes required to its internal JS modules.

Decision: call `skill-porter` CLI directly instead of wrapping `convert-all-skills.sh`

* **Rationale**:

  * Avoid double-interactive behavior (gum prompts inside a TUI).
  * Prevent long-term divergence between TUI and script by centralizing semantics around the CLI contract.
  * Keeps Bash script as an independent, still-supported entrypoint.
* **Trade-offs**:

  * Some logic must be re-implemented in Go (e.g., auto-convert mode, summaries).
  * Two orchestrators (Bash and TUI) to maintain conceptually.
* **Alternatives considered**:

  * Adding `--non-interactive`/`--json` to `convert-all-skills.sh` and wrapping it from Go (rejected due to layering and coupling).
  * Embedding Node’s `SkillPorter` directly via a cgo/Node bridge (too complex and brittle).

Decision: modular Go layout under `internal/skillportertui`

* **Rationale**: Aligns with enforced module design rules; avoids god modules and “utils”.
* **Trade-offs**: More files, but clearer maintenance.
* **Alternatives**: Single `tui.go` or `main.go` with everything (explicitly rejected).

8. Test Strategy
   ================

## Test Pyramid

```text
        /\
       /E2E\        ← ~10%: end-to-end runs of CLI in controlled environments
      /------\
     /Integration\  ← ~30%: Bubble Tea model transitions + process exec with fake commands
    /------------\
   /  Unit Tests  \← ~60%: pure discovery, config, command building, summary logic
  /----------------\
```

## Coverage Requirements

* Line coverage: ≥ 80% on `internal/skillportertui` packages.
* Branch coverage: ≥ 70% on discovery, config, and conversion modules.
* Function coverage: ≥ 90% of exported functions.
* Statement coverage: ≥ 80% overall.

## Critical Test Scenarios

### Discovery

**Happy path**

* Scenario: Directory tree with multiple skills and nested directories.
* Expected: All unique skill directories detected; counts match; no duplicates.

**Edge cases**

* Scenario: Empty root, root not existing, only `SKILL.md` files, only `gemini-extension.json`.
* Expected: Informative error or “no skills found” message; no panics.

**Error cases**

* Scenario: Permission-denied directories.
* Expected: Discovery continues where possible; logs a warning; surfaces partial results gracefully.

**Integration points**

* Discovery output consumed by UI; list view correctly reflects number of skills and selection.

### Conversion Command Builder & Executor

**Happy path**

* Scenario: Valid skill dir, target `gemini`, out base set.
* Expected: Command matches `skill-porter convert <dir> --to gemini --output <_expected>`; exitCode=0 when stubbed.

**Edge cases**

* Scenario: Paths with spaces, nested directories.
* Expected: Proper quoting/arg building; no shell interpolation issues.

**Error cases**

* Scenario: `skill-porter` not on PATH.
* Expected: Clear error surfaced in TUI detail panel and logs; global error message instructing to install `skill-porter`.

**Integration points**

* Conversion messages updating Bubble Tea model statuses and Summary.

### TUI Model & Views

**Happy path**

* Scenario: User uses only `c` to convert all skills sequentially.
* Expected: All statuses reach `success`; summary matches; exit code 0.

**Edge cases**

* Scenario: Narrow terminal width; multiple pages of skills.
* Expected: List scrolls correctly; footer remains readable.

**Error cases**

* Scenario: Several conversions fail; user quits early.
* Expected: Failed statuses and summary counts reflect actual results; exit code non-zero.

**Integration points**

* Interaction between key handling and conversion engine; ensuring no state corruption when multiple conversions triggered quickly.

## Test Generation Guidelines

* Prefer table-driven tests for config parsing, discovery, and command building.
* For Bubble Tea model tests, simulate sequences of messages and assert final model state (pure Update tests).
* Keep any e2e tests deterministic by using small synthetic trees and stubbed `skill-porter` commands where possible.
* Avoid fragile TUI snapshot tests that depend on terminal width; instead test structure (presence of key labels, counts, and status text).

9. Risks and Mitigations
   ========================

## Technical Risks

**Risk**: Dual semantics with Bash script diverge over time

* **Impact**: Medium – confusing differences between TUI and script behavior.
* **Likelihood**: Medium.
* **Mitigation**:

  * Treat `convert-all-skills.sh` as a behavioral reference; update TUI when script semantics change and vice versa.
  * Keep core defaults identical (root, target, output base).
* **Fallback**: Document any intentional differences in `docs/skill-porter-tui.md`.

**Risk**: TUI rendering performance on large trees

* **Impact**: Medium – sluggish UX.
* **Likelihood**: Medium on large repos.
* **Mitigation**:

  * Use efficient data structures and avoid excessive re-renders; only re-render on relevant messages.
  * Consider throttling visual updates for very large lists if needed.
* **Fallback**: Recommend script-based workflow for extremely large trees.

**Risk**: Cross-platform process execution differences

* **Impact**: Medium – inconsistent behavior on Windows/WSL vs Unix.
* **Likelihood**: Medium.
* **Mitigation**:

  * Use `exec.Command` without shell where possible; keep commands simple.
  * Keep path handling platform-neutral.
* **Fallback**: Document OS limitations; add integration tests on multiple platforms as the project matures.

## Dependency Risks

**Risk**: Bubble Tea / Lip Gloss API changes

* **Impact**: Low–Medium.
* **Likelihood**: Low.
* **Mitigation**:

  * Pin versions in Go module; track release notes.
* **Fallback**: If APIs shift, adjust TUI implementation; core discovery/conversion logic remains unaffected.

**Risk**: `skill-porter` CLI changes

* **Impact**: High – command options or behavior may change.
* **Likelihood**: Medium.
* **Mitigation**:

  * Keep TUI command construction minimal and aligned with documented CLI interface (`convert <source> --to <platform> --output <path>`).
* **Fallback**: Detect CLI version mismatches and warn users.

## Scope Risks

**Risk**: Over-extending features (config persistence, concurrency, advanced filters)

* **Impact**: Medium – delays MVP ship.
* **Likelihood**: High without discipline.
* **Mitigation**:

  * Strictly treat Phase 2 as MVP; non-MVP features gated to later phases.
* **Fallback**: Ship MVP with documented limitations; postpone nonessential features.

10. Appendix
    ============

## References

* `convert-all-skills.sh`: current recursive scanning + gum script, including interactive prompts, auto-convert mode, and summary counters.
* `skill-porter` CLI and core modules: conversion, detection, validation, universalization logic.
* Module design rules: avoid god modules, single-responsibility files, feature-oriented slices.
* RPG PRD template and method: structure for capability/structural/dependency/phase design used here.

## Glossary

* **Skill**: Claude Code skill or Gemini CLI extension directory containing `SKILL.md` and/or `gemini-extension.json`.
* **TUI**: Text-based User Interface using Bubble Tea and Lip Gloss.
* **MVP**: Minimum viable product; smallest end-to-end implementation that delivers core conversion flows.

## Open Questions

* Whether to add a non-interactive/JSON-output mode to `convert-all-skills.sh` for non-TUI tooling, or keep all new orchestration in Go only.
* Whether future versions should support parallel conversions with bounded concurrency and how to present that safely in the TUI.
