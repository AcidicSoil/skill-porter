Usage of the gum-based `convert-all-skills.sh` script:

1. Prerequisites

   * `skill-porter` installed and on `PATH`.
   * `gum` installed and on `PATH`.
   * Script is executable:

     ```bash
     chmod +x convert-all-skills.sh
     ```

2. Invocation

   * Run from a shell with no arguments:

     ```bash
     ./convert-all-skills.sh
     ```

3. Initial interactive prompts
   The script is fully interactive via `gum`:

   a) Scan root directory

   * Prompt:
     `Enter root directory to scan for skills (recursively):`
   * Default shown: `./skills`
   * Press Enter to accept `./skills` or type another directory path.

   b) Default conversion target

   * `gum choose` menu with header `Select default conversion target`
   * Options:

     * `gemini`
     * `claude`
   * Use arrow keys and Enter to select.

   c) Output base directory

   * Prompt:
     `Enter output base directory:`
   * Default: `./converted-<target>-skills`
   * Press Enter to accept or type a different path.
   * Script creates this directory if it does not exist.

4. Skill discovery behavior

   * Script recursively scans the selected root directory for files named:

     * `SKILL.md`
     * `gemini-extension.json`
   * Each parent directory of those files is treated as a “skill root”.
   * Duplicate directories (with both files) are deduplicated.

5. Per-skill interaction
   For each discovered skill directory:

   a) Header

   * A bordered `gum style` box is shown with:

     * `Skill:   <dir basename>`
     * `Path:    <absolute path>`
     * `Default: <current default target>`

   b) Action menu

   * `gum choose` menu with options:

     * `Convert (target=<current>)`
     * `Convert as gemini`
     * `Convert as claude`
     * `Skip this skill`
     * `Quit now`

   c) Action semantics

   * `Convert (target=<current>)`

     * Uses the current target (from default or previous override).
   * `Convert as gemini`

     * Overrides target to `gemini` for this skill.
   * `Convert as claude`

     * Overrides target to `claude` for this skill.
   * `Skip this skill`

     * Marks the skill as skipped and moves on to the next.
   * `Quit now`

     * Aborts immediately and prints the summary.

6. Conversion details

   * Output directory for each skill:

     ```bash
     ${OUT_BASE}/${skill-name}-${target}
     ```

   * Command executed (wrapped in `gum spin`):

     ```bash
     skill-porter convert "<skill-dir>" --to "<target>" --output "<out>"
     ```

   * On success:

     * Prints `OK: <skill-name> -> <target>`
     * Increments `success` counter.
   * On failure:

     * Prints error with exit code.
     * Increments `failed` counter.

7. Summary and exit

   * After all skills are processed, or if interrupted (`Ctrl+C`) or `Quit now` is selected, the script prints:

     ```text
     Summary:
       Succeeded: <success>
       Skipped:   <skipped>
       Failed:    <failed>
     ```

   * Exit code is `0` in all cases (including interrupted/quit), non-zero only if early checks fail (missing `skill-porter`, missing `gum`, or nonexistent scan root directory).
