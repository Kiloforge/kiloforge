#!/usr/bin/env bash
set -euo pipefail

# run_tests.sh — Run all kf-track and kf-track-content tests in isolation.
#
# Each test creates a temporary git repo with a fresh kf directory.
# Tests are fully isolated — no shared state between test cases.
#
# Usage: ./run_tests.sh [test-name-pattern]
#   Example: ./run_tests.sh conflicts   # run only tests matching "conflicts"

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
BIN_DIR="${SCRIPT_DIR}/.."
PATTERN="${1:-}"

# --- Test framework ---
TESTS_RUN=0
TESTS_PASSED=0
TESTS_FAILED=0
FAILURES=()

setup_test_env() {
  # Create a temporary directory with a git repo and kf structure
  TEST_DIR=$(mktemp -d)
  cd "$TEST_DIR"
  git init --quiet
  git commit --allow-empty -m "initial" --quiet

  mkdir -p .agent/kf/bin .agent/kf/tracks
  # Copy the tools under test
  cp "$BIN_DIR/kf-track" .agent/kf/bin/
  cp "$BIN_DIR/kf-track-content" .agent/kf/bin/
  chmod +x .agent/kf/bin/kf-track .agent/kf/bin/kf-track-content

  export PATH=".agent/kf/bin:$PATH"
}

teardown_test_env() {
  cd /
  rm -rf "$TEST_DIR"
}

run_test() {
  local name="$1"
  local func="$2"

  # Skip if pattern doesn't match
  if [[ -n "$PATTERN" && "$name" != *"$PATTERN"* ]]; then
    return
  fi

  ((TESTS_RUN++)) || true
  setup_test_env

  local result=0
  if "$func" 2>&1; then
    ((TESTS_PASSED++)) || true
    printf "  \033[32m✓\033[0m %s\n" "$name"
  else
    ((TESTS_FAILED++)) || true
    printf "  \033[31m✗\033[0m %s\n" "$name"
    FAILURES+=("$name")
  fi

  teardown_test_env
}

assert_eq() {
  local expected="$1" actual="$2" msg="${3:-}"
  if [[ "$expected" != "$actual" ]]; then
    echo "ASSERT_EQ FAILED${msg:+: $msg}"
    echo "  Expected: $expected"
    echo "  Actual:   $actual"
    return 1
  fi
}

assert_contains() {
  local haystack="$1" needle="$2" msg="${3:-}"
  if [[ "$haystack" != *"$needle"* ]]; then
    echo "ASSERT_CONTAINS FAILED${msg:+: $msg}"
    echo "  Looking for: $needle"
    echo "  In: $haystack"
    return 1
  fi
}

assert_not_contains() {
  local haystack="$1" needle="$2" msg="${3:-}"
  if [[ "$haystack" == *"$needle"* ]]; then
    echo "ASSERT_NOT_CONTAINS FAILED${msg:+: $msg}"
    echo "  Should not contain: $needle"
    echo "  In: $haystack"
    return 1
  fi
}

assert_file_exists() {
  local path="$1" msg="${2:-}"
  if [[ ! -f "$path" ]]; then
    echo "ASSERT_FILE_EXISTS FAILED${msg:+: $msg}"
    echo "  Not found: $path"
    return 1
  fi
}

assert_exit_code() {
  local expected="$1"
  shift
  local actual=0
  "$@" >/dev/null 2>&1 || actual=$?
  if [[ "$expected" != "$actual" ]]; then
    echo "ASSERT_EXIT_CODE FAILED"
    echo "  Expected: $expected"
    echo "  Actual:   $actual"
    echo "  Command:  $*"
    return 1
  fi
}

# ============================================================================
# kf-track tests (bash)
# ============================================================================

test_track_add_basic() {
  local out
  out=$(kf-track add test-track_20260101Z --title "Test Track" --type feature)
  assert_contains "$out" "Added: test-track_20260101Z"
  assert_file_exists ".agent/kf/tracks.yaml"
  local content
  content=$(cat .agent/kf/tracks.yaml)
  assert_contains "$content" "test-track_20260101Z"
  assert_contains "$content" '"title":"Test Track"'
  assert_contains "$content" '"status":"pending"'
}

test_track_add_with_deps() {
  kf-track add dep-track_20260101Z --title "Dep" --type feature >/dev/null
  kf-track add main-track_20260102Z --title "Main" --type feature --deps "dep-track_20260101Z" >/dev/null
  assert_file_exists ".agent/kf/tracks/deps.yaml"
  local deps
  deps=$(cat .agent/kf/tracks/deps.yaml)
  assert_contains "$deps" "main-track_20260102Z"
  assert_contains "$deps" "dep-track_20260101Z"
}

test_track_update_status() {
  kf-track add test-track_20260101Z --title "Test" --type feature >/dev/null
  kf-track update test-track_20260101Z --status in-progress >/dev/null
  local status
  status=$(kf-track get test-track_20260101Z 2>&1)
  assert_contains "$status" "in-progress"
}

test_track_update_invalid_status() {
  kf-track add test-track_20260101Z --title "Test" --type feature >/dev/null
  assert_exit_code 1 kf-track update test-track_20260101Z --status invalid
}

test_track_set_field() {
  kf-track add test-track_20260101Z --title "Test" --type feature >/dev/null
  kf-track set test-track_20260101Z --title "Updated Title" >/dev/null
  local line
  line=$(grep "test-track_20260101Z" .agent/kf/tracks.yaml)
  assert_contains "$line" '"title":"Updated Title"'
}

test_track_get_not_found() {
  kf-track add test-track_20260101Z --title "Test" --type feature >/dev/null
  assert_exit_code 1 kf-track get nonexistent_20260101Z
}

test_track_list_default_ready() {
  # Add two tracks: one with deps (not ready), one without (ready)
  kf-track add dep-track_20260101Z --title "Dep" --type feature >/dev/null
  kf-track add ready-track_20260102Z --title "Ready" --type feature >/dev/null
  kf-track add blocked-track_20260103Z --title "Blocked" --type feature --deps "dep-track_20260101Z" >/dev/null

  local out
  out=$(kf-track list 2>&1)
  assert_contains "$out" "ready-track_20260102Z"
  assert_contains "$out" "dep-track_20260101Z"
  # blocked-track should still show but with deps info
}

test_track_list_active() {
  kf-track add t1_20260101Z --title "T1" --type feature >/dev/null
  kf-track add t2_20260102Z --title "T2" --type feature >/dev/null
  kf-track update t1_20260101Z --status completed >/dev/null

  local out
  out=$(kf-track list --active 2>&1)
  assert_contains "$out" "t2_20260102Z"
  assert_not_contains "$out" "t1_20260101Z"
}

test_track_list_all() {
  kf-track add t1_20260101Z --title "T1" --type feature >/dev/null
  kf-track update t1_20260101Z --status completed >/dev/null

  local out
  out=$(kf-track list --all 2>&1)
  assert_contains "$out" "t1_20260101Z"
}

test_track_list_ids() {
  kf-track add t1_20260101Z --title "T1" --type feature >/dev/null
  kf-track add t2_20260102Z --title "T2" --type feature >/dev/null

  local out
  out=$(kf-track list --active --ids 2>&1)
  assert_contains "$out" "t1_20260101Z"
  assert_contains "$out" "t2_20260102Z"
}

test_track_archive() {
  kf-track add test-track_20260101Z --title "Test" --type feature >/dev/null
  kf-track archive test-track_20260101Z "done" >/dev/null
  local line
  line=$(grep "test-track_20260101Z" .agent/kf/tracks.yaml)
  assert_contains "$line" '"status":"archived"'
  assert_contains "$line" '"archive_reason":"done"'
}

test_track_archive_removes_deps() {
  kf-track add dep-track_20260101Z --title "Dep" --type feature >/dev/null
  kf-track add main-track_20260102Z --title "Main" --type feature --deps "dep-track_20260101Z" >/dev/null
  kf-track archive dep-track_20260101Z >/dev/null
  # dep-track should be removed from deps.yaml as a key entry
  if grep -q "^dep-track_20260101Z:" .agent/kf/tracks/deps.yaml 2>/dev/null; then
    echo "dep-track should have been removed from deps.yaml"
    return 1
  fi
}

test_track_canonical_json_order() {
  kf-track add test-track_20260101Z --title "Test" --type feature >/dev/null
  local line
  line=$(grep "test-track_20260101Z" .agent/kf/tracks.yaml)
  # Verify title comes before status which comes before type
  local title_pos status_pos type_pos
  title_pos=$(echo "$line" | grep -bo '"title"' | head -1 | cut -d: -f1)
  status_pos=$(echo "$line" | grep -bo '"status"' | head -1 | cut -d: -f1)
  type_pos=$(echo "$line" | grep -bo '"type"' | head -1 | cut -d: -f1)
  if [[ $title_pos -gt $status_pos || $status_pos -gt $type_pos ]]; then
    echo "JSON field order is not canonical: title=$title_pos status=$status_pos type=$type_pos"
    return 1
  fi
}

test_track_alphabetical_sort() {
  kf-track add zebra_20260101Z --title "Zebra" --type feature >/dev/null
  kf-track add alpha_20260102Z --title "Alpha" --type feature >/dev/null
  # alpha should come before zebra in the file
  local alpha_line zebra_line
  alpha_line=$(grep -n "alpha_20260102Z" .agent/kf/tracks.yaml | cut -d: -f1)
  zebra_line=$(grep -n "zebra_20260101Z" .agent/kf/tracks.yaml | cut -d: -f1)
  if [[ $alpha_line -ge $zebra_line ]]; then
    echo "Alphabetical sort failed: alpha at line $alpha_line, zebra at line $zebra_line"
    return 1
  fi
}

# ============================================================================
# Deps tests
# ============================================================================

test_deps_add() {
  kf-track add a_20260101Z --title "A" --type feature >/dev/null
  kf-track add b_20260102Z --title "B" --type feature >/dev/null
  kf-track deps add b_20260102Z a_20260101Z >/dev/null
  local out
  out=$(kf-track deps list b_20260102Z 2>&1)
  assert_contains "$out" "a_20260101Z"
}

test_deps_check_satisfied() {
  kf-track add a_20260101Z --title "A" --type feature >/dev/null
  kf-track add b_20260102Z --title "B" --type feature --deps "a_20260101Z" >/dev/null
  kf-track update a_20260101Z --status completed >/dev/null
  assert_exit_code 0 kf-track deps check b_20260102Z
}

test_deps_check_blocked() {
  kf-track add a_20260101Z --title "A" --type feature >/dev/null
  kf-track add b_20260102Z --title "B" --type feature --deps "a_20260101Z" >/dev/null
  assert_exit_code 1 kf-track deps check b_20260102Z
}

test_deps_remove() {
  kf-track add a_20260101Z --title "A" --type feature >/dev/null
  kf-track add b_20260102Z --title "B" --type feature --deps "a_20260101Z" >/dev/null
  kf-track deps remove b_20260102Z a_20260101Z >/dev/null
  local out
  out=$(kf-track deps list b_20260102Z 2>&1)
  assert_contains "$out" "(no dependencies)"
}

test_deps_cleaned_on_complete() {
  kf-track add a_20260101Z --title "A" --type feature >/dev/null
  kf-track add b_20260102Z --title "B" --type feature --deps "a_20260101Z" >/dev/null
  kf-track update a_20260101Z --status completed >/dev/null
  # a's own entry should be removed from deps.yaml
  if grep -q "^a_20260101Z:" .agent/kf/tracks/deps.yaml 2>/dev/null; then
    echo "Completed track entry should have been removed from deps.yaml"
    return 1
  fi
}

# ============================================================================
# Conflicts tests
# ============================================================================

test_conflicts_add_basic() {
  local out
  out=$(kf-track conflicts add track-a_20260101Z track-b_20260102Z high "overlapping files" 2>&1)
  assert_contains "$out" "Added conflict pair"
  assert_file_exists ".agent/kf/tracks/conflicts.yaml"
}

test_conflicts_pair_ordering() {
  # Regardless of argument order, the pair key should be alphabetically sorted
  kf-track conflicts add zebra_20260101Z alpha_20260102Z medium "test" >/dev/null
  local content
  content=$(grep -v "^#" .agent/kf/tracks/conflicts.yaml | grep -v "^$")
  # Should be alpha/zebra, not zebra/alpha
  assert_contains "$content" "alpha_20260102Z/zebra_20260101Z"
}

test_conflicts_update_semantics() {
  kf-track conflicts add a_20260101Z b_20260102Z high "first" >/dev/null
  kf-track conflicts add a_20260101Z b_20260102Z low "updated" >/dev/null
  # Should only have one entry
  local count
  count=$(grep -c "a_20260101Z/b_20260102Z" .agent/kf/tracks/conflicts.yaml || echo 0)
  assert_eq "1" "$count" "Should have exactly one entry per pair"
  local content
  content=$(grep "a_20260101Z/b_20260102Z" .agent/kf/tracks/conflicts.yaml)
  assert_contains "$content" '"risk":"low"'
}

test_conflicts_remove() {
  kf-track conflicts add a_20260101Z b_20260102Z high "test" >/dev/null
  kf-track conflicts remove a_20260101Z b_20260102Z >/dev/null
  local out
  out=$(kf-track conflicts list 2>&1)
  assert_contains "$out" "(no conflict pairs)"
}

test_conflicts_list_filtered() {
  kf-track conflicts add a_20260101Z b_20260102Z high "test1" >/dev/null
  kf-track conflicts add c_20260103Z d_20260104Z low "test2" >/dev/null
  local out
  out=$(kf-track conflicts list a_20260101Z 2>&1)
  assert_contains "$out" "a_20260101Z"
  assert_not_contains "$out" "c_20260103Z"
}

test_conflicts_cleaned_on_archive() {
  kf-track add a_20260101Z --title "A" --type feature >/dev/null
  kf-track add b_20260102Z --title "B" --type feature >/dev/null
  kf-track conflicts add a_20260101Z b_20260102Z high "test" >/dev/null
  kf-track archive a_20260101Z "done" >/dev/null
  local out
  out=$(kf-track conflicts list 2>&1)
  assert_contains "$out" "(no conflict pairs)"
}

test_conflicts_cleaned_on_complete() {
  kf-track add a_20260101Z --title "A" --type feature >/dev/null
  kf-track add b_20260102Z --title "B" --type feature >/dev/null
  kf-track conflicts add a_20260101Z b_20260102Z medium "test" >/dev/null
  kf-track update a_20260101Z --status completed >/dev/null
  # The pair should be cleaned since a is completed
  local content
  content=$(grep -v "^#" .agent/kf/tracks/conflicts.yaml 2>/dev/null | grep -v "^$" || true)
  if [[ -n "$content" ]]; then
    echo "Conflict pair should have been cleaned on completion"
    echo "Content: $content"
    return 1
  fi
}

test_conflicts_self_pair_rejected() {
  assert_exit_code 1 kf-track conflicts add same_20260101Z same_20260101Z high "self"
}

# ============================================================================
# kf-track-content tests (python)
# ============================================================================

test_content_init() {
  kf-track-content init test_20260101Z --title "Test Track" --type feature --summary "A test" >/dev/null
  assert_file_exists ".agent/kf/tracks/test_20260101Z/track.yaml"
  local content
  content=$(cat .agent/kf/tracks/test_20260101Z/track.yaml)
  assert_contains "$content" "title: Test Track"
  assert_contains "$content" "type: feature"
  assert_contains "$content" "summary: A test"
}

test_content_init_duplicate_rejected() {
  kf-track-content init test_20260101Z --title "Test" >/dev/null
  assert_exit_code 1 kf-track-content init test_20260101Z --title "Test2"
}

test_content_show_full() {
  kf-track-content init test_20260101Z --title "Test Track" --summary "Summary here" >/dev/null
  local out
  out=$(kf-track-content show test_20260101Z 2>&1)
  assert_contains "$out" "Test Track"
  assert_contains "$out" "Summary here"
}

test_content_show_section_header() {
  kf-track-content init test_20260101Z --title "Test Track" >/dev/null
  local out
  out=$(kf-track-content show test_20260101Z --section header 2>&1)
  assert_contains "$out" "id: test_20260101Z"
  assert_contains "$out" "title: Test Track"
  assert_contains "$out" "type: feature"
}

test_content_show_section_spec() {
  kf-track-content init test_20260101Z --title "Test" --summary "My summary" >/dev/null
  local out
  out=$(kf-track-content show test_20260101Z --section spec 2>&1)
  assert_contains "$out" "spec:"
  assert_contains "$out" "summary: My summary"
}

test_content_show_json() {
  kf-track-content init test_20260101Z --title "Test" >/dev/null
  local out
  out=$(kf-track-content show test_20260101Z --json 2>&1)
  # Should be valid JSON
  echo "$out" | python3 -c "import sys,json; json.load(sys.stdin)" || return 1
}

test_content_spec_set_field() {
  kf-track-content init test_20260101Z --title "Test" >/dev/null
  kf-track-content spec test_20260101Z --field context --set "New context" >/dev/null
  local out
  out=$(kf-track-content spec test_20260101Z --field context 2>&1)
  assert_eq "New context" "$out"
}

test_content_spec_append_criteria() {
  kf-track-content init test_20260101Z --title "Test" >/dev/null
  kf-track-content spec test_20260101Z --field acceptance_criteria --append "Criterion 1" >/dev/null
  kf-track-content spec test_20260101Z --field acceptance_criteria --append "Criterion 2" >/dev/null
  local out
  out=$(kf-track-content spec test_20260101Z --field acceptance_criteria 2>&1)
  assert_contains "$out" "Criterion 1"
  assert_contains "$out" "Criterion 2"
}

test_content_task_done() {
  # Create a track with a plan
  mkdir -p .agent/kf/tracks/test_20260101Z
  cat > .agent/kf/tracks/test_20260101Z/track.yaml <<'YAML'
id: test_20260101Z
title: Test
type: feature
status: pending
created: 2026-01-01
updated: 2026-01-01
spec:
  summary: test
plan:
  - phase: Setup
    tasks:
      - text: Task one
        done: false
      - text: Task two
        done: false
extra: {}
YAML
  kf-track-content task test_20260101Z 1.1 --done >/dev/null
  local out
  out=$(kf-track-content progress test_20260101Z 2>&1)
  assert_contains "$out" "1/2 tasks"
}

test_content_task_pending() {
  mkdir -p .agent/kf/tracks/test_20260101Z
  cat > .agent/kf/tracks/test_20260101Z/track.yaml <<'YAML'
id: test_20260101Z
title: Test
type: feature
status: pending
created: 2026-01-01
updated: 2026-01-01
spec:
  summary: test
plan:
  - phase: Setup
    tasks:
      - text: Task one
        done: true
      - text: Task two
        done: false
extra: {}
YAML
  kf-track-content task test_20260101Z 1.1 --pending >/dev/null
  local out
  out=$(kf-track-content progress test_20260101Z 2>&1)
  assert_contains "$out" "0/2 tasks"
}

test_content_progress() {
  mkdir -p .agent/kf/tracks/test_20260101Z
  cat > .agent/kf/tracks/test_20260101Z/track.yaml <<'YAML'
id: test_20260101Z
title: Test
type: feature
status: pending
created: 2026-01-01
updated: 2026-01-01
spec:
  summary: test
plan:
  - phase: Phase 1
    tasks:
      - text: T1
        done: true
      - text: T2
        done: true
  - phase: Phase 2
    tasks:
      - text: T3
        done: false
extra: {}
YAML
  local out
  out=$(kf-track-content progress test_20260101Z 2>&1)
  assert_contains "$out" "2/3 tasks"
  assert_contains "$out" "1/2 phases"
}

test_content_progress_json() {
  mkdir -p .agent/kf/tracks/test_20260101Z
  cat > .agent/kf/tracks/test_20260101Z/track.yaml <<'YAML'
id: test_20260101Z
title: Test
type: feature
status: pending
created: 2026-01-01
updated: 2026-01-01
spec:
  summary: test
plan:
  - phase: P1
    tasks:
      - text: T1
        done: true
extra: {}
YAML
  local out
  out=$(kf-track-content progress test_20260101Z --json 2>&1)
  echo "$out" | python3 -c "import sys,json; d=json.load(sys.stdin); assert d['percent']==100" || return 1
}

test_content_extra_set() {
  kf-track-content init test_20260101Z --title "Test" >/dev/null
  kf-track-content extra test_20260101Z --key pr_url --set "https://github.com/test/pr/1" >/dev/null
  local out
  out=$(kf-track-content extra test_20260101Z --key pr_url 2>&1)
  assert_eq "https://github.com/test/pr/1" "$out"
}

test_content_extra_delete() {
  kf-track-content init test_20260101Z --title "Test" >/dev/null
  kf-track-content extra test_20260101Z --key tmp --set "val" >/dev/null
  kf-track-content extra test_20260101Z --key tmp --delete >/dev/null
  local out
  out=$(kf-track-content extra test_20260101Z --key tmp 2>&1)
  assert_eq "" "$out"
}

test_content_migrate_legacy() {
  # Create legacy format files
  mkdir -p .agent/kf/tracks/legacy_20260101Z
  cat > .agent/kf/tracks/legacy_20260101Z/metadata.json <<'JSON'
{"id":"legacy_20260101Z","title":"Legacy Track","type":"feature","status":"pending","created":"2026-01-01","updated":"2026-01-01"}
JSON
  cat > .agent/kf/tracks/legacy_20260101Z/spec.md <<'MD'
# Specification: Legacy Track

## Summary

This is the summary.

## Context

Some context here.

## Acceptance Criteria

- [ ] First criterion
- [ ] Second criterion

## Dependencies

- dep-track-id

## Out of Scope

Nothing here.
MD
  cat > .agent/kf/tracks/legacy_20260101Z/plan.md <<'MD'
## Phase 1: Setup

- [ ] Task 1.1: Do thing one
- [ ] Task 1.2: Do thing two

## Phase 2: Implementation

- [ ] Task 2.1: Do thing three
MD

  kf-track-content migrate legacy_20260101Z >/dev/null
  assert_file_exists ".agent/kf/tracks/legacy_20260101Z/track.yaml"
  # Legacy files should be removed
  if [[ -f ".agent/kf/tracks/legacy_20260101Z/spec.md" ]]; then
    echo "Legacy spec.md should have been removed"
    return 1
  fi
  local content
  content=$(cat .agent/kf/tracks/legacy_20260101Z/track.yaml)
  assert_contains "$content" "This is the summary"
  assert_contains "$content" "First criterion"
  # Dependencies should NOT be in the track.yaml (they're in deps.yaml)
  assert_not_contains "$content" "dep-track-id"
  assert_not_contains "$content" "dependencies"
}

test_content_show_not_found() {
  assert_exit_code 1 kf-track-content show nonexistent_20260101Z
}

# ============================================================================
# Integration tests
# ============================================================================

test_full_lifecycle() {
  # Create two tracks with dependency
  kf-track add infra_20260101Z --title "Infra Setup" --type chore >/dev/null
  kf-track add feature_20260102Z --title "Feature" --type feature --deps "infra_20260101Z" >/dev/null
  kf-track-content init infra_20260101Z --title "Infra Setup" --type chore --summary "Setup" >/dev/null
  kf-track-content init feature_20260102Z --title "Feature" --type feature --summary "Feature" >/dev/null

  # Add conflict pair
  kf-track conflicts add infra_20260101Z feature_20260102Z medium "shared config" >/dev/null

  # Feature should be blocked
  assert_exit_code 1 kf-track deps check feature_20260102Z

  # Complete infra
  kf-track update infra_20260101Z --status completed >/dev/null

  # Feature should now be unblocked
  assert_exit_code 0 kf-track deps check feature_20260102Z

  # Conflict pair should be cleaned (infra is completed)
  local conflicts
  conflicts=$(kf-track conflicts list 2>&1)
  assert_contains "$conflicts" "(no conflict pairs)"

  # Complete feature
  kf-track update feature_20260102Z --status completed >/dev/null

  # Both should show in --all
  local out
  out=$(kf-track list --all 2>&1)
  assert_contains "$out" "infra_20260101Z"
  assert_contains "$out" "feature_20260102Z"
}

test_yaml_roundtrip() {
  # Create a track via CLI, read it back, verify content survives roundtrip
  kf-track-content init test_20260101Z --title "Roundtrip Test" --type feature --summary "Test roundtrip" >/dev/null
  kf-track-content spec test_20260101Z --field context --set "Context with special chars: colons, #hashes, and 'quotes'" >/dev/null
  kf-track-content spec test_20260101Z --field acceptance_criteria --append "Criterion with: colon" >/dev/null

  # Read back
  local context
  context=$(kf-track-content spec test_20260101Z --field context 2>&1)
  assert_eq "Context with special chars: colons, #hashes, and 'quotes'" "$context"

  local criteria
  criteria=$(kf-track-content spec test_20260101Z --field acceptance_criteria 2>&1)
  assert_contains "$criteria" "Criterion with: colon"
}

# ============================================================================
# Run all tests
# ============================================================================

echo "Running kf-track test suite..."
echo ""
echo "kf-track (bash):"
run_test "track_add_basic" test_track_add_basic
run_test "track_add_with_deps" test_track_add_with_deps
run_test "track_update_status" test_track_update_status
run_test "track_update_invalid_status" test_track_update_invalid_status
run_test "track_set_field" test_track_set_field
run_test "track_get_not_found" test_track_get_not_found
run_test "track_list_default_ready" test_track_list_default_ready
run_test "track_list_active" test_track_list_active
run_test "track_list_all" test_track_list_all
run_test "track_list_ids" test_track_list_ids
run_test "track_archive" test_track_archive
run_test "track_archive_removes_deps" test_track_archive_removes_deps
run_test "track_canonical_json_order" test_track_canonical_json_order
run_test "track_alphabetical_sort" test_track_alphabetical_sort

echo ""
echo "deps:"
run_test "deps_add" test_deps_add
run_test "deps_check_satisfied" test_deps_check_satisfied
run_test "deps_check_blocked" test_deps_check_blocked
run_test "deps_remove" test_deps_remove
run_test "deps_cleaned_on_complete" test_deps_cleaned_on_complete

echo ""
echo "conflicts:"
run_test "conflicts_add_basic" test_conflicts_add_basic
run_test "conflicts_pair_ordering" test_conflicts_pair_ordering
run_test "conflicts_update_semantics" test_conflicts_update_semantics
run_test "conflicts_remove" test_conflicts_remove
run_test "conflicts_list_filtered" test_conflicts_list_filtered
run_test "conflicts_cleaned_on_archive" test_conflicts_cleaned_on_archive
run_test "conflicts_cleaned_on_complete" test_conflicts_cleaned_on_complete
run_test "conflicts_self_pair_rejected" test_conflicts_self_pair_rejected

echo ""
echo "kf-track-content (python):"
run_test "content_init" test_content_init
run_test "content_init_duplicate_rejected" test_content_init_duplicate_rejected
run_test "content_show_full" test_content_show_full
run_test "content_show_section_header" test_content_show_section_header
run_test "content_show_section_spec" test_content_show_section_spec
run_test "content_show_json" test_content_show_json
run_test "content_spec_set_field" test_content_spec_set_field
run_test "content_spec_append_criteria" test_content_spec_append_criteria
run_test "content_task_done" test_content_task_done
run_test "content_task_pending" test_content_task_pending
run_test "content_progress" test_content_progress
run_test "content_progress_json" test_content_progress_json
run_test "content_extra_set" test_content_extra_set
run_test "content_extra_delete" test_content_extra_delete
run_test "content_migrate_legacy" test_content_migrate_legacy
run_test "content_show_not_found" test_content_show_not_found

echo ""
echo "integration:"
run_test "full_lifecycle" test_full_lifecycle
run_test "yaml_roundtrip" test_yaml_roundtrip

echo ""
echo "========================================"
printf "Results: %d passed, %d failed, %d total\n" "$TESTS_PASSED" "$TESTS_FAILED" "$TESTS_RUN"
echo "========================================"

if [[ ${#FAILURES[@]} -gt 0 ]]; then
  echo ""
  echo "Failed tests:"
  for f in "${FAILURES[@]}"; do
    echo "  - $f"
  done
  exit 1
fi
