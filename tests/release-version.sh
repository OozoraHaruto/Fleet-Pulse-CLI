#!/usr/bin/env bash
set -euo pipefail

repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd -P)"
tmp_dir="$(mktemp -d)"
trap 'rm -rf "$tmp_dir"' EXIT

script="$repo_root/scripts/resolve-release-version.sh"
local_version_file="$tmp_dir/version.txt"

printf '%s\n' 'v1.0.0' >"$local_version_file"

assert_output() {
  local name="$1"
  local want_version="$2"
  local want_prerelease="$3"
  shift 3

  local output
  if ! output="$("$@")"; then
    echo "$name: command failed" >&2
    exit 1
  fi

  if ! grep -Fxq "version=$want_version" <<<"$output"; then
    echo "$name: missing version=$want_version in output:" >&2
    printf '%s\n' "$output" >&2
    exit 1
  fi
  if ! grep -Fxq "prerelease=$want_prerelease" <<<"$output"; then
    echo "$name: missing prerelease=$want_prerelease in output:" >&2
    printf '%s\n' "$output" >&2
    exit 1
  fi
}

git_main="$tmp_dir/git-main"
mkdir -p "$git_main"
cat >"$git_main/git" <<'GIT'
#!/usr/bin/env bash
set -euo pipefail

if [ "$1" = "show" ] && [ "$2" = "origin/main:version.txt" ]; then
  printf '%s\n' 'v2.3.4'
  exit 0
fi

exit 1
GIT
chmod 0755 "$git_main/git"

assert_output "ci release reads main version" "v2.3.4-ci.123.abcdef1" "true" \
  env PATH="$git_main:$PATH" GITHUB_RUN_NUMBER=123 GITHUB_SHA=abcdef1234567890 \
  "$script" --mode ci

git_missing="$tmp_dir/git-missing"
mkdir -p "$git_missing"
cat >"$git_missing/git" <<'GIT'
#!/usr/bin/env bash
exit 1
GIT
chmod 0755 "$git_missing/git"

assert_output "ci release falls back to local version" "v1.0.0-ci.7.0123456" "true" \
  env PATH="$git_missing:$PATH" GITHUB_RUN_NUMBER=7 GITHUB_SHA=0123456789abcdef \
  "$script" --mode ci --version-file "$local_version_file"

assert_output "final release uses local version" "v1.0.0" "false" \
  "$script" --mode final --version-file "$local_version_file"

if ! "$script" --mode final --version-file "$repo_root/version.txt" >/dev/null; then
  echo "repo version.txt is not a valid final release version" >&2
  exit 1
fi
