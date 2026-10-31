#!/bin/sh
set -eu
cd "$(dirname "$0")/.."

GITHUB_REPOSITORY=wyndure/octelium-initializer
GITHUB_RUN_ID=1
GITHUB_EVENT_NAME=push

export GITHUB_REPOSITORY GITHUB_RUN_ID GITHUB_EVENT_NAME

unset IMAGE_REPOSITORY

image=ghcr.io/wyndure/octelium-initializer

check() {
  expected=$1
  GITHUB_REF_TYPE=$2
  GITHUB_REF_NAME=$3
  GITHUB_REF_PROTECTED=$4
  export GITHUB_REF_TYPE GITHUB_REF_NAME GITHUB_REF_PROTECTED
  actual=$(sh ci/image-tags.sh)
  if [ "$actual" != "$expected" ]; then
    printf 'Expected %s, got %s\n' "$expected" "$actual" >&2
    exit 1
  fi
}

check "$image:v1.2.3,$image:latest" tag v1.2.3 true
check "$image:v1.2.3-rc.1,$image:latest" tag v1.2.3-rc.1 false
check "$image:dev-1,$image:dev" branch main true
check "$image:dev-1,$image:dev" branch main false
check "$image:dev-1,$image:dev-release-1-2" branch release/1.2 true
check "$image:dev-1,$image:dev-release-next" branch Release/Next true
check '' branch feature/topic false

GITHUB_EVENT_NAME=pull_request

check '' branch main true
check '' tag v1.2.3 true

GITHUB_EVENT_NAME=push
GITHUB_REF_TYPE=tag
GITHUB_REF_NAME=invalid/tag

export GITHUB_EVENT_NAME GITHUB_REF_TYPE GITHUB_REF_NAME

if sh ci/image-tags.sh >/dev/null 2>&1; then
  printf 'Invalid Docker tag was accepted\n' >&2
  exit 1
fi

printf 'Image tag policy tests passed\n'
