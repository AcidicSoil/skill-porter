#!/usr/bin/env bash
set -uo pipefail

###############################################################################
# Counters and helpers
###############################################################################

success=0
skipped=0
failed=0
found_skills=0
RECURSIVE_MODE=1
AUTO_CONVERT_MODE=0
AUTO_TARGET=""

summarize_and_exit() {
  echo
  echo "Summary:"
  echo "  Succeeded: $success"
  echo "  Skipped:   $skipped"
  echo "  Failed:    $failed"
  echo
  exit 0
}

trap 'echo; echo "Interrupted by user."; summarize_and_exit' INT

if ! command -v skill-porter >/dev/null 2>&1; then
  echo "ERROR: skill-porter not found on PATH. Install or link it first." >&2
  exit 1
fi

if ! command -v gum >/dev/null 2>&1; then
  echo "ERROR: gum not found on PATH. Install it first." >&2
  exit 1
fi

###############################################################################
# Initial interactive prompts (gum)
###############################################################################

DEFAULT_SKILLS_DIR="./skills"

SKILLS_DIR="$(gum input \
  --placeholder "Skill directory or parent" \
  --value "$DEFAULT_SKILLS_DIR" \
  --prompt "Enter root directory (skill dir or parent): ")"
SKILLS_DIR="${SKILLS_DIR:-$DEFAULT_SKILLS_DIR}"

if [ ! -d "$SKILLS_DIR" ]; then
  echo "ERROR: Directory does not exist: $SKILLS_DIR" >&2
  exit 1
fi

if gum confirm "Recursively scan this directory and its subdirectories and convert each matching skill?"; then
  RECURSIVE_MODE=1
else
  RECURSIVE_MODE=0
fi

TARGET_DEFAULT="$(printf '%s\n' gemini claude | gum choose \
  --header "Select default conversion target")"

DEFAULT_OUT_BASE="./converted-${TARGET_DEFAULT}-skills"

OUT_BASE="$(gum input \
  --placeholder "Output base directory" \
  --value "$DEFAULT_OUT_BASE" \
  --prompt "Enter output base directory: ")"
OUT_BASE="${OUT_BASE:-$DEFAULT_OUT_BASE}"

mkdir -p "$OUT_BASE"

echo
echo "Configured:"
echo "  Scan root:       $SKILLS_DIR"
if [[ "$RECURSIVE_MODE" -eq 1 ]]; then
  echo "  Scan mode:       Recursive (all skills under root)"
else
  echo "  Scan mode:       Single skill (root dir only)"
fi
echo "  Default target:  $TARGET_DEFAULT"
echo "  Output base dir: $OUT_BASE"
echo

###############################################################################
# Per-skill logic with gum UI and error handling
###############################################################################

convert_one() {
  local skill="$1"
  local target="$2"
  local name out choice

  name="$(basename "$skill")"

  if [[ ! -f "$skill/SKILL.md" && ! -f "$skill/gemini-extension.json" ]]; then
    echo "Skipping $name: no SKILL.md or gemini-extension.json (not a skill root)."
    skipped=$((skipped + 1))
    return 0
  fi

  if [[ "$AUTO_CONVERT_MODE" -eq 1 ]]; then
    target="$AUTO_TARGET"
    gum style \
      --border normal \
      --margin "1 0" \
      --padding "0 1" \
      "Skill:   $name" \
      "Path:    $skill" \
      "Mode:    AUTO (target=$target)"
  else
    gum style \
      --border normal \
      --margin "1 0" \
      --padding "0 1" \
      "Skill:   $name" \
      "Path:    $skill" \
      "Default: $target"

    choice="$(gum choose \
      "Convert (target=$target)" \
      "Convert as gemini" \
      "Convert as claude" \
      "Convert ALL remaining (target=$target)" \
      "Skip this skill" \
      "Quit now")"

    case "$choice" in
      "Convert (target=$target)")
        ;;
      "Convert as gemini")
        target="gemini"
        ;;
      "Convert as claude")
        target="claude"
        ;;
      "Convert ALL remaining (target=$target)")
        AUTO_CONVERT_MODE=1
        AUTO_TARGET="$target"
        ;;
      "Skip this skill")
        echo "Skipping: $name"
        skipped=$((skipped + 1))
        return 0
        ;;
      "Quit now")
        echo "Aborting on user request."
        summarize_and_exit
        ;;
    esac
  fi

  out="${OUT_BASE}/${name}-${target}"
  mkdir -p "$out"

  echo
  if [[ "$AUTO_CONVERT_MODE" -eq 1 ]]; then
    echo "Auto-converting: $name"
  else
    echo "Converting: $name"
  fi
  echo "  Source: $skill"
  echo "  Target: $target"
  echo "  Output: $out"
  echo

  if gum spin --title "Converting $name -> $target" -- \
    skill-porter convert "$skill" --to "$target" --output "$out"
  then
    echo
    echo "OK: $name -> $target"
    success=$((success + 1))
  else
    status=$?
    echo
    echo "ERROR: Conversion failed for $name (exit code $status)."
    echo "       See skill-porter output above."
    failed=$((failed + 1))
  fi

  echo
}

###############################################################################
# Discover skills (recursive or single) and process them
###############################################################################

if [[ "$RECURSIVE_MODE" -eq 1 ]]; then
  declare -A SEEN_DIRS
  SKILL_FILES=()

  mapfile -d '' -t SKILL_FILES < <(
    find "$SKILLS_DIR" \
      -type f \( -name "SKILL.md" -o -name "gemini-extension.json" \) \
      -print0
  )

  for file in "${SKILL_FILES[@]}"; do
    [[ -z "$file" ]] && continue

    dir="$(dirname "$file")"
    dir="$(cd "$dir" && pwd)"

    if [[ -n "${SEEN_DIRS[$dir]:-}" ]]; then
      continue
    fi
    SEEN_DIRS["$dir"]=1

    found_skills=$((found_skills + 1))
    convert_one "$dir" "$TARGET_DEFAULT"
  done

  if (( found_skills == 0 )); then
    echo "No skill directories found under: $SKILLS_DIR"
  fi
else
  if [[ -f "$SKILLS_DIR/SKILL.md" || -f "$SKILLS_DIR/gemini-extension.json" ]]; then
    found_skills=1
    convert_one "$SKILLS_DIR" "$TARGET_DEFAULT"
  else
    echo "No skill files (SKILL.md or gemini-extension.json) found in: $SKILLS_DIR"
  fi
fi

summarize_and_exit
