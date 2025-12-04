You are an expert Node.js / TypeScript/JavaScript CLI engineer working in the `jduncan-rva/skill-porter` repository. Your task is to extend the existing Skill Porter tool to support a third platform, `codex`, targeting the OpenAI Codex CLI ecosystem and its MCP-based tool configuration, in a way that mirrors the existing Claude Code and Gemini CLI support.

The end state: users can do for Codex CLI what they can already do for Claude Code and Gemini CLI:

* Convert Claude Code skills and Gemini extensions into Codex-compatible plugin bundles.
* Optionally convert from Codex bundles back into Claude/Gemini formats.
* Analyze, validate, and make “universal” multi-platform bundles including Codex.
* Achieve a Codex plugin UX that is structurally similar to the Claude plugin system (`SKILL.md`, `.claude/commands`, `.claude-plugin`, `mcp-servers.json`), but implemented around Codex’s `config.toml` and MCP server configuration model.

Use the existing architecture as the baseline:

* CLI: `src/cli.js` (Commander-based CLI with `convert`, `analyze`, `validate`, `universal`, `create-pr`, `fork` commands).
* Core orchestrator: `src/index.js` (`SkillPorter` class with `analyze`, `convert`, `validate`, `makeUniversal`).
* Platform detection: `src/analyzers/detector.js` (`PlatformDetector`, `PLATFORM_TYPES`).
* Validation layer: `src/analyzers/validator.js` (`Validator` with separate Claude and Gemini validators).
* Converters: `src/converters/claude-to-gemini.js`, `src/converters/gemini-to-claude.js`.

The hi-ai MCP server and its development workflow are context only; do not modify anything related to hi-ai.

---

## 1. High-level goals

1. Add first-class Codex platform support to Skill Porter:

   * New `codex` platform identifier.
   * Detection of Codex plugin/config bundles.
   * Validation of Codex-specific manifests and config fragments.
   * Conversion pipelines between Claude ↔ Codex and Gemini ↔ Codex, building on the existing Claude ↔ Gemini patterns.

2. Define a Codex “plugin bundle” format that:

   * Is self-contained in a directory, like Claude plugins.
   * Encapsulates Codex `config.toml`-compatible MCP server definitions and related settings.
   * Provides human-readable documentation for the Codex user, analogous to `SKILL.md` and `GEMINI.md`.

3. Extend the CLI UX:

   * `skill-porter convert <source-path> --to codex` produces a Codex bundle.
   * `skill-porter analyze` recognizes Codex bundles.
   * `skill-porter validate` can validate Codex bundles.
   * Optionally extend `universal` to support Codex as a third platform in a controlled way.

4. Maintain the existing code style and patterns:

   * Follow the shape and behavior of `ClaudeToGeminiConverter` and `GeminiToClaudeConverter`.
   * Reuse analyzer/validator patterns.
   * Preserve current error handling and logging conventions.

5. Ship with tests and minimal docs updates so that the new behavior is verifiable and discoverable.

---

## 2. Codex plugin/bundle format design

Design a Codex bundle layout that is analogous in spirit to Claude plugins but aligned with Codex’s `config.toml` and MCP server concepts:

### 2.1 Directory layout

Define a Codex bundle root with this structure:

* `codex-plugin.toml`

  * Primary manifest for the bundle.
* `CODEX.md`

  * Human-readable documentation / context for Codex users (analogous to `GEMINI.md` and `SKILL.md` content).
* `config-snippet.toml`

  * A TOML fragment that can be appended/merged into `~/.codex/config.toml` to enable MCP servers and settings.
* `commands/` (optional)

  * TOML or Markdown for any Codex-specific CLI guidance or examples (low priority; only implement if trivial).
* `shared/` and `docs/`

  * Reuse the same pattern as existing converters for shared docs and architecture references where applicable.

You are free to adjust subpaths slightly if you can keep them consistent with current `shared/` and `docs/` conventions.

### 2.2 `codex-plugin.toml` schema

Design a minimal but useful schema:

```toml
# codex-plugin.toml
name = "my-codex-plugin"
description = "Short human-readable description of what this plugin does."
version = "1.0.0"

# Optional tags/metadata
category = "general"
author = "Converted from Claude/Gemini via skill-porter"
homepage = "https://example.com"        # optional
repository = "https://github.com/..."   # optional

# MCP server definitions (Codex-compatible configuration)
# These should be structurally compatible with what Codex expects inside config.toml.
[mcp_servers.github]
transport = "stdio"                     # or "sse"/"http" as appropriate
command = "node"
args = ["mcp-server/index.js"]
enabled = true
startup_timeout_sec = 30
tool_timeout_sec = 60

[mcp_servers.github.env]
GITHUB_TOKEN = "${GITHUB_TOKEN}"        # for later substitution

# Optional plugin-level settings mirrored from Gemini or Claude env usage
[settings]
# Key/value metadata to help users know what env vars or settings they must supply.
```

Constraints:

* Keep the `mcp_servers.*` tables compatible with Codex’s MCP config concepts (transport, command/args/env/timeouts, enabled/disabled tools).
* Do not assume Codex supports “plugins” natively; this manifest is for Skill Porter and humans, but should map 1:1 to what would be placed inside `config.toml`.

### 2.3 `config-snippet.toml`

Design `config-snippet.toml` as a ready-to-paste TOML fragment:

* Contains only the relevant `[mcp_servers.*]` and tool-related config needed to enable this plugin’s MCP servers.
* Mirrors exactly the structure expected under `[mcp_servers]` in Codex’s `~/.codex/config.toml`.
* Optionally includes comments with installation instructions (e.g., “append these tables under `[mcp_servers]` in your config”).

For example:

```toml
# config-snippet.toml
[mcp_servers.github]
transport = "stdio"
command = "node"
args = ["${workspace}/mcp-server/index.js"]
enabled = true

[mcp_servers.github.env]
GITHUB_TOKEN = { from_env = "GITHUB_TOKEN" }

[mcp_servers.github.tools]
enabled_tools = ["issues", "pull_requests"]
disabled_tools = []
```

You must not hard-code the exact Codex schema; instead, ensure the converter logic is explicit about mapping fields from Claude/Gemini into the tables that Codex’s documentation describes.

---

## 3. Platform enumeration and detection changes

### 3.1 Extend `PLATFORM_TYPES`

In `src/analyzers/detector.js`:

* Add `CODEX: 'codex'` to `PLATFORM_TYPES`.
* Ensure any consumers of `PLATFORM_TYPES` (especially in `src/index.js`, `validator.js`, and CLI commands) can handle the new value.

### 3.2 Detection logic

In `PlatformDetector.detect`:

* Track Codex-specific files separately, similar to `claude` and `gemini`:

  * Add a `codex` array under `files`:

    ```js
    files: {
      claude: [],
      gemini: [],
      codex: [],
      shared: []
    }
    ```

* Implement `_detectCodexFiles(dirPath)`:

  * If `codex-plugin.toml` exists and is valid TOML, record it as `{ file: 'codex-plugin.toml', type: 'manifest', valid: true/false, issue?: string }`.
  * If `config-snippet.toml` exists and is valid TOML, record it as `{ file: 'config-snippet.toml', type: 'config', valid: true/false }`.
  * Optionally treat a `CODEX.md` file as `type: 'context'`.

* Platform resolution rules:

  * `hasClaude`, `hasGemini`, `hasCodex` booleans.
  * If two or more are true → `platform = PLATFORM_TYPES.UNIVERSAL`, `confidence = 'high'`.
  * Else if only one is true → set platform to that one.
  * Else → `UNKNOWN` as before.

* Extend metadata extraction:

  * In `_extractMetadata`, when platform is CODEX or UNIVERSAL, read `codex-plugin.toml` (if present) and store parsed manifest under `metadata.codex`.

Ensure error messages and thrown errors stay consistent with existing patterns.

---

## 4. Validation logic for Codex

In `src/analyzers/validator.js`:

### 4.1 Accept Codex platform

* Extend `validate(dirPath, platform)` so that:

  * If `platform === PLATFORM_TYPES.CODEX || platform === PLATFORM_TYPES.UNIVERSAL`, call a new `_validateCodex(dirPath)`.

* Update any conditions that assume only `claude` and `gemini` exist.

### 4.2 Implement `_validateCodex(dirPath)`

Add a private method `_validateCodex(dirPath)` that:

* Confirms `codex-plugin.toml` exists; if missing, push an error: `"Missing required file: codex-plugin.toml"`.

* Parses `codex-plugin.toml` as TOML (introduce a TOML parser dependency if needed, or reuse existing YAML/JSON patterns by adding a small TOML helper module).

* Enforces required fields:

  * `name` must be non-empty.
  * `description` should exist and be non-empty (warn if < ~30 chars).
  * `version` should exist.

* Validates `[mcp_servers]` block:

  * There must be at least one `[mcp_servers.*]` entry or else warn that the plugin provides no MCP integration.
  * For each server:

    * Check `command` or `url` is present depending on the transport type (stdio vs http/sse), according to Codex docs.
    * If `env` exists, ensure it is an object (warn if empty).

* Validates `config-snippet.toml` if present:

  * Parse TOML; if invalid, push an error like `"Invalid TOML in config-snippet.toml: <message>"`.
  * Ensure at least one `[mcp_servers.*]` table is present; warn if not.

* Warn if `CODEX.md` is missing: `"Missing CODEX.md (recommended for providing Codex context)"`.

Return a validation result consistent with existing structure:

```js
{
  valid: errors.length === 0,
  errors: this.errors,
  warnings: this.warnings
}
```

---

## 5. Converter design

Introduce Codex converters by following the pattern of `ClaudeToGeminiConverter` and `GeminiToClaudeConverter`. Reuse their extraction/metadata strategies as much as possible.

### 5.1 Reusable intermediate model (optional but preferred)

To avoid N² converter explosion, introduce a lightweight internal “universal skill” model:

* Define a TypeScript-like interface in comments (implementation can remain plain JS):

  ```js
  /**
   * UniversalSkill (internal IR)
   * - name, description, version
   * - mcpServers: map of serverName -> normalized MCP config
   * - envSettings: inferred settings with default/required/secret flags
   * - toolRestrictions: generic model for allowed/excluded tools
   * - docs: { claudeBody?, geminiBody?, codexBody? }
   * - commands: normalized command definitions
   */
  ```

* Refactor existing converters in-place:

  * Extract Claude → UniversalSkill logic from `ClaudeToGeminiConverter`’s `_extractClaudeMetadata`.
  * Extract Gemini → UniversalSkill logic from `GeminiToClaudeConverter`’s `_extractGeminiMetadata`.
  * Implement `UniversalSkill → Gemini` and `UniversalSkill → Claude` generation functions used by both converters.

This refactor is not strictly required but will significantly simplify Codex support. If time is constrained, you may implement direct converters instead, but ensure you do not regress Claude/Gemini behavior.

### 5.2 Claude → Codex

Create `src/converters/claude-to-codex.js`:

* Constructor signature mirrors `ClaudeToGeminiConverter`:

  ```js
  export class ClaudeToCodexConverter {
    constructor(sourcePath, outputPath) { ... }
    async convert() { ... }
  }
  ```

* Implementation steps:

  1. Ensure `outputPath` directory exists.

  2. Reuse the same Claude metadata extraction as `ClaudeToGeminiConverter`:

     * Parse `SKILL.md` frontmatter for `name`, `description`, optional `subagents`, `allowed-tools`.
     * Parse `.claude-plugin/marketplace.json` for MCP servers and version info if present.

  3. Derive `UniversalSkill` (if you implemented the IR) or equivalent local metadata structure.

  4. Generate `codex-plugin.toml`:

     * Map `name`, `description`, `version`.
     * Map any MCP servers found in marketplace into `[mcp_servers.<name>]` tables:

       * Convert relative paths to something appropriate for Codex; avoid `${extensionPath}`, and instead use variables Codex expects (e.g. `${workspace}` or similar).
       * Convert environment variables to `${VAR}` or `{ from_env = "VAR" }` style depending on Codex convention.
     * Convert `allowed-tools` (Claude) into Codex’s `enabled_tools`/`disabled_tools` where possible.

  5. Generate `CODEX.md`:

     * Header: `# <name> – Codex Plugin`
     * Short description from SKILL frontmatter.
     * A “Quick Start” section explaining how to enable the plugin by merging `config-snippet.toml` into `~/.codex/config.toml`.
     * Include content from `SKILL.md` (without frontmatter) as additional documentation.
     * Append a short footer noting it was generated by skill-porter.

  6. Generate `config-snippet.toml` from the MCP and tool restriction data:

     * Include only `[mcp_servers.*]` tables needed to enable the plugin’s MCP servers.
     * Map any tool restrictions (allowed-tools) to `enabled_tools`/`disabled_tools` if the Codex schema supports it; otherwise add comments indicating manual review.

  7. Ensure `shared/` and `docs/` structure is consistent with existing converters (copy or adapt `_ensureSharedStructure`/`_injectDocs` patterns where useful).

* Return a result object with `success`, `files`, `warnings`, `errors`, and `metadata` fields shaped like existing converters.

### 5.3 Gemini → Codex

Create `src/converters/gemini-to-codex.js`:

* Constructor and `convert()` signature mirror `GeminiToClaudeConverter`.

* Implementation steps:

  1. Extract Gemini metadata (reuse `GeminiToClaudeConverter._extractGeminiMetadata`):

     * `gemini-extension.json` for `name`, `description`, `version`, `mcpServers`, `settings`, `excludeTools`.
     * Optional `GEMINI.md` or other context file.

  2. Normalize into `UniversalSkill` or equivalent.

  3. Generate `codex-plugin.toml`:

     * `name`, `description`, `version`.
     * Map `mcpServers` entries to `[mcp_servers.<name>]` tables:

       * `command`, `args`, `env` map directly to Codex stdio servers.
       * Ensure any `${extensionPath}` placeholders are converted into an appropriate runtime path for Codex (e.g., `${workspace}` or use relative paths with a doc note).
     * Map `excludeTools` to `enabled_tools`/`disabled_tools` (inverse of Claude’s allowed-tools mapping).

  4. Generate `CODEX.md`:

     * Similar structure as above; incorporate information from `GEMINI.md` as general usage documentation.
     * Document any settings that came from `manifest.settings` as environment variables expected by Codex.

  5. Generate `config-snippet.toml` from `mcpServers` and `settings`.

  6. Shared/docs structure similar to other converters.

### 5.4 Codex → Claude and Codex → Gemini (optional but ideal)

If time allows, implement reverse converters:

* `src/converters/codex-to-claude.js`
* `src/converters/codex-to-gemini.js`

They should:

* Parse `codex-plugin.toml` and `config-snippet.toml`.
* Normalize into `UniversalSkill`.
* Generate `SKILL.md`, `.claude-plugin/marketplace.json`, and any commands for Claude.
* Generate `gemini-extension.json`, `GEMINI.md`, and `commands/*.toml` for Gemini.

Keep reverse converters symmetric with forward ones to preserve round-trip fidelity where possible.

---

## 6. Integrate Codex into `SkillPorter` and CLI

### 6.1 `SkillPorter.convert`

In `src/index.js`:

* Extend target platform validation to accept `'codex'`.

* After detection:

  * If `detection.platform === PLATFORM_TYPES.CODEX` and `targetPlatform === PLATFORM_TYPES.CODEX`, return the “already a codex plugin” early exit message.

* Add Codex conversion branches:

  * When `targetPlatform === PLATFORM_TYPES.CODEX` and detection is CLAUDE or GEMINI, call the appropriate Codex converter:

    * Claude → Codex: use `ClaudeToCodexConverter`.
    * Gemini → Codex: use `GeminiToCodexConverter`.

  * When detection is CODEX and targetPlatform is CLAUDE or GEMINI, call Codex→Claude/Gemini converters (if implemented). Otherwise, throw a clear error stating that reverse conversion is not yet supported.

* Ensure `validate` step uses `Validator.validate(outputPath, targetPlatform)` with the new Codex branch.

### 6.2 `SkillPorter.makeUniversal`

Decide on universal semantics:

* Minimal version: treat “universal” as “supports at least two platforms” and leave Codex out of `makeUniversal` for now.
* Better version: support producing all three platforms when a `--include-codex` flag is set or when Codex is auto-detected.

At minimum:

* Do not break existing `universal` behavior (Claude+Gemini).
* If you extend it to include Codex, document the behavior and keep the log messaging accurate.

### 6.3 CLI changes (`src/cli.js`)

Update CLI commands:

* `convert`:

  * Update description and `--to` option help text to include `codex`.
  * In “Next steps” output:

    * For `codex`, print something like:

      ```bash
      # Example next steps after conversion:
      # 1. Inspect config-snippet.toml
      # 2. Merge it into your ~/.codex/config.toml under [mcp_servers]
      ```

* `analyze`:

  * Add printing of `Codex files found:` section similar to Claude/Gemini sections.
  * Include any detection metadata for Codex.

* `validate`:

  * The `--platform` option help text should mention `codex`.
  * Validation output should work identically for Codex bundles.

* `universal`, `create-pr`, `fork`:

  * Only extend these if you actually wire in Codex support.
  * Ensure any printed messaging about “both platforms” is updated to reflect the new behavior if Codex is included.

Keep the CLI’s control flow, error handling, and Chalk styling consistent with the existing conventions.

---

## 7. Error handling, robustness, and security

* Treat malformed TOML in Codex manifests/config-snippets as validation errors, not hard crashes.
* When mapping environment variables, avoid leaking actual secrets into generated files; maintain the `${VAR}` placeholder pattern or `from_env` style references.
* When path-transforming MCP commands/args, avoid making assumptions about user directory layout; prefer relative paths and document any required environment variables in `CODEX.md`.

---

## 8. Testing

Add tests for the new functionality. If the repo already has a test framework, extend it; otherwise, introduce a minimal Node.js test setup (e.g., `node:test`).

Priority tests:

1. Detection:

   * A fixture directory with only Codex files (`codex-plugin.toml`, `config-snippet.toml`) is detected as `platform = codex`.
   * Mixed-platform directories (e.g., Claude + Codex) are detected as `UNIVERSAL`.

2. Validation:

   * Valid Codex plugin passes `_validateCodex`.
   * Missing or invalid `codex-plugin.toml` causes errors.
   * Invalid TOML in `config-snippet.toml` is reported.

3. Conversion:

   * Claude → Codex: simple fixture with `SKILL.md` + marketplace produces a Codex bundle with correct `codex-plugin.toml`, `CODEX.md`, and `config-snippet.toml`.
   * Gemini → Codex: simple fixture with `gemini-extension.json` + `GEMINI.md` produces a Codex bundle with correct mapping of MCP server config.
   * If reverse converters are implemented, add round-trip tests where feasible.

4. CLI smoke tests (if feasible):

   * Programmatic invocation of `SkillPorter.convert` with `targetPlatform = 'codex'` from a test harness.

---

## 9. Documentation updates

* Update any README or usage docs in the repo to:

  * Mention the new `codex` platform.

  * Show example commands:

    ```bash
    skill-porter convert path/to/claude-skill --to codex --output path/to/codex-plugin
    skill-porter validate path/to/codex-plugin --platform codex
    ```

  * Briefly describe the Codex bundle layout and how to merge `config-snippet.toml` into `~/.codex/config.toml`.

* Ensure log messages and CLI descriptions accurately reflect that Skill Porter now supports Claude, Gemini, and Codex.

Implement the above changes incrementally but coherently, preserving existing behavior for Claude and Gemini while adding Codex as a first-class, well-validated target and, where supported, source platform.
