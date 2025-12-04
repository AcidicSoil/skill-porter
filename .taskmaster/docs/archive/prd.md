1. Overview

---

### Problem Statement

`skill-porter` today supports converting skills between Claude and Gemini ecosystems. Codex CLI is emerging as a third major platform, but there is no automated, spec-driven way to generate Codex-compatible plugins from existing Claude/Gemini skills. This forces manual, error-prone translation of manifests, agents documentation, and MCP server configuration, and blocks “universal” skills that run across all three CLIs.

The product gap:

* No standard on-disk Codex plugin format within skill repos.
* No platform detection/validation for Codex artifacts.
* No Claude→Codex or Gemini→Codex converters.
* No unified “universal” flow that guarantees Claude + Gemini + Codex artifacts exist.

### Target Users

* Skill Authors

  * Own one or more Claude/Gemini skills.
  * Want Codex support without learning Codex internals.
  * Prefer running a single conversion command to generate Codex artifacts.

* Platform Integrators / Tooling Authors

  * Maintain multi-platform tools that depend on consistent skill formats.
  * Want a predictable Codex plugin directory layout and manifest schema.

* Codex Users

  * Want to install skills as plugins with minimal manual configuration.
  * Expect clear “next steps” instructions after conversion.

### Why Existing Solutions Fail

* Manual conversion is non-standardized and repetitive.
* Each skill author re-discovers Codex config semantics.
* No single, reusable Codex plugin directory convention across repos.
* No validation pipeline for Codex-specific problems before publish/usage.

### Constraints and Non-Goals

* No changes to Codex core behavior or APIs; integration is config/file-based only (config.toml + AGENTS-like docs).
* No bespoke Codex RPC protocol; only TOML/Markdown and MCP server wiring.
* `skill-porter` remains CLI/tooling only; Codex remains the runtime.

### Success Metrics

* ≥ 80% of existing Claude or Gemini skills in a representative sample convert to a working Codex plugin with zero manual edits.
* ≤ 10% of conversions produce validation errors that require manual intervention (excluding missing environment variables).
* ≥ 90% of converted Codex plugins pass a scripted smoke test (Codex CLI + MCP connectivity).
* “Universal” flow produces all three sets of artifacts (Claude, Gemini, Codex) in a single pass for ≥ 90% of repos tested.

---

2. Capability Tree (Functional Decomposition)

---

This section defines WHAT the system does. MVP-critical features are marked `[MVP]`.

### Capability: Codex Plugin Specification Management

Defines and enforces a standard Codex plugin directory, manifest, and config conventions.

#### Feature: Codex Plugin Directory Layout [MVP]

* **Description**: Define a canonical `codex/` directory structure and resolution rules for Codex plugins in a skill repo.
* **Inputs**:

  * Repo root path
  * Existing Claude/Gemini artifacts (optional)
* **Outputs**:

  * In-memory representation of plugin root and file paths:

    * `codex/codex-plugin.toml`
    * `codex/CODEX_AGENTS.md`
    * `codex/codex-config.snippet.toml`
    * `codex/docs/CODEX_ARCHITECTURE.md`
* **Behavior**:

  * Locate or propose creation of `codex/` at repo root (or sibling of `.claude/` / Gemini artifacts).
  * Normalize all Codex-related paths relative to repo root.
  * Provide consistent defaults when files are missing (for converters to create).

#### Feature: Codex Plugin Manifest Schema (`codex-plugin.toml`) [MVP]

* **Description**: Define and validate the TOML schema for Codex plugin metadata and capabilities.
* **Inputs**:

  * Raw `codex-plugin.toml` TOML text
  * Optional defaults (e.g., repo URL, license)
* **Outputs**:

  * Parsed manifest object with typed fields:

    * `plugin.id`, `display_name`, `description`, `version`, `homepage`, `source_repo`, `license`
    * `agents_file`, `config_snippet_file`
    * `plugin.capabilities` flags (`uses_mcp`, `reads_files`, `writes_files`, `runs_shell`)
  * Validation result with errors/warnings.
* **Behavior**:

  * Parse TOML and validate presence/format of required fields (e.g. `id` slug `[a-z0-9-]+`).
  * Populate default values when safe (e.g., infer homepage from Git remote).
  * Provide structured diagnostics for missing/invalid fields.

#### Feature: Codex Agents Documentation Convention (`CODEX_AGENTS.md`) [MVP]

* **Description**: Standardize structure and content expectations of `CODEX_AGENTS.md`.
* **Inputs**:

  * Markdown content (possibly generated from Claude/Gemini docs)
  * Plugin metadata (name, description)
  * MCP server list
* **Outputs**:

  * Normalized markdown AST or template model for:

    * H1 title with plugin name
    * Summary paragraph
    * Operational guidance sections
    * “MCP servers” section listing required servers
* **Behavior**:

  * Enforce presence of H1 and summary.
  * Ensure an explicit “MCP servers” section is present and synchronized with config snippet.
  * Provide helper to generate from existing SKILL/GEMINI docs with minimal Codex-specific content.

#### Feature: Codex Config Snippet Convention (`codex-config.snippet.toml`) [MVP]

* **Description**: Define required structure and generation rules for Codex config snippets.
* **Inputs**:

  * MCP server definitions (names, command/url, env variables)
  * Plugin capabilities (which tools/servers are used)
* **Outputs**:

  * TOML snippet with:

    * Instruction comment `# Append this to ~/.codex/config.toml`
    * `[mcp_servers.*]` sections for each server
    * Optional `enabled_tools` / `disabled_tools`
* **Behavior**:

  * Map each MCP server to either STDIO (`command`, `args`, `cwd`, `env`, `env_vars`) or HTTP (`url`, `bearer_token_env_var`, `http_headers`).
  * Ensure at least one MCP section exists when `uses_mcp = true`.
  * Automatically restrict tools when allowed/excluded tool lists exist.

#### Feature: Plugin Root Resolution Rules [MVP]

* **Description**: Determine where to create/discover `codex/` in mixed-platform repos.
* **Inputs**:

  * Repo directory tree
  * Detection of `.claude/`, `gemini-extension.json`, etc.
* **Outputs**:

  * Resolved Codex plugin root path
  * Resolution strategy description (for diagnostics)
* **Behavior**:

  * Prefer `./codex/` at repo root.
  * If conflicting candidates exist, choose deterministic winner and emit warning.
  * Guarantee relative paths inside `codex-config.snippet.toml` are repo-relative, never absolute.

---

### Capability: Platform Detection and Validation

Identify if a repo targets Codex and validate its Codex artifacts alongside Claude/Gemini.

#### Feature: Codex Platform Detection in PlatformDetector [MVP]

* **Description**: Detect Codex artifacts and classify platform type as `codex` or `universal`.
* **Inputs**:

  * Repo root path
  * File system snapshot (names and paths)
* **Outputs**:

  * `metadata.codex` structure containing:

    * Present Codex files
    * Parsed manifest summary (id, display_name, version, paths)
  * Platform classification: `'codex'`, `'claude'`, `'gemini'`, `'universal'`
* **Behavior**:

  * Search standard locations for `codex-plugin.toml`, `CODEX_AGENTS.md`, `codex-config.snippet.toml`.
  * Set `platform = 'codex'` when only Codex artifacts exist.
  * Set `platform = 'universal'` when Claude/Gemini/Codex artifacts appear in combination.
  * Emit a concise Codex summary in `analyze` output.

#### Feature: Codex Validation Branch [MVP]

* **Description**: Add Codex-specific validation to the existing validator pipeline.
* **Inputs**:

  * Repo root
  * Parsed Codex manifest and config snippet
  * Detected files from PlatformDetector
* **Outputs**:

  * Structured validation results:

    * Errors (blocking)
    * Warnings (non-blocking)
* **Behavior**:

  * Validate presence and parseability of `codex-plugin.toml`.
  * Ensure required fields and formats.
  * Resolve `agents_file` and `config_snippet_file` paths and assert existence.
  * Parse `codex-config.snippet.toml` and verify at least one `[mcp_servers.*]`.
  * Emit warnings when `uses_mcp = true` but MCP servers are missing.

#### Feature: Universal Platform Validation [MVP]

* **Description**: Validate Codex artifacts when platform is `'universal'`.
* **Inputs**:

  * Combined metadata for Claude, Gemini, Codex
* **Outputs**:

  * Aggregate validation results for all platforms
* **Behavior**:

  * Run `_validateCodex()` whenever Codex artifacts are present, regardless of primary platform.
  * Cross-check MCP server definitions across platforms for consistency (names, URLs, commands).
  * Include Codex diagnostics in consolidated validation output.

---

### Capability: Skill Conversion (Claude/Gemini → Codex)

Convert existing Claude and Gemini skills into Codex plugin artifacts.

#### Feature: Claude→Codex Converter [MVP]

* **Description**: Generate a Codex plugin from a Claude skill (`SKILL.md` + MCP metadata).
* **Inputs**:

  * Parsed `SKILL.md` with YAML frontmatter (name, description, allowed-tools)
  * Existing Claude MCP server metadata as exposed by skill-porter analyzers
  * Target plugin root path (`codex/`)
* **Outputs**:

  * `codex/codex-plugin.toml`
  * `codex/CODEX_AGENTS.md`
  * `codex/codex-config.snippet.toml`
  * `codex/docs/CODEX_ARCHITECTURE.md`
  * Conversion summary object (paths, counts, warnings)
* **Behavior**:

  * Map frontmatter `name` to `plugin.id` (slug) and `display_name`.
  * Map `description` to manifest description and agent summary.
  * Infer `plugin.capabilities` from allowed tools and MCP metadata.
  * Generate Codex agent docs by stripping Claude-only parts from SKILL content.
  * Translate each MCP server into `[mcp_servers.*]` blocks, including env var hints.
  * Map Claude `allowed-tools` into Codex `enabled_tools` / `disabled_tools` as applicable.

#### Feature: Gemini→Codex Converter [MVP]

* **Description**: Generate a Codex plugin from a Gemini extension (`gemini-extension.json` + `GEMINI.md`).
* **Inputs**:

  * `gemini-extension.json` (name, description, version, mcpServers, excludeTools, settings)
  * `GEMINI.md` usage docs
  * Derived MCP metadata and commands from existing analyzers
* **Outputs**:

  * `codex/codex-plugin.toml`
  * `codex/CODEX_AGENTS.md`
  * `codex/codex-config.snippet.toml`
  * `codex/docs/CODEX_ARCHITECTURE.md`
  * Conversion summary
* **Behavior**:

  * Map extension `name` to `plugin.id` and `display_name`.
  * Copy `description` and `version` into manifest.
  * Derive capabilities from `mcpServers` and extension settings.
  * Derive Codex-centric agent docs from `GEMINI.md` by removing Gemini CLI install/runtime sections.
  * Translate each `mcpServers` entry into `[mcp_servers.*]` with correct mode (STDIO/HTTP).
  * Use `excludeTools` and inferred tool list to compute `enabled_tools`, if possible.

#### Feature: Codex Architecture Docs Generation

* **Description**: Generate `docs/CODEX_ARCHITECTURE.md` explaining how the plugin fits into Codex/MCP.
* **Inputs**:

  * Plugin manifest
  * MCP server definitions
  * Source platform metadata (Claude or Gemini)
* **Outputs**:

  * `codex/docs/CODEX_ARCHITECTURE.md` markdown file
* **Behavior**:

  * Describe how Codex uses `AGENTS.md` and MCP servers.
  * Explain mapping between source platform concepts and Codex concepts.
  * Link to other docs in the repo (SKILL.md, GEMINI.md, etc.).

---

### Capability: CLI and Universal Flow Integration

Expose Codex capabilities via the `skill-porter` CLI.

#### Feature: `--to codex` Target Support [MVP]

* **Description**: Extend `convert` command to accept `codex` as a valid output target.
* **Inputs**:

  * CLI flags: `convert --to codex`
  * Detected source platform (Claude or Gemini)
* **Outputs**:

  * Generated Codex plugin artifacts on disk
  * CLI logs and summary (success/warnings)
* **Behavior**:

  * Route Claude sources to `ClaudeToCodexConverter`.
  * Route Gemini sources to `GeminiToCodexConverter`.
  * Preserve existing behavior for other `--to` values.
  * Fail fast with explicit message when source is unsupported.

#### Feature: Codex-Specific Post-Conversion Guidance [MVP]

* **Description**: Print clear next steps for using Codex plugin after conversion.
* **Inputs**:

  * Plugin root path
  * Generated config snippet path
* **Outputs**:

  * CLI text instructions, e.g.:

    * Append snippet to `~/.codex/config.toml`
    * Merge/point Codex to `CODEX_AGENTS.md`
* **Behavior**:

  * Print deterministic, copy-paste-friendly snippets.
  * Include both “merge into AGENTS.md” and “add to project_doc_fallback_filenames” options.

#### Feature: Tri-Platform “Universal” Flow [MVP]

* **Description**: Extend `universal` command to cover Claude, Gemini, and Codex.
* **Inputs**:

  * Repo root
  * Detected platform artifacts
* **Outputs**:

  * Missing platform artifacts generated (including Codex)
  * Consolidated summary stating repo is “universal”
* **Behavior**:

  * If Gemini artifacts missing → `convert --to gemini`.
  * If Claude artifacts missing → `convert --to claude`.
  * If Codex artifacts missing → `convert --to codex`.
  * Exit success only if all three platforms are present and validated.

---

### Capability: Documentation and Examples

#### Feature: Codex Support README Section

* **Description**: Document Codex support, usage, and limitations in the main README.
* **Inputs**:

  * Codex spec and CLI behavior
* **Outputs**:

  * README updates: “Codex CLI Support” section
* **Behavior**:

  * Outline inputs, outputs, commands, and known caveats.
  * Provide minimal quickstart example for Codex users.

#### Feature: Example `examples/codex-basic/` Repo

* **Description**: Provide a minimal example repo that demonstrates Claude + Gemini + Codex artifacts end-to-end.
* **Inputs**:

  * Reference skill content
* **Outputs**:

  * `examples/codex-basic/` directory with fully wired MCP server and commands
* **Behavior**:

  * Include a working MCP server (or mock) with Codex config snippet.
  * Ensure automated tests exercise this example.

---

### Capability: Reverse Conversion and Helper Tooling (Optional)

#### Feature: Codex→Claude Converter

* **Description**: Convert Codex plugin artifacts back into Claude skill artifacts.
* **Inputs**:

  * `codex-plugin.toml`, `CODEX_AGENTS.md`, `codex-config.snippet.toml`
* **Outputs**:

  * `SKILL.md`, `.claude-plugin/marketplace.json`, `.claude/commands/*`
* **Behavior**:

  * Reverse mapping of manifest and MCP config.
  * Use heuristics for reconstructing Claude-specific metadata.

#### Feature: Codex→Gemini Converter

* **Description**: Convert Codex plugin artifacts into Gemini extension format.
* **Inputs**:

  * Same as Codex→Claude feature
* **Outputs**:

  * `gemini-extension.json`, `GEMINI.md`, `commands/*`
* **Behavior**:

  * Reverse mapping of plugin metadata, MCP servers, and tools.

#### Feature: External `codex-plugins` Helper CLI

* **Description**: Optional separate CLI for installing/removing Codex plugins from user environment.
* **Inputs**:

  * Path to `codex/` directory
  * User’s `~/.codex/config.toml` path
* **Outputs**:

  * Plugin copied to `~/.codex/plugins/<id>/`
  * Config snippet appended idempotently
* **Behavior**:

  * `install`, `list`, `remove` commands.
  * Intentionally outside `skill-porter` repo.

---

3. Repository Structure + Module Definitions (Structural Decomposition)

---

This section defines HOW the code is organized. Capabilities map to modules; features to functions/classes. Layout assumes TypeScript Node CLI.

### Proposed Repository Structure (incremental additions)

```text
project-root/
├── src/
│   ├── spec/
│   │   └── codexPluginSpec.ts
│   ├── analyzers/
│   │   ├── platformDetector.ts         # extend for Codex
│   │   └── codexMetadata.ts
│   ├── validation/
│   │   └── codexValidator.ts
│   ├── converters/
│   │   ├── claudeToCodexConverter.ts
│   │   └── geminiToCodexConverter.ts
│   ├── cli/
│   │   ├── convertCommand.ts
│   │   └── universalCommand.ts
│   ├── docs/
│   │   └── codexDocsGenerator.ts
│   └── core/
│       ├── fs.ts
│       ├── toml.ts
│       └── logging.ts
├── examples/
│   └── codex-basic/
├── tests/
│   ├── spec/
│   ├── analyzers/
│   ├── validation/
│   ├── converters/
│   └── cli/
└── docs/
    └── codex-support.md
```

### Module Definitions

#### Module: `src/core/fs.ts`

* **Maps to capability**: Cross-cutting I/O support (foundation).
* **Responsibility**: File system abstraction for reading/writing/ensuring directories.
* **File structure**: Single file.
* **Exports**:

  * `readFile(path): Promise<string>`
  * `writeFile(path, contents): Promise<void>`
  * `ensureDir(path): Promise<void>`
  * `pathExists(path): Promise<boolean>`

#### Module: `src/core/toml.ts`

* **Maps to capability**: Parsing/serializing TOML.
* **Responsibility**: Wrap TOML library with consistent error handling.
* **Exports**:

  * `parseToml<T>(text: string): T`
  * `stringifyToml(obj: unknown): string`
  * `TomlError` (error type)

#### Module: `src/core/logging.ts`

* **Maps to capability**: Diagnostics/log output.
* **Responsibility**: Structured logging for CLI and internal steps.
* **Exports**:

  * `logInfo(message: string, meta?)`
  * `logWarn(message: string, meta?)`
  * `logError(message: string, meta?)`

---

#### Module: `src/spec/codexPluginSpec.ts`

* **Maps to capability**: Codex Plugin Specification Management.
* **Responsibility**: Canonical in-memory representation and helpers for Codex plugin spec.
* **File structure**: Single file.
* **Exports**:

  * Types: `CodexPluginManifest`, `CodexCapabilities`, `CodexPluginLayout`
  * `defaultCodexLayout(repoRoot: string): CodexPluginLayout`
  * `parseCodexManifest(text: string): CodexPluginManifest`
  * `validateCodexManifest(manifest: CodexPluginManifest): ValidationResult`
  * `inferCapabilitiesFromTools(tools: string[]): CodexCapabilities`

---

#### Module: `src/analyzers/platformDetector.ts` (existing, extended)

* **Maps to capability**: Platform Detection and Validation.
* **Responsibility**: Detect Claude, Gemini, Codex artifacts and compute `platform` classification.
* **Exports**:

  * `detectPlatform(repoRoot: string): Promise<PlatformMetadata>`

    * Platform types extended to `'codex'` and `'universal'`.

#### Module: `src/analyzers/codexMetadata.ts`

* **Maps to capability**: Codex Platform Detection (Codex-specific).
* **Responsibility**: Locate and parse Codex-specific files, return Codex metadata.
* **Exports**:

  * `detectCodexFiles(repoRoot: string): Promise<CodexMetadata>`
  * `summarizeCodexPlugin(meta: CodexMetadata): CodexSummary`

---

#### Module: `src/validation/codexValidator.ts`

* **Maps to capability**: Codex Validation Branch.
* **Responsibility**: Validate Codex artifacts and attach diagnostics to global validation result.
* **Exports**:

  * `validateCodex(repoRoot: string, meta: CodexMetadata): Promise<ValidationResult>`

---

#### Module: `src/converters/claudeToCodexConverter.ts`

* **Maps to capability**: Claude→Codex Converter.
* **Responsibility**: Implement Claude→Codex conversion pipeline.
* **Exports**:

  * `convertClaudeToCodex(options: ConvertClaudeToCodexOptions): Promise<ConversionResult>`

#### Module: `src/converters/geminiToCodexConverter.ts`

* **Maps to capability**: Gemini→Codex Converter.
* **Responsibility**: Implement Gemini→Codex conversion pipeline.
* **Exports**:

  * `convertGeminiToCodex(options: ConvertGeminiToCodexOptions): Promise<ConversionResult>`

---

#### Module: `src/cli/convertCommand.ts` (existing, extended)

* **Maps to capability**: `--to codex` Target Support.
* **Responsibility**: CLI handler for `convert` command and routing to converters.
* **Exports**:

  * `registerConvertCommand(program): void`

---

#### Module: `src/cli/universalCommand.ts` (existing, extended)

* **Maps to capability**: Tri-Platform “Universal” Flow.
* **Responsibility**: Ensure repo has Claude + Gemini + Codex artifacts by chaining conversions.
* **Exports**:

  * `registerUniversalCommand(program): void`

---

#### Module: `src/docs/codexDocsGenerator.ts`

* **Maps to capability**: Codex Architecture Docs Generation.
* **Responsibility**: Generate `CODEX_AGENTS.md` and `CODEX_ARCHITECTURE.md` content.
* **Exports**:

  * `generateCodexAgentsDoc(context: AgentsDocContext): string`
  * `generateCodexArchitectureDoc(context: ArchitectureDocContext): string`

---

4. Dependency Chain

---

### Foundation Layer (Phase 0)

No dependencies.

* **core/fs**: Basic file system operations.
* **core/toml**: TOML parsing/serialization.
* **core/logging**: Logging utilities.

### Specification Layer (Phase 1)

Depends on foundation.

* **spec/codexPluginSpec**

  * Depends on: `[core/fs, core/toml, core/logging]` (for validation and error reporting).

### Analysis Layer (Phase 1–2)

* **analyzers/platformDetector**

  * Depends on: `[core/fs, spec/codexPluginSpec]`
* **analyzers/codexMetadata**

  * Depends on: `[core/fs, core/toml, spec/codexPluginSpec]`

### Validation Layer (Phase 2)

* **validation/codexValidator**

  * Depends on: `[core/fs, core/toml, spec/codexPluginSpec, analyzers/codexMetadata]`

### Conversion Layer (Phase 2–3)

* **converters/claudeToCodexConverter**

  * Depends on: `[core/fs, core/toml, spec/codexPluginSpec, analyzers/codexMetadata, docs/codexDocsGenerator]`
* **converters/geminiToCodexConverter**

  * Depends on: `[core/fs, core/toml, spec/codexPluginSpec, analyzers/codexMetadata, docs/codexDocsGenerator]`

### CLI Layer (Phase 3)

* **cli/convertCommand**

  * Depends on: `[converters/claudeToCodexConverter, converters/geminiToCodexConverter, analyzers/platformDetector, validation/codexValidator, core/logging]`
* **cli/universalCommand**

  * Depends on: `[cli/convertCommand, analyzers/platformDetector, validation/codexValidator]`

### Documentation & Examples (Phase 4)

* **docs/codexDocsGenerator**

  * Depends on: `[spec/codexPluginSpec]`
* **examples/codex-basic**

  * Depends on: `[cli/universalCommand, converters/*, validation/*]`

### Optional Reverse/Helper Tools (Phase 5)

* **converters/codexToClaudeConverter**

  * Depends on: `[core/fs, core/toml, spec/codexPluginSpec]`
* **converters/codexToGeminiConverter**

  * Depends on: `[core/fs, core/toml, spec/codexPluginSpec]`
* **external codex-plugins CLI** (separate repo)

  * Depends on: `[core/fs, core/toml]` equivalents in its own codebase.

No cycles: all dependencies move upward from foundation toward CLI and optional tools.

---

5. Development Phases

---

Each phase is dependency-driven. No dates.

### Phase 0: Foundation

**Goal**: Establish shared core utilities for file I/O, TOML, and logging.

**Entry Criteria**: Existing `skill-porter` repository; no Codex-specific code required.

**Tasks**:

* [ ] Implement `src/core/toml.ts` (depends on: none)

  * Acceptance criteria: TOML parsing and serialization functions tested with valid/invalid manifests.
  * Test strategy: Unit tests for success/error cases; golden tests for round-tripping simple TOML objects.

* [ ] Confirm or implement `src/core/fs.ts` and `src/core/logging.ts` as reusable modules (depends on: none)

  * Acceptance criteria: All filesystem and logging usages in later phases call these modules.
  * Test strategy: Unit tests using temp directories; verify logging output structure.

**Exit Criteria**: All future modules can rely on shared core utilities without ad-hoc I/O or TOML handling.

**Delivers**: Stable base for spec, analyzers, validators, and converters.

---

### Phase 1: Codex Spec & Detection

**Goal**: Define Codex plugin spec and wire Codex into platform detection.

**Entry Criteria**: Phase 0 complete.

**Tasks**:

* [ ] Implement `spec/codexPluginSpec.ts` (depends on: core/fs, core/toml)

  * Acceptance criteria:

    * Data types for manifest and capabilities exist.
    * `parseCodexManifest` and `validateCodexManifest` produce expected diagnostics for sample manifests.
  * Test strategy: Unit tests with valid/invalid manifests; slug validation tests; missing fields checks.

* [ ] Implement `analyzers/codexMetadata.ts` (depends on: core/fs, core/toml, spec/codexPluginSpec)

  * Acceptance criteria:

    * Correctly finds Codex files in standard locations.
    * Returns structured metadata with manifest summary when present.
  * Test strategy: Unit tests over synthetic repo layouts.

* [ ] Extend `analyzers/platformDetector.ts` for Codex (depends on: analyzers/codexMetadata)

  * Acceptance criteria:

    * Correctly classifies repos as `codex`, `claude`, `gemini`, or `universal` given fixture sets.
    * `analyze` output shows Codex summary when applicable.
  * Test strategy: Unit tests with multiple repo fixtures; integration test for `analyze` command output.

**Exit Criteria**: Repos with Codex artifacts are detectable and classified correctly; spec module is stable.

**Delivers**: `analyze` surface that understands Codex.

---

### Phase 2: Codex Validation

**Goal**: Validate Codex artifacts and surface diagnostics alongside existing validators.

**Entry Criteria**: Phase 1 complete.

**Tasks**:

* [ ] Implement `validation/codexValidator.ts` (depends on: spec/codexPluginSpec, analyzers/codexMetadata)

  * Acceptance criteria:

    * Repos with missing or invalid Codex manifests produce structured errors.
    * `uses_mcp` without `[mcp_servers.*]` emits warning.
  * Test strategy: Unit tests over fixtures; integration tests that run global `validate` and check JSON output.

* [ ] Integrate Codex validator into global `validate()` function (depends on: validation/codexValidator)

  * Acceptance criteria:

    * For platform `codex` or `universal`, `_validateCodex` is invoked.
    * Codex diagnostics appear in global validation report without breaking existing behavior.
  * Test strategy: Integration tests for repos with and without Codex artifacts.

**Exit Criteria**: `validate` command surfaces Codex issues clearly; no regressions for Claude/Gemini validation.

**Delivers**: Safety net for Codex conversion and publish flows.

---

### Phase 3: Conversion and CLI Integration (MVP Completion)

**Goal**: Provide end-to-end Claude/Gemini → Codex conversion and CLI integration.

**Entry Criteria**: Phase 2 complete.

**Tasks**:

* [ ] Implement `ClaudeToCodexConverter` (depends on: spec/codexPluginSpec, analyzers/codexMetadata, docs/codexDocsGenerator, core/*)

  * Acceptance criteria:

    * Given a valid Claude skill, produces Codex plugin files in `codex/` with consistent naming.
    * Generated manifest passes `codexValidator`.
  * Test strategy: Golden-file tests comparing generated files against checked-in expectations; edge cases for missing fields.

* [ ] Implement `GeminiToCodexConverter` (depends on: same as above)

  * Acceptance criteria:

    * Given a valid Gemini extension, produces valid Codex artifacts.
  * Test strategy: Golden-file tests and integration tests via `convert --to codex`.

* [ ] Implement `docs/codexDocsGenerator.ts` (depends on: spec/codexPluginSpec)

  * Acceptance criteria:

    * Given basic metadata and MCP servers, generates valid `CODEX_AGENTS.md` and `CODEX_ARCHITECTURE.md`.
  * Test strategy: Snapshot/golden tests for generated markdown.

* [ ] Extend `cli/convertCommand.ts` with `--to codex` (depends on: converters/*, analyzers/platformDetector)

  * Acceptance criteria:

    * `skill-porter convert --to codex` routes to correct converter and exits with proper status codes.
  * Test strategy: CLI integration tests with temp repos.

* [ ] Add Codex-specific post-conversion guidance (depends on: cli/convertCommand)

  * Acceptance criteria:

    * CLI output matches specified “Next steps for Codex” template.
  * Test strategy: CLI output snapshot tests.

* [ ] Extend `cli/universalCommand.ts` to require Codex artifacts (depends on: cli/convertCommand, validation/codexValidator)

  * Acceptance criteria:

    * `universal` command ensures Claude + Gemini + Codex artifacts exist and validate.
  * Test strategy: End-to-end tests over multi-platform fixture repos.

**Exit Criteria**: A user with a Claude or Gemini skill can run `convert --to codex` and then follow printed instructions to use the plugin in Codex.

**Delivers**: MVP: Codex is a first-class platform with detection, validation, conversion, and CLI UX.

---

### Phase 4: Documentation and Examples

**Goal**: Document Codex support and provide a working example repo.

**Entry Criteria**: Phase 3 complete.

**Tasks**:

* [ ] Update README with “Codex CLI Support” section (depends on: CLI behavior)

  * Acceptance criteria:

    * README explains inputs/outputs, commands, and limitations for Codex.
  * Test strategy: Manual review plus link-checking.

* [ ] Create `examples/codex-basic/` (depends on: converters, cli/universalCommand)

  * Acceptance criteria:

    * Can be used in a scripted integration test to verify end-to-end behavior.
  * Test strategy: Automated example test runs convert + validate + Codex config generation.

* [ ] Add `docs/codex-support.md` or similar reference doc (depends on: spec, validation)

  * Acceptance criteria:

    * Outlines manifest schema, config conventions, and troubleshooting.

**Exit Criteria**: Codex support is discoverable through docs, and there is at least one fully tested example.

**Delivers**: Documentation and example to onboard new users.

---

### Phase 5: Optional Reverse Converters and Helper Tooling

**Goal**: Provide reverse conversion and external helper CLI (optional, non-MVP).

**Entry Criteria**: Phases 0–4 complete.

**Tasks**:

* [ ] Implement `CodexToClaudeConverter` (depends on: spec/codexPluginSpec)

  * Acceptance criteria:

    * Given valid Codex artifacts, can regenerate usable Claude artifacts that pass existing validation.
  * Test strategy: Round-trip tests: Claude → Codex → Claude.

* [ ] Implement `CodexToGeminiConverter` (depends on: spec/codexPluginSpec)

  * Acceptance criteria:

    * Similar round-trip tests for Gemini.

* [ ] Design and implement `codex-plugins` helper (in separate repo) (depends on: external)

  * Acceptance criteria:

    * Supports `install`, `list`, `remove`; works with generated `codex/` plugin directories.

**Exit Criteria**: Optional tools are available but not required for primary use cases.

**Delivers**: Better ecosystem ergonomics for Codex plugins.

---

6. User Experience

---

### Personas

* **Skill Author**: Uses `skill-porter` CLI; comfortable editing config files; wants minimal friction to support Codex.
* **Tooling Engineer**: Integrates skills into CI pipelines; expects machine-readable validation and predictable layouts.
* **Codex Power User**: Uses Codex CLI heavily; expects clear instructions and config snippets that “just work”.

### Key Flows

1. **Analyze a Repo for Codex Support**

   * User runs: `skill-porter analyze` at repo root.
   * `PlatformDetector` reports `platform = claude`, `gemini`, `codex`, or `universal` with Codex summary when applicable.
   * User can quickly see whether Codex artifacts exist and basic metadata (id, version, paths).

2. **Convert Claude Skill to Codex Plugin (MVP)**

   * Starting state: Claude-only skills with SKILL.md and MCP config.
   * User runs: `skill-porter convert --to codex`.
   * Converter generates `codex/` directory with all required files.
   * CLI prints “Next steps for Codex” including appending config snippet and pointing Codex to `CODEX_AGENTS.md`.

3. **Convert Gemini Extension to Codex Plugin (MVP)**

   * Starting state: Gemini-only extension.
   * Same flow as above with Gemini converter.

4. **Make Repo Universal Across All Three Platforms**

   * Starting state: Mixed or single-platform repo.
   * User runs: `skill-porter universal`.
   * Tool invokes missing conversions (including Codex).
   * Final summary declares repo “universal” and lists platform artifacts.

### UI/UX Notes

* Output must be concise and deterministic for automated parsing; structured JSON is already part of existing tooling and must include Codex sections.
* “Next steps” for Codex need to be copy/paste-ready commands or config fragments.
* Errors and warnings should explicitly name files (`codex/codex-plugin.toml`, etc.) and fields.

---

7. Technical Architecture

---

### System Components

* **CLI**: Existing Node-based `skill-porter` CLI extended with Codex options.
* **Spec & Validation Engine**: New Codex plugin spec module plus validators.
* **Analyzers**: Platform detector extended to Codex; new Codex metadata extractor.
* **Converters**: Two new conversion pipelines (Claude→Codex, Gemini→Codex).
* **Docs Generator**: Markdown generator for `CODEX_AGENTS.md` and architecture docs.
* **Examples**: Example repo wired into test suite.

### Data Models

* `CodexPluginManifest`

  * Fields: `plugin.id`, `display_name`, `description`, `version`, `homepage`, `source_repo`, `license`, `agents_file`, `config_snippet_file`, `capabilities`.
* `CodexCapabilities`

  * Booleans for `uses_mcp`, `reads_files`, `writes_files`, `runs_shell`.
* `CodexPluginLayout`

  * Paths for manifest, agents doc, config snippet, docs directory.
* `CodexMetadata`

  * Combined layout and parsed manifest, plus validation state.

### APIs and Integrations

* Internal API surface:

  * `convertClaudeToCodex(options)`, `convertGeminiToCodex(options)` for internal callers.
  * `validateCodex(repoRoot, meta)` for validators.
* External integration:

  * Reads/writes to local filesystem.
  * No network calls to Codex backend; integration is via files consumed by Codex CLI’s existing mechanisms.

### Infrastructure Requirements

* Same as existing `skill-porter` (Node, file permissions).
* No additional services or databases.

---

8. Test Strategy

---

### Test Pyramid

```text
        /\
       /E2E\        ≈ 10%
      /------\
     /Integration\  ≈ 30%
    /------------\
   /  Unit Tests  \ ≈ 60%
  /----------------\
```

### Coverage Requirements

* Line coverage: ≥ 85%
* Branch coverage: ≥ 80%
* Function coverage: ≥ 90%
* Statement coverage: ≥ 85%

### Critical Test Scenarios

#### Codex Plugin Spec

* **Happy path**:

  * Valid `codex-plugin.toml` parses and validates; slug, required fields, and capabilities accepted.
* **Edge cases**:

  * Missing optional fields (e.g., `homepage`).
  * Non-standard but slug-safe `id` variations.
* **Error cases**:

  * Invalid TOML syntax.
  * Missing required fields.
  * Invalid slug format.
* **Integration points**:

  * Spec used by converters and validators without type mismatches.

#### Codex Validation

* **Happy path**:

  * Complete Codex plugin passes validation with no errors.
* **Edge cases**:

  * `uses_mcp = false` with no MCP servers.
* **Error cases**:

  * `uses_mcp = true` but no `[mcp_servers.*]` tables.
  * Manifest references files that do not exist.
* **Integration points**:

  * `validate` command aggregates Codex diagnostics with other platform diagnostics.

#### Claude/Gemini → Codex Conversion

* **Happy path**:

  * Typical Claude/Gemini skills produce Codex artifacts that validate and are usable after config append.
* **Edge cases**:

  * Minimal skills with only one MCP server.
  * Skills with complex `allowed-tools` / `excludeTools`.
* **Error cases**:

  * Missing source files (e.g., SKILL.md not found).
  * Unsupported configuration combinations.
* **Integration points**:

  * Running `convert --to codex` in CI fixtures; verifying output and exit codes.

#### CLI Commands

* **Happy path**:

  * `convert --to codex` and `universal` succeed with expected logs and instructions.
* **Edge cases**:

  * Running in repos without any recognized platform artifacts.
* **Error cases**:

  * Unsupported `--to` values.
* **Integration points**:

  * Combined flows with analyze → convert → validate.

### Test Generation Guidelines

* Prefer table-driven tests for manifest and config validation.
* Use golden fixtures (on disk) for conversion outputs; assertions compare file content.
* For CLI tests, route output to buffers and assert on stable substrings rather than entire logs to avoid brittleness.
* Include at least one round-trip test per optional reverse converter.

---

9. Risks and Mitigations

---

### Technical Risks

1. **Risk**: Codex plugin spec or config semantics evolve.

   * **Impact**: Medium; may break existing plugins or require migration.
   * **Likelihood**: Medium.
   * **Mitigation**: Encapsulate spec in `codexPluginSpec` module; centralize mapping logic.
   * **Fallback**: Versioned spec support; add `schemaVersion` and migration helpers.

2. **Risk**: Heuristic mapping of capabilities and tool lists is inaccurate.

   * **Impact**: Low–Medium; non-optimal or broken plugins.
   * **Likelihood**: Medium.
   * **Mitigation**: Start conservative; default to broader capabilities when unsure; surface warnings.
   * **Fallback**: Allow manual overrides via follow-up edits; document limitations.

3. **Risk**: Complex source skills fail conversion due to edge-case constructs.

   * **Impact**: Medium; reduces metric on automatic conversion success.
   * **Likelihood**: Medium–High.
   * **Mitigation**: Test against diverse real-world skills; refine heuristics over time.
   * **Fallback**: Provide clear diagnostics pointing to manual fix locations.

### Dependency Risks

1. **Risk**: Tight coupling to internal Codex formats that are not fully stable.

   * **Impact**: Medium.
   * **Likelihood**: Medium.
   * **Mitigation**: Use public, documented Codex config and file structures only.
   * **Fallback**: Introduce adapters in `codexPluginSpec` if upstream changes.

2. **Risk**: Existing `skill-porter` architecture makes integration difficult.

   * **Impact**: Medium.
   * **Likelihood**: Low–Medium.
   * **Mitigation**: Keep Codex modules aligned with existing Claude/Gemini module patterns.

### Scope Risks

1. **Risk**: Optional reverse converters and `codex-plugins` helper expand scope beyond MVP.

   * **Impact**: High on timeline; low on MVP value.
   * **Likelihood**: Medium.
   * **Mitigation**: Keep them in Phase 5 and explicitly non-MVP.
   * **Fallback**: Drop optional tools from initial release.

2. **Risk**: Docs and examples lag behind implementation.

   * **Impact**: Medium; confuses users.
   * **Likelihood**: Medium–High.
   * **Mitigation**: Treat docs/example tasks as required exit criteria for phases.
   * **Fallback**: Ship minimal but accurate docs; defer polishing.

---

10. Appendix

---

### References

* Codex plugin extension plan and workstreams.
* RPG method for dependency-aware PRDs and Task Master integration.
* Generic PRD structure reference for sections and roadmap.

### Glossary

* **Codex CLI**: Target CLI platform consuming generated plugins.
* **Plugin**: A directory containing Codex-compatible manifest, agents docs, and config snippet.
* **MCP**: Model Context Protocol, for external tools/services used by skills.
* **Universal repo**: Repo with valid artifacts for Claude, Gemini, and Codex.

### Open Questions

* Should plugin `id` be globally unique or only within local environment?
* What minimum subset of Codex capabilities is required for a plugin to be considered “working” in metrics?
* How strict should heuristics be when inferring tool subsets (fail closed vs open)?

---

11. Task-Master Integration Notes

---

* **Capabilities → tasks**

  * Each “Capability” section becomes a top-level Task Master task (e.g., “Codex Plugin Specification Management”, “Platform Detection and Validation”).

* **Features → subtasks**

  * Each “Feature” becomes a subtask with fields:

    * `description`, `inputs`, `outputs`, `behavior`.
  * MVP flag should map to high-priority subtasks.

* **Dependencies → task deps**

  * Dependency chain section maps module dependencies to task dependencies.
  * Foundation modules (core/*) have no dependencies.
  * Higher-layer tasks depend on lower-layer tasks per the chain.

* **Phases → priorities**

  * Phase 0 tasks get highest priority.
  * Later phases follow sequentially; Phase 3 marks MVP completion.
  * Optional Phase 5 tasks flagged as non-MVP / lower priority.

This structure yields a dependency-aware task graph that can be executed topologically by Task Master.
