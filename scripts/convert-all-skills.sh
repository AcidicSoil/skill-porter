#!/usr/bin/env bash
set -uo pipefail

###############################################################################
# Counters and helpers
###############################################################################

success=0
skipped=0
failed=0
found_skills=0

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
  --placeholder "Scan root directory" \
  --value "$DEFAULT_SKILLS_DIR" \
  --prompt "Enter root directory to scan for skills (recursively): ")"
SKILLS_DIR="${SKILLS_DIR:-$DEFAULT_SKILLS_DIR}"

if [ ! -d "$SKILLS_DIR" ]; then
  echo "ERROR: Directory does not exist: $SKILLS_DIR" >&2
  exit 1
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

  # Safety check in case this directory was matched oddly
  if [[ ! -f "$skill/SKILL.md" && ! -f "$skill/gemini-extension.json" ]]; then
    echo "Skipping $name: no SKILL.md or gemini-extension.json (not a skill root)."
    skipped=$((skipped + 1))
    return 0
  fi

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
    "Skip this skill" \
    "Quit now")"

  case "$choice" in
    "Convert (target=$target)")
      # use current target
      ;;
    "Convert as gemini")
      target="gemini"
      ;;
    "Convert as claude")
      target="claude"
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

  out="${OUT_BASE}/${name}-${target}"
  mkdir -p "$out"

  echo
  echo "Converting: $name"
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
# Recursively discover skill roots and process them
###############################################################################

# Use an associative array to deduplicate directories that contain both files
declare -A SEEN_DIRS

while IFS= read -r -d '' file; do
  dir="$(dirname "$file")"

  # Normalize to absolute path to avoid duplicates via different prefixes
  dir="$(cd "$dir" && pwd)"

  if [[ -n "${SEEN_DIRS[$dir]:-}" ]]; then
    continue
  fi
  SEEN_DIRS["$dir"]=1

  found_skills=$((found_skills + 1))
  convert_one "$dir" "$TARGET_DEFAULT"

done < <(find "$SKILLS_DIR" \
            -type f \( -name "SKILL.md" -o -name "gemini-extension.json" \) \
            -print0)

if (( found_skills == 0 )); then
  echo "No skill directories found under: $SKILLS_DIR"
fi

summarize_and_exit
