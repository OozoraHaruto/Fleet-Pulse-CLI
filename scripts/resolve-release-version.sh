#!/usr/bin/env bash
set -euo pipefail

mode=""
version_file="version.txt"

while (($#)); do
  case "$1" in
    --mode)
      shift
      mode="${1:-}"
      ;;
    --version-file)
      shift
      version_file="${1:-}"
      ;;
    *)
      echo "unknown argument: $1" >&2
      exit 2
      ;;
  esac
  shift
done

if [ -z "$mode" ]; then
  echo "--mode is required" >&2
  exit 2
fi

read_local_version() {
  tr -d '[:space:]' <"$version_file"
}

read_main_version() {
  git show origin/main:version.txt 2>/dev/null | tr -d '[:space:]'
}

validate_final_version() {
  local version="$1"
  case "$version" in
    v[0-9]*.[0-9]*.[0-9]*)
      if [[ "$version" =~ ^v[0-9]+\.[0-9]+\.[0-9]+$ ]]; then
        return 0
      fi
      ;;
  esac

  echo "version must look like v1.2.3: $version" >&2
  exit 1
}

base_version=""
case "$mode" in
  ci)
    if base_version="$(read_main_version)" && [ -n "$base_version" ]; then
      :
    else
      base_version="$(read_local_version)"
    fi
    validate_final_version "$base_version"

    : "${GITHUB_RUN_NUMBER:?GITHUB_RUN_NUMBER is required in ci mode}"
    : "${GITHUB_SHA:?GITHUB_SHA is required in ci mode}"
    short_sha="${GITHUB_SHA:0:7}"

    echo "version=${base_version}-ci.${GITHUB_RUN_NUMBER}.${short_sha}"
    echo "prerelease=true"
    ;;
  final)
    base_version="$(read_local_version)"
    validate_final_version "$base_version"

    echo "version=$base_version"
    echo "prerelease=false"
    ;;
  *)
    echo "unsupported mode: $mode" >&2
    exit 2
    ;;
esac
