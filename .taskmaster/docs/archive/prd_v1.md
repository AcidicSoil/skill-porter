1. Overview

---

### Problem

`skill-porter` currently bridges Claude Code and Gemini CLI. Codex CLI has enough primitives (MCP config + AGENTS docs) to act as a third peer, but there is no standard, file-based “Codex plugin” unit that can be generated from existing Claude/Gemini skills and consumed by Codex.

Pain points:

* No canonical `codex/` plugin directory format.
* No Codex-aware platform detection/validation.
* No `Claude → Codex` or `Gemini → Codex` converter in `skill-porter`.
* No “universal” flow that guarantees Claude + Gemini + Codex artifacts from a single skill definition.

### Who

* Skill authors who already ship Claude or Gemini skills and want Codex support without learning Codex internals.
* Tooling engineers who integrate skills across CLIs and need consistent on-disk formats and validation.
* Codex users who just want to run a command and paste a config snippet to get a working plugin.

### Why current solutions fail

* Every author hand-rolls Codex config and AGENTS docs, duplicating effort and making error-prone edits.
* There is no reusable spec for a Codex plugin that mirrors Claude Skills/plugins.
* Platform detection/validation doesn’t know about Codex, so CI cannot enforce correctness.
* No end-to-end “universal” path: authors cannot declare a single skill and get all three CLIs wired.

### Success metrics

* ≥ 80% of Claude/Gemini skills in a representative sample convert to Codex with zero manual edits.
* ≥ 90% of converted Codex plugins pass automated validation and a minimal smoke test (Codex + MCP).
* `skill-porter universal` yields valid Claude + Gemini + Codex artifacts in ≥ 90% of tested repos.
* No regressions in existing Claude/Gemini flows (all current tests still pass).

Constraints and assumptions:

* Codex integration is config + docs only (TOML + Markdown); no changes to Codex internals.
* All Codex-specific behavior is isolated behind a Codex Plugin spec module.
* Reverse converters (Codex→Claude/Gemini) and a `codex-plugins` helper CLI are explicitly non-MVP.

---

2. Capability Tree (Functional Decomposition)

---

Each feature lists Description, Inputs, Outputs, Behavior. `[MVP]` marks minimum end-to-end path.

### Capability A: Codex Plugin Spec & Layout

#### A1. Codex Plugin Directory Layout [MVP]

* **Description**
  Canonical on-disk layout for Codex plugins under `codex/` in a repo.

* **Inputs**

  * Repo root path.
  * Existing Claude/Gemini artifacts (optional context).

* **Outputs**

  * `CodexPluginLayout` with resolved paths for:

    * `codex/codex-plugin.toml`
    * `codex/CODEX_AGENTS.md`
    * `codex/codex-config.snippet.toml`
    * `codex/docs/CODEX_ARCHITECTURE.md`

* **Behavior**

  * Default plugin root to `<repoRoot>/codex/`.
  * Ensure directories exist (or can be created) for all required files.
  * Provide deterministic rules if multiple candidates exist (e.g., conflicting `codex/` dirs).
  * Always use repo-relative paths; never emit absolute paths in spec or snippet.

#### A2. Codex Plugin Manifest Schema (`codex-plugin.toml`) [MVP]

* **Description**
  Typed manifest schema + validation for Codex plugin metadata and capabilities.

* **Inputs**

  * Raw TOML string for `codex-plugin.toml`.
  * Optional inferred defaults (repo URL, license, etc.).

* **Outputs**

  * `CodexPluginManifest` object:

    * `plugin.id`, `display_name`, `description`, `version`
    * `homepage`, `source_repo`, `license`
    * `agents_file`, `config_snippet_file`
    * `capabilities` (`uses_mcp`, `reads_files`, `writes_files`, `runs_shell`).
  * Validation result with errors/warnings.

* **Behavior**

  * Parse TOML; validate required fields and formats (e.g. slug for `plugin.id`).
  * Apply safe defaults (e.g. infer `source_repo` from git remote if available).
  * Return machine-readable diagnostics; never throw uncaught parse errors.

#### A3. `CODEX_AGENTS.md` Convention [MVP]

* **Description**
  Structural rules and generation helpers for Codex agent instructions.

* **Inputs**

  * Existing docs (e.g. `SKILL.md`, `GEMINI.md`) parsed to Markdown AST.
  * Plugin metadata (name, description).
  * MCP server list.

* **Outputs**

  * Markdown text for `CODEX_AGENTS.md` with:

    * H1 title = plugin name.
    * Summary “Use this when…” paragraph.
    * Operational guidance sections.
    * Explicit “MCP servers” section.

* **Behavior**

  * Strip platform-specific sections (Claude/Gemini CLI install flows).
  * Normalize headings and section order.
  * Ensure MCP server section matches `codex-config.snippet.toml`.

#### A4. Codex Config Snippet (`codex-config.snippet.toml`) [MVP]

* **Description**
  Standard structure for Codex MCP config snippet emitted by `skill-porter`.

* **Inputs**

  * MCP server definitions (names, connection mode, env vars).
  * Tool-level configuration (enabled/disabled tools where known).

* **Outputs**

  * TOML snippet with:

    * Comment header instructing user to append to `~/.codex/config.toml`.
    * `[mcp_servers.<name>]` blocks for each server.
    * Optional tool filters.

* **Behavior**

  * Map Claude/Gemini MCP definitions to Codex MCP syntax (STDIO vs HTTP).
  * Require at least one `[mcp_servers.*]` if `uses_mcp = true`.
  * Preserve env var placeholders; do not inline secrets.

---

### Capability B: Codex Platform Detection & Validation

#### B1. Codex Platform Detection [MVP]

* **Description**
  Extend platform detection to recognize Codex artifacts and classify multi-platform repos.

* **Inputs**

  * Repo root path.
  * Directory/file listing.

* **Outputs**

  * `PlatformMetadata` including:

    * `platform` ∈ {`claude`, `gemini`, `codex`, `universal`, `unknown`}.
    * `files.codex` (found Codex files).
    * `metadata.codex` (manifest summary).

* **Behavior**

  * Look for `codex-plugin.toml`, `CODEX_AGENTS.md` / `AGENTS.md`, `codex-config.snippet.toml`.
  * Parse `codex-plugin.toml` if present to extract summary.
  * Set `platform = universal` if ≥ 2 platforms are present, including Codex.

#### B2. Codex Validation [MVP]

* **Description**
  Codex-specific validation pipeline integrated into the global validator.

* **Inputs**

  * Repo root path.
  * `CodexMetadata` (layout + parsed manifest).

* **Outputs**

  * `ValidationResult` for Codex:

    * Blocking errors (e.g., invalid manifest).
    * Warnings (e.g., no MCP servers while `uses_mcp = true`).

* **Behavior**

  * Validate manifest structure, required fields, and file references.
  * Parse config snippet and ensure MCP sections align with manifest capabilities.
  * Attach results to global validation output; never crash when Codex files are malformed.

#### B3. Universal Platform Validation [MVP]

* **Description**
  Validate Codex artifacts whenever present, alongside Claude/Gemini.

* **Inputs**

  * Combined metadata for all platforms.

* **Outputs**

  * Aggregated validation summary across platforms.

* **Behavior**

  * Always run Codex validation when `files.codex` is non-empty.
  * Cross-check MCP server names and URLs/commands across platforms.
  * Flag obvious inconsistencies but do not block if platform semantics legitimately diverge.

---

### Capability C: Forward Converters (Claude/Gemini → Codex)

#### C1. Claude → Codex Converter [MVP]

* **Description**
  Generate Codex plugin artifacts from a Claude skill.

* **Inputs**

  * `SKILL.md` YAML frontmatter and body.
  * Existing Claude MCP metadata (servers, commands, allowed tools).
  * Target plugin layout from Capability A.

* **Outputs**

  * `codex/codex-plugin.toml`.
  * `codex/CODEX_AGENTS.md`.
  * `codex/codex-config.snippet.toml`.
  * `codex/docs/CODEX_ARCHITECTURE.md`.
  * Conversion summary (paths, warnings).

* **Behavior**

  * Map frontmatter `name` → `plugin.id` (slug) and `display_name`.
  * Map `description` to manifest + agents summary.
  * Infer `capabilities` from allowed tools and MCP servers.
  * Generate agent docs and architecture doc via shared doc generator.
  * Generate config snippet that passes Codex validation.

#### C2. Gemini → Codex Converter [MVP]

* **Description**
  Generate Codex plugin artifacts from a Gemini extension.

* **Inputs**

  * `gemini-extension.json` (name, description, version, mcpServers, excludeTools, settings).
  * `GEMINI.md`.
  * Target plugin layout.

* **Outputs**

  * Same four Codex files as C1.
  * Conversion summary.

* **Behavior**

  * Map extension metadata into Codex manifest fields.
  * Use `mcpServers` to generate MCP config blocks.
  * Derive `enabled_tools`/`disabled_tools` from `excludeTools` and known tool list where possible.
  * Strip Gemini-specific prose from docs, leaving Codex-relevant guidance.

---

### Capability D: Reverse Converters (Codex → Claude/Gemini) [Non-MVP]

#### D1. Codex → Claude Converter

* **Description**
  Reconstruct Claude skill artifacts from a Codex plugin.

* **Inputs**

  * `codex-plugin.toml`.
  * `CODEX_AGENTS.md`.
  * `codex-config.snippet.toml`.

* **Outputs**

  * `SKILL.md`.
  * `.claude-plugin/marketplace.json`.
  * `.claude/commands/*` and MCP config.

* **Behavior**

  * Reverse manifest mapping, preserving as much metadata as possible.
  * Derive `allowed-tools` from MCP config and tool filters.
  * Emit best-effort docs; mark uncertain decisions with comments.

#### D2. Codex → Gemini Converter

* **Description**
  Reconstruct Gemini extension artifacts from a Codex plugin.

* **Inputs**

  * Same as D1.

* **Outputs**

  * `gemini-extension.json`.
  * `GEMINI.md`.
  * Command/manifest files as needed.

* **Behavior**

  * Map plugin metadata into Gemini manifest fields.
  * Map MCP servers to `mcpServers`.
  * Rebuild usage docs from `CODEX_AGENTS.md`.

---

### Capability E: CLI & Universal Flow

#### E1. `convert --to codex` CLI Target [MVP]

* **Description**
  Codex as first-class target in `skill-porter convert`.

* **Inputs**

  * CLI invocation: `skill-porter convert --to codex [--output …]`.
  * Detected source platform (Claude or Gemini).

* **Outputs**

  * Generated Codex plugin on disk.
  * Exit code and CLI logs.

* **Behavior**

  * Route to C1 or C2 based on detection.
  * Fail fast with explicit error when source is unsupported.
  * Preserve existing behavior for other targets.

#### E2. Codex Post-Conversion Guidance [MVP]

* **Description**
  Deterministic “next steps” snippet for Codex usage.

* **Inputs**

  * Plugin root path.
  * Path to `codex-config.snippet.toml`.

* **Outputs**

  * CLI instructions, e.g.:

    * `cat codex/codex-config.snippet.toml >> ~/.codex/config.toml`
    * Instructions for wiring `CODEX_AGENTS.md` via Codex config.

* **Behavior**

  * Print copy/paste-ready commands.
  * Do not hide manual steps; keep instructions minimal and explicit.

#### E3. Tri-Platform `universal` Command [MVP]

* **Description**
  Ensure repos can be made Claude + Gemini + Codex-capable in one command.

* **Inputs**

  * Repo root.
  * Existing detection + validation.

* **Outputs**

  * Missing artifacts generated for all three platforms.
  * Final summary with per-platform status.

* **Behavior**

  * Use detection results to decide which conversions are needed.
  * Run validations for all platforms post-conversion.
  * Fail if any platform’s validation fails; print concise reasons.

---

### Capability F: Docs & Examples

#### F1. Codex Docs Generator [MVP]

* **Description**
  Shared generator for `CODEX_AGENTS.md` and `CODEX_ARCHITECTURE.md`.

* **Inputs**

  * Plugin manifest + capabilities.
  * Source docs AST (Claude/Gemini).
  * MCP server list.

* **Outputs**

  * Two Markdown strings for agents and architecture docs.

* **Behavior**

  * Encapsulate all Codex-specific doc formatting.
  * Guarantee presence of required headings and sections.

#### F2. README and Reference Docs

* **Description**
  Public docs describing Codex support in `skill-porter`.

* **Inputs**

  * Finalized spec and CLI behavior.

* **Outputs**

  * README section “Codex CLI Support”.
  * `docs/codex-support.md` (manifest schema, config details, troubleshooting).

* **Behavior**

  * Document only what is implemented; no speculative sections.
  * Include one complete example snippet for manifest and config.

#### F3. `examples/codex-basic/` Repo

* **Description**
  Minimal end-to-end example repository used in docs and tests.

* **Inputs**

  * Sample skill definition.

* **Outputs**

  * `examples/codex-basic/` with Claude + Gemini + Codex artifacts and a simple MCP server.

* **Behavior**

  * Must be runnable in CI for E2E tests.
  * Demonstrate the full “universal” pipeline.

---

3. Repository Structure + Module Definitions

---

New/extended modules; existing names are indicative.

```text
src/
  core/
    fs.ts
    toml.ts
    logging.ts
  spec/
    codexPluginSpec.ts
  analyzers/
    platformDetector.ts
    codexMetadata.ts
  validation/
    codexValidator.ts
  converters/
    claudeToCodexConverter.ts
    geminiToCodexConverter.ts
    codexToClaudeConverter.ts      # optional
    codexToGeminiConverter.ts      # optional
  docs/
    codexDocsGenerator.ts
  cli/
    convertCommand.ts
    universalCommand.ts
docs/
  codex-support.md
examples/
  codex-basic/
```

For each module:

#### core/fs.ts

* **Responsibility**
  Thin abstraction over filesystem I/O for reading/writing files and ensuring directories.

* **Exports**

  * `readFile(path): Promise<string>`
  * `writeFile(path, contents): Promise<void>`
  * `ensureDir(path): Promise<void>`
  * `pathExists(path): Promise<boolean>`

#### core/toml.ts

* **Responsibility**
  Parse and serialize TOML with uniform error handling.

* **Exports**

  * `parseToml<T>(text: string): T`
  * `stringifyToml(obj: unknown): string`
  * `TomlError`

#### core/logging.ts

* **Responsibility**
  Structured logging utilities.

* **Exports**

  * `logInfo(msg, meta?)`
  * `logWarn(msg, meta?)`
  * `logError(msg, meta?)`

#### spec/codexPluginSpec.ts

* **Responsibility**
  Single source of truth for Codex plugin layout and manifest schema.

* **Exports**

  * Types: `CodexPluginLayout`, `CodexPluginManifest`, `CodexCapabilities`.
  * `defaultCodexLayout(repoRoot): CodexPluginLayout`.
  * `parseCodexManifest(text): CodexPluginManifest`.
  * `validateCodexManifest(manifest): ValidationResult`.
  * `inferCapabilitiesFromSources(sourceMeta): CodexCapabilities`.

#### analyzers/codexMetadata.ts

* **Responsibility**
  Detect Codex-specific files and parse minimal metadata.

* **Exports**

  * `detectCodexFiles(repoRoot): Promise<CodexMetadata>`.
  * `summarizeCodexPlugin(meta: CodexMetadata): CodexSummary`.

#### analyzers/platformDetector.ts (extended)

* **Responsibility**
  Multi-platform detection (Claude, Gemini, Codex, universal).

* **Exports**

  * `detectPlatform(repoRoot): Promise<PlatformMetadata>`.

#### validation/codexValidator.ts

* **Responsibility**
  Apply Codex-specific validation rules and integrate them into global validation.

* **Exports**

  * `validateCodex(repoRoot, meta: CodexMetadata): Promise<ValidationResult>`.

#### docs/codexDocsGenerator.ts

* **Responsibility**
  Generate Codex docs (`CODEX_AGENTS.md`, `CODEX_ARCHITECTURE.md`).

* **Exports**

  * `generateCodexAgentsDoc(ctx: AgentsDocContext): string`.
  * `generateCodexArchitectureDoc(ctx: ArchitectureDocContext): string`.

#### converters/claudeToCodexConverter.ts

* **Responsibility**
  Convert a Claude skill into Codex plugin artifacts.

* **Exports**

  * `convertClaudeToCodex(options: ConvertClaudeToCodexOptions): Promise<ConversionResult>`.

#### converters/geminiToCodexConverter.ts

* **Responsibility**
  Convert a Gemini extension into Codex plugin artifacts.

* **Exports**

  * `convertGeminiToCodex(options: ConvertGeminiToCodexOptions): Promise<ConversionResult>`.

#### converters/codexToClaudeConverter.ts (optional)

* **Responsibility**
  Reverse conversion from Codex plugin to Claude artifacts.

* **Exports**

  * `convertCodexToClaude(options: ConvertCodexToClaudeOptions): Promise<ConversionResult>`.

#### converters/codexToGeminiConverter.ts (optional)

* **Responsibility**
  Reverse conversion from Codex plugin to Gemini artifacts.

* **Exports**

  * `convertCodexToGemini(options: ConvertCodexToGeminiOptions): Promise<ConversionResult>`.

#### cli/convertCommand.ts (extended)

* **Responsibility**
  CLI wiring for `convert`, including `--to codex`.

* **Exports**

  * `registerConvertCommand(program): void`.

#### cli/universalCommand.ts (extended)

* **Responsibility**
  CLI wiring for `universal`, ensuring all three platforms are present and valid.

* **Exports**

  * `registerUniversalCommand(program): void`.

---

4. Dependency Chain

---

Layered foundation → spec → analysis/validation → conversion/docs → CLI → examples.

### Layer 0: Foundation

* `core/fs`
* `core/toml`
* `core/logging`

Depends on: none.

### Layer 1: Spec

* `spec/codexPluginSpec`

Depends on: `core/toml`, `core/logging` (for error shaping), optionally `core/fs` for layout defaults.

### Layer 2: Analysis / Metadata

* `analyzers/codexMetadata`

  * Depends on: `core/fs`, `core/toml`, `spec/codexPluginSpec`.
* `analyzers/platformDetector`

  * Depends on: `core/fs`, `spec/codexPluginSpec`, `analyzers/codexMetadata`.

### Layer 3: Validation

* `validation/codexValidator`

  * Depends on: `core/fs`, `core/toml`, `spec/codexPluginSpec`, `analyzers/codexMetadata`.

### Layer 4: Docs & Conversion

* `docs/codexDocsGenerator`

  * Depends on: `spec/codexPluginSpec`.

* `converters/claudeToCodexConverter`

  * Depends on: `core/fs`, `core/toml`, `spec/codexPluginSpec`, `analyzers/codexMetadata`, `docs/codexDocsGenerator`.

* `converters/geminiToCodexConverter`

  * Same dependencies as Claude→Codex.

* `converters/codexToClaudeConverter` (optional)

  * Depends on: `core/fs`, `core/toml`, `spec/codexPluginSpec`.

* `converters/codexToGeminiConverter` (optional)

  * Same dependencies as Codex→Claude.

No cycles: docs generator does not depend on converters.

### Layer 5: CLI

* `cli/convertCommand`

  * Depends on: `converters/*`, `analyzers/platformDetector`, `validation/codexValidator`, `core/logging`.

* `cli/universalCommand`

  * Depends on: `cli/convertCommand`, `analyzers/platformDetector`, `validation/codexValidator`.

### Layer 6: Docs & Examples

* `docs/codex-support.md`

  * Depends on: implementation behavior (no code dependency).

* `examples/codex-basic/`

  * Depends on: working CLI commands; treated as consumer of the above layers.

---

5. Development Phases

---

Phases follow dependency order; MVP completes at Phase 3.

### Phase 0 – Foundation

**Scope**

* Implement or confirm `core/fs`, `core/toml`, `core/logging`.

**Dependencies**

* None.

**Tasks**

1. `core/toml`

   * Implement parse/serialize + `TomlError`.
   * Acceptance: invalid TOML yields structured errors; round-trip tests for simple objects.
   * Tests: unit tests for success/error paths.

2. `core/fs` and `core/logging`

   * Standardize read/write, path existence, directory creation, logging.
   * Acceptance: later modules use these instead of raw `fs`/`console`.
   * Tests: filesystem tests on temp dirs; log output sanity checks.

**Exit criteria**

* All higher layers can rely on these shared utilities.

---

### Phase 1 – Codex Spec & Detection

**Scope**

* Codex spec module and Codex-aware platform detection.

**Dependencies**

* Phase 0.

**Tasks**

1. `spec/codexPluginSpec`

   * Implement manifest types, validation rules, default layout.
   * Acceptance: fixture manifests validate as expected; layout resolves `codex/` paths.
   * Tests: table-driven manifest validation; layout tests.

2. `analyzers/codexMetadata`

   * Implement file discovery + manifest parsing.
   * Acceptance: fixture repos with Codex files produce correct metadata.
   * Tests: repo fixtures with various Codex file combinations.

3. Extend `analyzers/platformDetector`

   * Add `PLATFORM_TYPES.CODEX` and `UNIVERSAL` behaviors per draft.
   * Acceptance: classification matches expectations for single-/multi-platform fixtures.
   * Tests: detection tests across mixed platform repo snapshots.

**Exit criteria**

* `analyze` can see Codex artifacts and classify repos.

---

### Phase 2 – Codex Validation

**Scope**

* Codex validator integrated into global `validate`.

**Dependencies**

* Phase 1.

**Tasks**

1. `validation/codexValidator`

   * Implement manifest + snippet + layout checks.
   * Acceptance: invalid manifests/snippets report precise errors; missing MCP servers flagged where required.
   * Tests: fixtures for each failure mode; full-success fixture.

2. Wire into global `validate` command

   * Use platform metadata and `files.codex` to decide when to run Codex validation.
   * Acceptance: `validate` outputs a Codex section when relevant; no changes when Codex absent.
   * Tests: regression tests for existing platforms; new tests for Codex-only and universal repos.

**Exit criteria**

* Any Codex-related issues appear in `validate` in a structured, parseable way.

---

### Phase 3 – Forward Converters + CLI (MVP)

**Scope**

* Claude/Gemini → Codex, shared doc generator, CLI support for `--to codex` and universal.

**Dependencies**

* Phases 1–2.

**Tasks**

1. `docs/codexDocsGenerator`

   * Implement generators for agents + architecture docs.
   * Acceptance: generated docs contain required sections and are stable under snapshot tests.
   * Tests: golden markdown snapshots for multiple contexts.

2. `converters/claudeToCodexConverter`

   * Implement mapping from SKILL + MCP metadata to Codex plugin.
   * Acceptance: fixtures convert successfully and pass Codex validation.
   * Tests: golden file comparison for generated `codex/` contents; error-case fixtures.

3. `converters/geminiToCodexConverter`

   * Implement mapping from Gemini extension to Codex plugin.
   * Acceptance: same as Claude path, with excludeTools→tool filters handling.
   * Tests: golden outputs across several manifest shapes.

4. CLI `convert --to codex`

   * Extend `convertCommand` to accept `codex` and route to correct converter.
   * Acceptance: CLI exit codes and logs correct; unsupported source types fail cleanly.
   * Tests: CLI integration tests using temp fixture repos.

5. Codex post-conversion guidance

   * Implement Codex-specific “next steps” printing.
   * Acceptance: output contains minimal, copy/paste-ready instructions for config + docs usage.
   * Tests: snapshot tests for CLI output.

6. Update `universal` command

   * Extend to require Codex artifacts and validation.
   * Acceptance: `skill-porter universal` generates missing platforms and validates all three; fails if any validation fails.
   * Tests: fixture repos representing each starting state (Claude-only, Gemini-only, Codex-only, mixed).

**Exit criteria**

* From a valid Claude or Gemini skill, `convert --to codex` + following printed steps yield a working Codex setup.
* `universal` can promote single-platform repos to three-way.

---

### Phase 4 – Docs & Example

**Scope**

* User-facing documentation and a minimal runnable example.

**Dependencies**

* Phase 3.

**Tasks**

1. Docs (`codex-support.md`, README section)

   * Acceptance: docs reference actual CLI flags and file names; contain one end-to-end Codex example.
   * Tests: link checker; manual review.

2. `examples/codex-basic/`

   * Create minimal skill with one MCP server and build all three platform artifacts via CLI.
   * Acceptance: CI job runs analyze → universal → validate successfully.
   * Tests: CI E2E test using this example.

**Exit criteria**

* A new user can read docs + run the example and replicate the Codex flow.

---

### Phase 5 – Reverse Converters (Optional)

**Scope**

* Codex→Claude/Gemini converters.

**Dependencies**

* Phases 1–3.

**Tasks**

1. Codex→Claude converter

   * Acceptance: for chosen fixture skills, Claude→Codex→Claude round-trip yields artifacts that pass Claude validation and are functionally equivalent.
   * Tests: round-trip golden tests.

2. Codex→Gemini converter

   * Same pattern as above.

**Exit criteria**

* Round-trip stories work for selected fixtures; these features remain non-MVP and can be cut if needed.

---

6. User Experience

---

### Personas

* Skill authors: operate from terminal, comfortable with config files.
* Tooling engineers: need deterministic formats and machine-readable validation.
* Codex users: want simple instructions to wire skills into Codex.

### Core flows

1. **Inspect repo platforms**

   * Command: `skill-porter analyze`.
   * Output: platform classification including Codex summary (id, version, files).

2. **Convert Claude skill → Codex plugin**

   * Command: `skill-porter convert --to codex`.
   * Output: `codex/` directory + short “Next steps for Codex” block.

3. **Convert Gemini extension → Codex plugin**

   * Same command from a Gemini-first repo.

4. **Make repo universal**

   * Command: `skill-porter universal`.
   * Output: All three platforms present and validated; any failure surfaces causes.

### UX constraints

* CLI logs must be stable enough for snapshot tests.
* Error messages must name the exact file and field causing issues.
* Copy/paste instructions must not include placeholders that silently fail.

---

7. Technical Architecture

---

### Components

* Codex Plugin Spec: `spec/codexPluginSpec.ts`.
* Detection + Metadata: `analyzers/platformDetector.ts`, `analyzers/codexMetadata.ts`.
* Validator: `validation/codexValidator.ts`.
* Converters: `converters/*`.
* Docs generator: `docs/codexDocsGenerator.ts`.
* CLI entrypoints: `cli/convertCommand.ts`, `cli/universalCommand.ts`.
* Example + Docs: `examples/codex-basic/`, `docs/codex-support.md`.

### Data models

* `CodexPluginManifest`
* `CodexCapabilities`
* `CodexPluginLayout`
* `CodexMetadata` (layout + manifest + derived state)
* `ConversionResult` (generated files + warnings)
* `ValidationResult` (errors, warnings, severity)

### Integration

* Local filesystem only; no network calls to Codex.
* Output is TOML + Markdown, consumed by Codex as documented in its config spec.
* Existing JSON/CLI schemas for `analyze`/`validate` extended with Codex sections.

---

8. Test Strategy

---

### Targets

* Unit: ≥ 60% of tests, ≥ 85% line/statement coverage.
* Integration: converters + validators + CLI commands.
* E2E: example repo pipeline.

### Key scenarios

* Spec and validation:

  * Valid manifest + snippet pass.
  * Missing required fields, invalid slugs, broken TOML yield precise errors.

* Detection:

  * Single-platform and universal repos classified correctly.
  * Codex metadata summaries stable across runs.

* Conversion:

  * Claude/Gemini fixtures convert to Codex and pass validation.
  * Edge cases: minimal MCP, complex tool lists, absent optional fields.
  * Failure: missing source docs, malformed source manifests.

* CLI:

  * `convert --to codex` success + failure cases (unsupported platform, invalid source).
  * `universal` from each starting platform state.
  * Snapshot testing of “Next steps” blocks.

* E2E:

  * Example repo: analyze → universal → validate for all platforms.

---

9. Risks and Mitigations

---

1. **Codex config semantics change**

   * Impact: Medium; generated plugins may become outdated.
   * Mitigation: centralize assumptions in `codexPluginSpec`; version spec if needed.

2. **Heuristic capability/tool mapping is imperfect**

   * Impact: Medium; incorrect tool exposure or capability flags.
   * Mitigation: prefer conservative defaults and warnings; document known limitations.

3. **Complex source skills break converters**

   * Impact: Medium–High on conversion success.
   * Mitigation: expand fixture set; surface actionable diagnostics with clear file/field references.

4. **Scope creep from optional reverse converters**

   * Impact: High on delivery; low on MVP value.
   * Mitigation: keep all D* features in Phase 5; easy to cut without affecting MVP.

5. **Docs lag implementation**

   * Impact: Medium; confusion for early adopters.
   * Mitigation: treat README + example as hard exit criteria for Phase 4.

---

10. Appendix

---

* Source draft outcome and system scan describing Codex capabilities and missing plugin unit.
* hi-ai–generated roadmap describing targets, non-goals, and Codex file conventions.
* Links (in original draft) to Codex config docs and Claude skills docs, used as reference for spec design.

---

11. Task-Master Integration Notes

---

* **Capabilities → tasks**

  * A–F map to top-level tasks.

* **Features → subtasks**

  * Each feature (A1–F3, D1–D2) is an individually testable subtask with defined Inputs/Outputs/Behavior.

* **Dependencies → task deps**

  * Use Section 4 dependency chain directly; no cycles.
  * Foundation (core/*) has zero deps; CLI tasks depend on converters + validators.

* **Phases → priorities**

  * Phase 0–3 tasks flagged as MVP-critical; Phase 4 as onboarding; Phase 5 as optional/low priority.
