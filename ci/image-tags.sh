#!/bin/sh
set -eu

# Publishing is restricted to push events, including when this helper is used alone.
[ "${GITHUB_EVENT_NAME:-}" = push ] || exit 0

if [ "${GITHUB_REF_TYPE:-}" = tag ]; then
  if ! printf '%s\n' "${GITHUB_REF_NAME:-}" | grep -Eq '^[A-Za-z0-9_][A-Za-z0-9_.-]{0,127}$'; then
    printf 'Git tag is not a valid container image tag: %s\n' "${GITHUB_REF_NAME:-}" >&2
    exit 1
  fi

  image_tags="$GITHUB_REF_NAME,latest"
elif [ "${GITHUB_REF_TYPE:-}" = branch ] && [ "${GITHUB_REF_NAME:-}" = main ]; then
  image_tags="dev-${GITHUB_RUN_ID:?},dev"
elif [ "${GITHUB_REF_TYPE:-}" = branch ] && [ "${GITHUB_REF_PROTECTED:-false}" = true ]; then
  branch_slug=$(printf '%s' "${GITHUB_REF_NAME:?}" | LC_ALL=C tr '[:upper:]' '[:lower:]' | sed 's/[^a-z0-9]/-/g; s/^-*//; s/-*$//' | cut -c1-63 | sed 's/-*$//')
  branch_slug=${branch_slug:-branch}

  image_tags="dev-${GITHUB_RUN_ID:?},dev-$branch_slug"
else
  exit 0
fi

image=${IMAGE_REPOSITORY:-ghcr.io/${GITHUB_REPOSITORY:?}}
image=$(printf '%s' "$image" | tr '[:upper:]' '[:lower:]')

printf '%s:%s,%s:%s\n' "$image" "${image_tags%,*}" "$image" "${image_tags#*,}"
