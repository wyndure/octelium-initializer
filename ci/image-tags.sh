#!/bin/sh
set -eu

# Only publish on push events.
[ "${GITHUB_EVENT_NAME:-}" = "push" ] || exit 0

branch_slug() {
  printf '%s' "$1" |
    LC_ALL=C tr '[:upper:]' '[:lower:]' |
    sed 's/[^a-z0-9]/-/g; s/^-*//; s/-*$//' |
    cut -c1-63 |
    sed 's/-*$//'
}

case "${GITHUB_REF_TYPE:-}:${GITHUB_REF_NAME:-}" in
  tag:*)
    ref=$GITHUB_REF_NAME

    if ! printf '%s\n' "$ref" |
      grep -Eq '^[A-Za-z0-9_][A-Za-z0-9_.-]{0,127}$'; then
      printf 'Invalid container image tag: %s\n' "$ref" >&2
      exit 1
    fi

    case "$ref" in
      v[0-9]*) tags="$ref,${ref#v},latest" ;;
      *)       tags="$ref,latest" ;;
    esac
    ;;

  branch:main)
    tags="dev-${GITHUB_RUN_ID:?},dev"
    ;;

  branch:*)
    [ "${GITHUB_REF_PROTECTED:-false}" = "true" ] || exit 0

    slug=$(branch_slug "$GITHUB_REF_NAME")
    tags="dev-${GITHUB_RUN_ID:?},dev-${slug:-branch}"
    ;;

  *)
    exit 0
    ;;
esac

image=${IMAGE_REPOSITORY:-ghcr.io/${GITHUB_REPOSITORY:?}}
image=$(printf '%s' "$image" | LC_ALL=C tr '[:upper:]' '[:lower:]')

printf '%s\n' "$tags" |
  tr ',' '\n' |
  sed "s|^|$image:|" |
  paste -sd, -
