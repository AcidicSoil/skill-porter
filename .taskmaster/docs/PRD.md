# PRD

## 1. Overview

---

### Problem Statement

The current `convert-all-skills.sh` script discovers skill directories (containing `SKILL.md` or `gemini-extension.json`), prompts the user with gum, and invokes `skill-porter convert` one skill at a time. It is functional but linear and text-heavy: there is no structured overview of all discovered skills, no at-a-glance status dashboard, and limited ability to manage per-skill actions once a run starts.

This makes multi-skill/multi-target conversion runs harder to reason about, slower to operate, and more error-prone, especially when dealing with dozens or hundreds of skills.

### Target Users

* **Skill authors / maintainers**

  * Maintain collections of skills organized in directory trees.
  * Need to convert or re-convert entire trees after changes in targets or schemas.
  * Want fast, visible feedback when something fails and the ability to selectively retry.

* **Toolchain maintainers / infra engineers**

  * Need a more maintainable, testable conversion orchestration layer than a single interactive bash script.
  * Need a clear separation of concerns (discovery, orchestration, UI) and testable modules.

### Why existing solutions fail

* `convert-all-skills.sh` mixes:

  * User interaction (gum prompts/spinners),
  * Discovery (`find` + de-dup),
  * Orchestration (per-skill control flow, AUTO_CONVERT_MODE),
  * Reporting (summary).
* The flow is strictly linear and log-oriented; you cannot:

  * See all discovered skills and their status at once.
  * Navigate between skills to inspect details or errors.
  * Change strategy mid-run with clear visual feedback.
* The script is not easily consumed by other tools: no JSON/structured status API; behavior is encoded in shell control flow.

### Success Metrics

* 100% feature parity with existing bash workflow for:

  * Root selection, recursive vs single-skill mode, target selection, per-skill decision, summary counters (success/skipped/failed).
* User can complete an end-to-end multi-skill conversion from a single `skill-porter-tui` binary without needing the script, while `convert-all-skills.sh` remains fully usable standalone (no regressions).
* Users can:

  * See a list of all discovered skills with per-skill status.
  * Convert/skip individual skills and see live updates.
  * Identify failed skills in one glance and inspect error messages.
* Codebase:

  * No “god modules”; each module has a single responsibility and is testable in isolation.
  * Unit tests cover ≥80% of non-UI domain logic (discovery, command building, state transitions).

---

2. Capability Tree (Functional Decomposition)

---

### Capability: Configuration & Session Setup

High-level: gather and manage configuration for a conversion session (scan root, recursive mode, default target, output base directory, and CLI overrides).

#### Feature: Configuration model and defaults (MVP)

* **Description**: Represent and store all configuration needed to run a conversion session with sensible defaults.
* **Inputs**:

  * CLI flags/env vars (`--root`, `--recursive/--no-recursive`, `--target`, `--out-base`).
  * Implicit defaults (e.g., `./skills`, `./converted-<target>-skills`).
* **Outputs**:

  * Immutable `Config` struct used by discovery and conversion modules.
* **Behavior**:

  * On startup, read CLI flags/env vars.
  * Apply defaults when values are not supplied.
  * Validate paths/values; on invalid, return errors with clear messages.
  * Provide a snapshot used to seed the TUI model.

#### Feature: In-TUI configuration editing (post-MVP)

* **Description**: Allow users to change configuration (root, recursive mode, default target, output base) from within the TUI before discovery runs.
* **Inputs**:

  * Current `Config`.
  * User keyboard input in TUI.
* **Outputs**:

  * Updated `Config` instance and corresponding `ConfigUpdatedMsg`.
* **Behavior**:

  * Provide a focused “configuration view” or inline controls.
  * On change, validate and update model; invalid values show inline errors.
  * Optionally trigger re-discovery if config changes after skills have been loaded.

---

### Capability: Skill Discovery & Modeling

High-level: find skill directories and represent them as domain entities.

#### Feature: Recursive skill discovery (MVP)

* **Description**: Discover skill directories under a root, matching the bash script semantics.
* **Inputs**:

  * `Config.ScanRoot`.
  * `Config.RecursiveMode` (bool).
  * File system (presence of `SKILL.md` or `gemini-extension.json` in directories).
* **Outputs**:

  * `[]SkillDir` list where each `SkillDir` has:

    * `Path` (absolute or normalized),
    * `Name` (basename),
    * `HasSKILL`, `HasGeminiExtension`.
  * `SkillsDiscoveredMsg` to Bubble Tea.
* **Behavior**:

  * If recursive: walk directories under root, find files matching `SKILL.md` or `gemini-extension.json`, then de-duplicate by directory, matching `SEEN_DIRS` semantics.
  * If non-recursive: treat the root as a single candidate and check for skill files.
  * Sort results by path or name for stable display.
  * If none found, emit an empty message and a model state that reflects “no skills found”.

#### Feature: Skill domain model and status (MVP)

* **Description**: Represent each discovered skill, its selected target, status, and error state.
* **Inputs**:

  * `[]SkillDir` from discovery.
  * Conversion actions invoked by user.
  * Conversion results (exit code, stderr/stdout).
* **Outputs**:

  * `SkillState` per skill:

    * `Status` (pending, running, success, skipped, failed).
    * `SelectedTarget`.
    * `OutputPath`.
    * `LastErrorMessage` (optional).
  * Aggregate `Summary` (success, skipped, failed, total).
* **Behavior**:

  * Initialize statuses to `pending`.
  * Update statuses and counters as conversions complete.
  * Provide derived summary at all times.

#### Feature: Rescan / refresh (post-MVP)

* **Description**: Allow user to rescan the tree without restarting the binary.
* **Inputs**:

  * Current `Config`.
  * User “rescan” action.
* **Outputs**:

  * Recomputed `[]SkillDir` and corresponding `SkillsDiscoveredMsg`.
* **Behavior**:

  * Re-run discovery and replace the model’s skills list and summary.
  * Optionally preserve error history in a separate log.

---

### Capability: Conversion Orchestration

High-level: orchestrate calls to `skill-porter convert` for selected skills, track progress, and handle errors, using the bash script as behavioral reference.

#### Feature: Command building and process execution (MVP)

* **Description**: Build and run `skill-porter convert` commands for a given skill, target, and output base.
* **Inputs**:

  * `SkillDir.Path`, `SkillDir.Name`.
  * Target (`gemini`, `claude`, future).
  * `Config.OutBaseDir`.
* **Outputs**:

  * OS commands (`exec.Cmd` or equivalent).
  * `SkillConvertedMsg` with success/failure and error details.
* **Behavior**:

  * Build output directory as `${OutBaseDir}/${name}-${target}` consistent with script.
  * Ensure output directory exists before invoking.
  * Execute `skill-porter convert "<path>" --to "<target>" --output "<out>"`.
  * Capture exit code and stderr.
  * Map exit code 0 → success; non-zero → failed with message.
  * Surface missing `skill-porter` as a pre-check error, mirroring script behavior.

#### Feature: Per-skill action selection (MVP)

* **Description**: Choose, per skill, what action to take, equivalent to bash/gum choices.
* **Inputs**:

  * User keypress (`c`, `g`, `a`, `s`, etc.).
  * Current `SkillState`.
  * `Config.DefaultTarget`.
* **Outputs**:

  * `StartConversionMsg` (with skill id + chosen target) or `SkipSkill` side-effect.
* **Behavior**:

  * Map keys:

    * `c`: convert with default target,
    * `g`: convert as gemini,
    * `a`: convert as claude,
    * `s`: skip,
    * `q`: quit,
    * optional: `A`: auto-convert remaining with current target (AUTO_CONVERT_MODE analog).
  * Prevent starting conversion if skill is already `running` or `success`.
  * If skip: set status `skipped`, update summary.

#### Feature: AUTO_CONVERT_MODE equivalent (post-MVP)

* **Description**: Provide a mode to auto-convert remaining skills with a selected target, matching script’s “Convert ALL remaining” behavior.
* **Inputs**:

  * User action enabling auto mode.
  * List of skills with status `pending`.
* **Outputs**:

  * Batched `StartConversionMsg` events for remaining skills (likely sequentially).
* **Behavior**:

  * Set a session-level `AutoConvertEnabled` + `AutoTarget`.
  * As the user advances selection or triggers “convert next pending”, automatically run conversions without separate confirmation.

---

### Capability: TUI Presentation & Interaction

High-level: provide a Bubble Tea-based TUI with Lip Gloss styling, representing state and managing interactions.

#### Feature: Main list + detail layout (MVP)

* **Description**: Display discovered skills in a scrollable list with a detail panel and header/footer.
* **Inputs**:

  * Current list of `SkillState`.
  * Current selection index.
  * `Summary` data.
  * `Config` snapshot.
* **Outputs**:

  * Bubble Tea `View` rendering:

    * Header (title + config snapshot),
    * Main panel (list of skills with status),
    * Detail panel (selected skill info),
    * Footer (summary and key hints).
* **Behavior**:

  * Render per-skill rows with:

    * Name,
    * Status icon (pending/running/success/skipped/failed),
    * Selected target.
  * Detail panel shows path, output directory, last error snippet.
  * Footer shows `Succeeded / Skipped / Failed / Total`.

#### Feature: Keyboard navigation and commands (MVP)

* **Description**: Provide intuitive keyboard controls for navigating skills and triggering actions.
* **Inputs**:

  * Keyboard events (↑/↓, `j/k`, `c/g/a/s/q/r`).
  * Current selection and state.
* **Outputs**:

  * Bubble Tea messages updating selection or triggering domain actions (`StartConversionMsg`, `SkipSkill`, `RescanRequestedMsg`).
* **Behavior**:

  * Up/down/j/k move selection within bounds.
  * Conversion keys operate on selected skill.
  * `q` triggers quit flow (may confirm if there are pending skills).
  * `r` rescans using current `Config`.

#### Feature: Theming and styling with Lip Gloss (MVP)

* **Description**: Apply consistent color, padding, borders, and typography-like spacing.
* **Inputs**:

  * Theme configuration (constants for colors/styles).
  * Current UI state (e.g., status for color choice).
* **Outputs**:

  * Styled header, list rows, selection highlight, error messages.
* **Behavior**:

  * Use Lip Gloss to define:

    * Header style (bold, inverted or accent color).
    * Panel borders and margins.
    * Row styles for different statuses (success/failed/skipped/pending).
  * Ensure layout degrades reasonably on narrow terminals.

#### Feature: Summary / completion view (MVP)

* **Description**: Provide a clear summary after all conversions are done or user quits early.
* **Inputs**:

  * `Summary`.
  * List of failed skills (names + errors).
* **Outputs**:

  * End-of-run summary view with counts and a short list of failed skills.
  * Final exit code (0 if no failures, non-zero otherwise).
* **Behavior**:

  * When all skills are non-pending or user chooses to end: show focused summary.
  * Exit code policy: non-zero if `failed > 0`, mirroring script semantics.

---

### Capability: Gum Interoperability (Optional)

#### Feature: Gum-driven confirmations (post-MVP)

* **Description**: For specific one-off confirmations, optionally shell out to gum instead of building full custom TUI prompts.
* **Inputs**:

  * Requests from app (e.g., “confirm quit with N pending skills”).
* **Outputs**:

  * Boolean decisions.
* **Behavior**:

  * Execute `gum confirm` with appropriate title and parse exit code.
  * Only used where it simplifies implementation and does not conflict with Bubble Tea event loop.

---

### Capability: Error Handling, Logging, and Diagnostics

#### Feature: Error mapping and surfacing (MVP)

* **Description**: Map process-level and domain-level errors into user-visible messages.
* **Inputs**:

  * Process exit codes and stderr from `skill-porter`.
  * Discovery errors (e.g., unreadable directories).
* **Outputs**:

  * `ConversionErrorMsg` / diagnostics stored in `SkillState.LastErrorMessage`.
  * Log lines to stderr when appropriate.
* **Behavior**:

  * For unexpected failures (missing `skill-porter`, permission issues), show a global error banner and abort if necessary (matching script for missing `skill-porter` and `gum`).
  * For per-skill failures, record error snippets and highlight failed rows.

---

### Capability: Documentation & Developer Support

#### Feature: User documentation (MVP)

* **Description**: Document usage, flags, and interaction model.
* **Inputs**:

  * Final behavior of TUI.
* **Outputs**:

  * `docs/skill-porter-tui.md`.
  * README snippet.
  * Task plan doc `docs/tasks/todo/01-gum-wrapper-for-skill-porter.md`.
* **Behavior**:

  * Describe installation, flags, UI, and relation to `convert-all-skills.sh`.
  * Include examples for common workflows.

---

3. Repository Structure + Module Definitions (Structural Decomposition)

---

Assumptions:

* Go-based TUI with Bubble Tea + Lip Gloss.
* Feature-oriented, no god modules; each file has one primary responsibility.

### Proposed Repository Structure (new code)

```text
project-root/
  cmd/
    skill-porter-tui/
      main.go

  internal/
    skillporter/
      config/
        config.go
        flags.go
      domain/
        types.go
        status.go
        summary.go
      discovery/
        discovery.go
      convert/
        command_builder.go
        runner.go
      tui/
        model/
          model.go
          messages.go
          update.go
        view/
          layout.go
          list_view.go
          detail_view.go
          summary_view.go
        theme/
          theme.go
          status_styles.go
        keys/
          keys.go
      errors/
        errors.go

  docs/
    skill-porter-tui.md
    tasks/
      todo/
        01-gum-wrapper-for-skill-porter.md

  scripts/
    convert-all-skills.sh  # existing, unchanged :contentReference[oaicite:22]{index=22}

  internal_tests/
    skillporter/
      discovery_test.go
      convert_command_builder_test.go
      model_update_test.go
      config_flags_test.go
```

### Module Definitions

#### Module: `cmd/skill-porter-tui`

* **Maps to capability**: Configuration & Session Setup; TUI bootstrapping.
* **Responsibility**: CLI entry point; parse flags, construct `Config`, instantiate and run TUI program.
* **File structure**:

  * `main.go`
* **Exports**:

  * Binary `skill-porter-tui` (no exported Go symbols).

#### Module: `internal/skillporter/config`

* **Maps to capability**: Configuration & Session Setup.
* **Responsibility**: Parse command-line flags/env; create and validate `Config`.
* **Files**:

  * `config.go`: defines `Config` struct, validation logic, default computation.
  * `flags.go`: wires Go flag parsing into `Config`.
* **Exports**:

  * `type Config struct { ... }`
  * `func NewConfigFromFlags() (Config, error)`
  * `func WithDefaults(Config) Config`

#### Module: `internal/skillporter/domain`

* **Maps to capability**: Skill Discovery & Modeling.
* **Responsibility**: Define domain types and status enums.
* **Files**:

  * `types.go`: `SkillDir`, `ConversionTarget`, identifiers.
  * `status.go`: `ConversionStatus` enum and helpers.
  * `summary.go`: `Summary` struct and aggregation helpers.
* **Exports**:

  * `type SkillDir struct { Path, Name string; HasSKILL, HasGeminiExtension bool }`
  * `type ConversionTarget string`
  * `type ConversionStatus string`
  * `type Summary struct { Success, Skipped, Failed, Total int }`
  * Helper functions like `func NewSummaryFromStates([]SkillState) Summary`.

#### Module: `internal/skillporter/discovery`

* **Maps to capability**: Skill Discovery & Modeling.
* **Responsibility**: Implement filesystem discovery consistent with `convert-all-skills.sh`.
* **Files**:

  * `discovery.go`
* **Exports**:

  * `func DiscoverSkills(cfg Config) ([]SkillDir, error)`
* **Behavior notes**:

  * Handles recursive vs single mode and directory de-duplication.

#### Module: `internal/skillporter/convert`

* **Maps to capability**: Conversion Orchestration.
* **Responsibility**: Build and run `skill-porter convert` commands.
* **Files**:

  * `command_builder.go`: pure functions for constructing command + args + output path.
  * `runner.go`: executes commands and returns structured result.
* **Exports**:

  * `type ConversionRequest struct { Skill SkillDir; Target ConversionTarget; OutBase string }`
  * `type ConversionResult struct { Skill SkillDir; Target ConversionTarget; OutputPath string; Err error }`
  * `func BuildCommand(req ConversionRequest) (name string, args []string, outDir string, err error)`
  * `func RunConversion(ctx context.Context, req ConversionRequest) ConversionResult`

#### Module: `internal/skillporter/tui/model`

* **Maps to capability**: TUI Presentation & Interaction.
* **Responsibility**: Bubble Tea model/state and update loop.
* **Files**:

  * `model.go`: defines `Model` struct (skills slice, config, selection, summary).
  * `messages.go`: Go types for Bubble Tea messages (`ConfigUpdatedMsg`, `SkillsDiscoveredMsg`, `StartConversionMsg`, `SkillConvertedMsg`, `ConversionErrorMsg`, `SummaryUpdatedMsg`, `QuitMsg`).
  * `update.go`: `Update` function implementing Bubble Tea update pattern.
* **Exports**:

  * `type Model struct { ... }`
  * `func NewModel(cfg Config) Model`
  * Message types.
  * `func (m Model) Init() tea.Cmd`
  * `func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd)`

#### Module: `internal/skillporter/tui/view`

* **Maps to capability**: TUI Presentation & Interaction.
* **Responsibility**: Render layout, list, detail, and summary views as strings.
* **Files**:

  * `layout.go`: orchestrates header/main/detail/footer composition.
  * `list_view.go`: renders skill list.
  * `detail_view.go`: renders selected skill detail.
  * `summary_view.go`: renders completion summary.
* **Exports**:

  * `func View(m Model) string`
  * Internal helpers for tests (e.g., `renderList`, `renderSummary`).

#### Module: `internal/skillporter/tui/theme`

* **Maps to capability**: TUI Presentation & Interaction (Theming).
* **Responsibility**: Define Lip Gloss styles and status-based styling.
* **Files**:

  * `theme.go`: global theme (colors, borders, padding).
  * `status_styles.go`: style mapping for statuses.
* **Exports**:

  * `type Theme struct { ... }`
  * `func DefaultTheme() Theme`
  * `func (t Theme) StatusStyle(status ConversionStatus) lipgloss.Style`

#### Module: `internal/skillporter/tui/keys`

* **Maps to capability**: TUI Presentation & Interaction (Keybindings).
* **Responsibility**: Centralize key bindings.
* **Files**:

  * `keys.go`
* **Exports**:

  * `type KeyMap struct { Up, Down, ConvertDefault, ConvertGemini, ConvertClaude, Skip, Quit, Rescan key.Binding }`
  * `func DefaultKeyMap() KeyMap`

#### Module: `internal/skillporter/errors`

* **Maps to capability**: Error Handling, Logging, Diagnostics.
* **Responsibility**: Define error types and wrapping helpers for consistent messaging.
* **Files**:

  * `errors.go`
* **Exports**:

  * Error constructors (e.g., `ErrSkillPorterMissing`, `ErrNoSkillsFound`).

---

4. Dependency Chain

---

### Foundation Layer (Phase 0)

No dependencies.

* **config**

  * Provides `Config` struct and flag parsing.
* **domain**

  * Provides core domain types (`SkillDir`, `ConversionTarget`, `ConversionStatus`, `Summary`).
* **errors**

  * Provides structured error types.

### Filesystem & Process Layer (Phase 1)

* **discovery**

  * Depends on: `config`, `domain`, `errors`.
  * Provides `DiscoverSkills`.
* **convert**

  * Depends on: `domain`, `config`, `errors`.
  * Provides command builder + process runner.

### TUI Core Layer (Phase 2)

* **tui/model**

  * Depends on: `config`, `domain`, `convert`, `discovery`, `errors`.
  * Provides Bubble Tea model, messages, update logic.
* **tui/keys**

  * Depends on: none (constants only) or optionally on Bubble Tea key types.
* **tui/theme**

  * Depends on: `domain` (status enums).

### TUI View Layer (Phase 3)

* **tui/view**

  * Depends on: `tui/model`, `tui/theme`, `tui/keys`, `domain`.
  * Provides `View()` production function.

### CLI Entry Layer (Phase 4)

* **cmd/skill-porter-tui**

  * Depends on: `config`, `tui/model`, `tui/view`, Bubble Tea runtime.
  * Wires everything into a runnable binary.

No cycles: each layer depends only on equal or lower layers; there are no mutual dependencies.

---

5. Development Phases

---

### Phase 0: Domain & Configuration Foundation

**Goal**: Establish configuration, domain types, and error primitives.

**Entry Criteria**: Existing repo; no dependencies within this feature set.

**Tasks**:

* [ ] Implement `Config` and flag parsing (depends on: none)

  * Acceptance criteria:

    * `NewConfigFromFlags()` correctly parses provided flags and applies defaults.
    * Invalid roots or targets produce descriptive errors.
  * Test strategy:

    * Unit tests in `config_flags_test.go` cover:

      * Default configuration with no flags.
      * Each flag individually and combinations.
      * Error cases for invalid paths/targets.

* [ ] Implement domain types and statuses (depends on: none)

  * Acceptance criteria:

    * `SkillDir`, `ConversionTarget`, `ConversionStatus`, and `Summary` defined.
    * Helper functions produce correct summary counts based on sample states.
  * Test strategy:

    * Unit tests verifying:

      * Summary aggregation logic.
      * Correct mapping from status lists to counts.

* [ ] Implement error types (depends on: none)

  * Acceptance criteria:

    * Distinct error types for missing `skill-porter`, missing skills, etc.
    * Errors implement `error` and are distinguishable via `errors.Is`.
  * Test strategy:

    * Unit tests verifying type identity and wrapping behavior.

**Exit Criteria**:

* Config, domain, and error modules compile and are covered by unit tests.
* Other modules can import `Config`, domain types, and errors without changes.

**Delivers**:

* Stable foundation for discovery, conversion, and TUI layers.

---

### Phase 1: Discovery & Conversion Engine

**Goal**: Implement skill discovery and conversion execution without TUI.

**Entry Criteria**: Phase 0 complete.

**Tasks**:

* [ ] Implement `DiscoverSkills` (depends on: config, domain, errors)

  * Acceptance criteria:

    * Given synthetic directory trees, discovery returns expected `SkillDir` list:

      * Recursive and non-recursive modes.
      * De-duplication semantics match bash script.
      * Correct handling of `SKILL.md` and `gemini-extension.json`.
    * Returns `ErrNoSkillsFound` when no skills exist.
  * Test strategy:

    * Unit tests with temporary directories and files.
    * Cases: zero, one, many skills; overlapping trees.

* [ ] Implement conversion command builder (depends on: config, domain)

  * Acceptance criteria:

    * For given request, `BuildCommand` returns expected binary name, args, and output directory.
    * Output directory mirrors `${OUT_BASE}/${name}-${target}` pattern from script.
  * Test strategy:

    * Pure unit tests on `BuildCommand`.
    * Validate path joining and target handling.

* [ ] Implement conversion runner (depends on: config, domain, errors)

  * Acceptance criteria:

    * On a fake or stubbed `skill-porter`, runner correctly interprets exit codes and stderr.
    * Missing `skill-porter` produces `ErrSkillPorterMissing`.
  * Test strategy:

    * Use an injected command-runner interface to simulate success and failure.
    * Tests for success, non-zero exit, binary missing.

**Exit Criteria**:

* Discovery and conversion modules are test-covered and usable from a simple Go main (for manual sanity checks).
* No UI dependencies in these modules.

**Delivers**:

* Programmatic, testable engine for skill discovery and conversion.

---

### Phase 2: Core TUI Model & Event Loop

**Goal**: Implement Bubble Tea model, messages, and wiring to discovery/conversion engine.

**Entry Criteria**: Phases 0–1 complete.

**Tasks**:

* [ ] Define Bubble Tea messages and model struct (depends on: domain, config)

  * Acceptance criteria:

    * Message types exist for configuration updates, discovery complete, start conversion, conversion finished, errors, summary updates, quit.
    * Model holds skills list, config, selection index, summary, and any global errors.
  * Test strategy:

    * Unit tests verifying model initialization and message type usage.

* [ ] Implement `Init` and discovery kick-off (depends on: discovery)

  * Acceptance criteria:

    * When model starts, it triggers `DiscoverSkills` via a Tea command using current `Config`.
    * On completion, `SkillsDiscoveredMsg` populates skills and sets summary accordingly.
  * Test strategy:

    * Use Bubble Tea testing patterns to simulate `Init` and message dispatch; assert model state updates.

* [ ] Implement update logic for actions and conversion orchestration (depends on: convert)

  * Acceptance criteria:

    * Selection changes via navigation messages.
    * Conversion actions produce `StartConversionMsg` and conversion commands.
    * Conversion completion updates statuses and summary correctly.
  * Test strategy:

    * State-machine-style unit tests that feed sequences of messages and assert transitions.

**Exit Criteria**:

* A minimal TUI (even with a placeholder view) can:

  * Discover skills on startup.
  * Trigger a conversion for a selected skill.
  * Reflect updated status and summary.

**Delivers**:

* Working TUI logic with placeholder view; ready for visual polishing.

---

### Phase 3: TUI View, Theming, and UX

**Goal**: Implement full Bubble Tea view with Lip Gloss styling and keybindings.

**Entry Criteria**: Phases 0–2 complete.

**Tasks**:

* [ ] Implement key map and navigation (depends on: model)

  * Acceptance criteria:

    * Key mappings defined in `KeyMap`.
    * Model update correctly interprets navigation and action keys.
  * Test strategy:

    * Unit tests verifying mapping and update responses for key events.

* [ ] Implement theme and status styles (depends on: domain)

  * Acceptance criteria:

    * A coherent theme object exists with header, list, detail, and footer styles.
    * Status-specific styles distinguish pending/running/success/skipped/failed.
  * Test strategy:

    * Snapshot tests of style strings where appropriate.
    * Manual inspection for readability in common terminals.

* [ ] Implement list/detail/summary views (depends on: model, theme, keys)

  * Acceptance criteria:

    * Skills rendered as scrollable list; selected row visibly highlighted.
    * Detail panel shows path, target, output path, and error snippet.
    * Summary footer/summary view shows counts and statuses.
  * Test strategy:

    * Golden/snapshot tests for representative model states.
    * Manual verification in real terminal.

**Exit Criteria**:

* `skill-porter-tui` provides a visually distinct, usable TUI that meets MVP criteria:

  * End-to-end conversions from TUI.
  * Clear status and summary.

**Delivers**:

* User-facing TUI with structured layout and theming.

---

### Phase 4: Enhancements & Gum Interoperability

**Goal**: Add non-essential but valuable enhancements.

**Entry Criteria**: Phases 0–3 complete.

**Tasks**:

* [ ] Implement AUTO_CONVERT_MODE equivalent (depends on: model, convert)

  * Acceptance criteria:

    * A mode where user can auto-convert remaining pending skills with a chosen target.
    * Behavior mirrors `Convert ALL remaining` semantics conceptually.
  * Test strategy:

    * Unit tests on model transitions and summary updates in auto mode.

* [ ] Implement gum-based optional confirmations (depends on: model, external gum binary)

  * Acceptance criteria:

    * For actions like quitting with pending skills, gum confirm can be used when available.
    * TUI gracefully falls back to in-TUI confirmation if gum is missing.
  * Test strategy:

    * Tests using injected command runner to simulate gum presence/absence.

* [ ] Documentation completion (depends on: all previous)

  * Acceptance criteria:

    * `docs/skill-porter-tui.md` and task doc updated with final behavior.
    * README snippet present.
  * Test strategy:

    * Manual review.

**Exit Criteria**:

* TUI matches or exceeds bash script ergonomics.
* Docs describe all major behaviors.

**Delivers**:

* Polished, production-appropriate tool.

---

6. User Experience

---

### Personas

* **Skill maintainer “Alex”**

  * Maintains 20–200 skills in nested directories.
  * Frequently re-runs conversions for new targets (e.g., gemini, claude).
  * Needs a clear overview of what has been processed and what failed.

* **Tooling engineer “Jordan”**

  * Integrates `skill-porter-tui` into CI or standardized local workflows.
  * Wants deterministic behavior, clear exit codes, and scriptability.

### Key Flows

1. **Quick single-directory conversion**

   * Alex runs `skill-porter-tui --root ./skills/my-skill --no-recursive --target gemini`.
   * TUI loads a single skill, displays it as `pending`.
   * Alex presses `c` to convert; status becomes `running` then `success`.
   * Summary shows `Succeeded: 1, Skipped: 0, Failed: 0`.

2. **Full-tree recursive conversion**

   * Alex runs `skill-porter-tui --root ./skills`.
   * TUI discovers all skills; list shows them as `pending`.
   * Alex navigates with `j/k` to inspect entries; triggers conversions for some manually, then enables auto-convert for remaining.
   * Failed entries show as `failed` with error snippets; Alex inspects and decides which to retry.

3. **Error handling**

   * `skill-porter` missing: on start, tool detects absence and shows an error banner similar to `convert-all-skills.sh` behavior and exits with non-zero.
   * Per-skill failures: failed status with last error line shown in detail panel.

### UI/UX Notes

* Prefer immediate, at-a-glance visibility:

  * Status icons on each row.
  * Global summary always visible in footer.
* TUI layout:

  * Header: tool name, default target, mode (recursive/single), root path.
  * Main area: list of skills, multi-line rows where necessary.
  * Side/bottom: detail panel with human-readable path and output directory.
* Key hints:

  * Show a minimal key legend in footer (`↑/↓ j/k move, c/g/a convert, s skip, q quit, r rescan`).
* Ensure UI remains responsive on large skill sets via throttled rendering if necessary (handled in implementation).

---

7. Technical Architecture

---

### System Components

* **CLI entry (Go main)**

  * Parses flags, creates `Config`, launches Bubble Tea program.

* **Domain layer**

  * Type definitions for skills, statuses, targets, summaries.

* **Discovery layer**

  * Filesystem traversal and skill detection.

* **Conversion layer**

  * Command builder and process runner wrapping `skill-porter convert`.

* **TUI core**

  * Bubble Tea model (state + update loop).
  * Integration with discovery and conversion via Tea commands.

* **TUI presentation**

  * View rendering using Lip Gloss.
  * Key handling and view composition.

* **Existing bash script**

  * `scripts/convert-all-skills.sh` remains unchanged and can be used independently.

### Data Models

* `Config`

  * `ScanRoot string`
  * `Recursive bool`
  * `DefaultTarget ConversionTarget`
  * `OutBaseDir string`

* `SkillDir`

  * `Path string`
  * `Name string`
  * `HasSKILL bool`
  * `HasGeminiExtension bool`

* `SkillState`

  * `Skill SkillDir`
  * `Status ConversionStatus`
  * `SelectedTarget ConversionTarget`
  * `OutputPath string`
  * `LastErrorMessage string`

* `Summary`

  * `Success int`
  * `Skipped int`
  * `Failed int`
  * `Total int`

### Technology Stack

* **Language**: Go.
* **TUI Framework**: Bubble Tea (program loop) + Lip Gloss (styling).
* **External tools**: `skill-porter` CLI (required), `gum` (optional for confirmations in Phase 4).

### Key Decisions

**Decision: Implement conversion logic in Go, not via `convert-all-skills.sh` subprocess**

* **Rationale**:

  * Avoid piping interactive gum-based script from a TUI.
  * Reduce coupling and complexity; treat the script as a reference and legacy CLI.
  * Align with “no god modules” / one responsibility per module.
* **Trade-offs**:

  * Duplication of some semantics (AUTO_CONVERT_MODE behavior) between Go and bash.
* **Alternatives**:

  * Add non-interactive, JSON-output mode to `convert-all-skills.sh` and call it from Go (rejected for MVP due to complexity and tight coupling).

**Decision: Sequential conversions for MVP**

* **Rationale**:

  * Simpler mapping to existing script semantics.
  * Avoid complexity of parallel process management and UI concurrency.
* **Trade-offs**:

  * Less throughput for large trees.
* **Alternatives**:

  * Configurable concurrency with a worker pool (can be added later, but not required for initial PRD).

---

8. Test Strategy

---

### Test Pyramid

```
        /\
       /E2E\        ← ~10% (manual and scripted terminal runs)
      /------\
     /Integration\  ← ~30% (TUI model + engine interaction)
    /------------\
   /  Unit Tests  \← ~60% (config, discovery, conversion, model update)
  /----------------\
```

### Coverage Requirements

* Line coverage (non-UI core packages): ≥80%.
* Branch coverage (discovery and conversion logic): ≥80%.
* Function coverage: ≥90% for exported functions in `config`, `domain`, `discovery`, `convert`, `tui/model`.
* Statement coverage: track as part of line coverage; ensure all error paths in discovery and conversion are exercised.

### Critical Test Scenarios

#### Module: `discovery`

* **Happy path**:

  * Recursive scan of a tree with multiple skills; ensure all are discovered once.
  * Non-recursive mode on a single skill directory.
* **Edge cases**:

  * Directories with both `SKILL.md` and `gemini-extension.json`.
  * Very deep directory nesting.
* **Error cases**:

  * Non-existent root path.
  * Permission-denied path.
* **Integration points**:

  * `DiscoverSkills` used by TUI `Init`—ensure the model handles empty results.

#### Module: `convert`

* **Happy path**:

  * `BuildCommand` for typical skill names and targets.
  * Successful `RunConversion` with simulated `skill-porter`.
* **Edge cases**:

  * Skill name with spaces or unusual characters.
* **Error cases**:

  * Missing `skill-porter` binary.
  * Non-zero exit code from `skill-porter` with stderr.
* **Integration points**:

  * Correct status transitions and summary updates in `tui/model` on success/failure.

#### Module: `tui/model`

* **Happy path**:

  * Sequence: `Init` → `SkillsDiscoveredMsg` → `StartConversionMsg` → `SkillConvertedMsg`.
* **Edge cases**:

  * Navigation beyond list edges.
  * Converting already-successful skill (should be a no-op).
* **Error cases**:

  * Handling `ConversionErrorMsg` and preserving `LastErrorMessage`.
* **Integration points**:

  * Summary updates after each change.

### Test Generation Guidelines

* Preference for deterministic tests:

  * Avoid depending on real filesystem where possible; use temporary directories.
  * Inject process runner interfaces to avoid invoking real `skill-porter` in unit tests.
* Keep test modules aligned with feature slices:

  * No cross-feature “god” tests aggregating unrelated logic.
* For TUI:

  * Use state-level tests for model logic.
  * Reserve manual tests for terminal rendering and UX details.

---

9. Risks and Mitigations

---

### Technical Risks

**Risk**: Divergence between Go orchestration and bash script semantics

* **Impact**: Medium (confusing behavior differences).
* **Likelihood**: Medium.
* **Mitigation**:

  * Use `convert-all-skills.sh` as a reference test case and manually compare behavior on sample trees.
* **Fallback**:

  * If divergence becomes problematic, add a script-based non-interactive mode and call it from Go for authoritative behavior.

**Risk**: TUI performance on large skill trees

* **Impact**: Medium.
* **Likelihood**: Medium.
* **Mitigation**:

  * Keep rendering efficient; avoid re-rendering entire list on every minor update.
  * Consider batched updates if needed.
* **Fallback**:

  * Offer a “summary-only” or non-interactive mode for very large runs.

**Risk**: Terminal compatibility

* **Impact**: Low–Medium.
* **Likelihood**: Medium.
* **Mitigation**:

  * Test on common terminals (macOS Terminal/iTerm2, Linux shells).
  * Favor simple, robust ANSI styling.
* **Fallback**:

  * Provide a `--no-style` or minimal mode if necessary.

### Dependency Risks

**Risk**: `skill-porter` CLI breaking changes

* **Impact**: High (core conversions fail).
* **Likelihood**: Low–Medium.
* **Mitigation**:

  * Keep all interactions limited to `skill-porter convert` command and document expectations.
* **Fallback**:

  * Adjust conversion command builder quickly; tool remains otherwise intact.

**Risk**: Missing gum for optional confirmations

* **Impact**: Low (only affects optional gum usage).
* **Likelihood**: Medium.
* **Mitigation**:

  * Treat gum strictly as optional; TUI confirmations are primary.
* **Fallback**:

  * Disable gum interoperability gracefully.

### Scope Risks

**Risk**: Over-expansion of TUI features (multi-target, advanced filters)

* **Impact**: Medium.
* **Likelihood**: Medium.
* **Mitigation**:

  * Keep MVP scope tightly aligned with: configuration, discovery, per-skill actions, summary.
* **Fallback**:

  * Push advanced features into future phases without blocking core delivery.

**Risk**: Violating module-design constraints (creating “god modules”)

* **Impact**: High (technical debt; violates environment rules).
* **Likelihood**: Medium.
* **Mitigation**:

  * Enforce feature slices and single responsibility during code reviews.
* **Fallback**:

  * Refactor early; do not add new behavior into mixed-responsibility files.

---

10. Appendix

---

### References

* `scripts/convert-all-skills.sh` – existing bash script performing discovery, interactive gum prompts, conversion, and summary counts.
* `SKILL.md` – example of skill root structure referenced during discovery (presence of `SKILL.md` as marker).
* System module design rules – constraints on module responsibilities and feature-based organization.
* General rules – early-stage project, no workarounds, no tech debt, maintain existing features.
* RPG PRD template and Task Master integration.

### Glossary

* **Skill**: A directory containing `SKILL.md` and/or `gemini-extension.json`, representing a unit for conversion.
* **Target**: Conversion destination (e.g., `gemini`, `claude`).
* **TUI**: Text-based User Interface, here implemented with Bubble Tea and Lip Gloss.
* **AUTO_CONVERT_MODE**: Mode where remaining skills are automatically converted with a chosen target.

### Open Questions

* Should the TUI support concurrent conversions out of the box or only sequential for now?
* Should a future phase add a programmatic, non-interactive mode (e.g., JSON output) for CI workflows?
* Do we eventually deprecate or simplify `convert-all-skills.sh` once the TUI is stable, or keep both indefinitely?
