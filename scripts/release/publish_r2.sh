#!/usr/bin/env bash
set -euo pipefail

usage() {
  cat <<'EOF'
usage: publish_r2.sh \
  --version <tag-with-v> \
  --staged-root <dir>

Required environment variables:
  R2_BUCKET
  AWS_ACCESS_KEY_ID
  AWS_SECRET_ACCESS_KEY

One of:
  R2_ENDPOINT_URL
  R2_ENDPOINT

Optional environment variables:
  AWS_SESSION_TOKEN
  AWS_DEFAULT_REGION (defaults to auto)
EOF
}

version=""
staged_root=""

while [[ $# -gt 0 ]]; do
  case "$1" in
    --version)
      version="${2:-}"
      shift 2
      ;;
    --staged-root)
      staged_root="${2:-}"
      shift 2
      ;;
    -h|--help)
      usage
      exit 0
      ;;
    *)
      echo "unknown argument: $1" >&2
      usage >&2
      exit 1
      ;;
  esac
done

if [[ -z "$version" || -z "$staged_root" ]]; then
  usage >&2
  exit 1
fi

if [[ ! "$version" =~ ^v[0-9]+(\.[0-9]+){2}([-.][0-9A-Za-z.-]+)?$ ]]; then
  echo "expected --version to include a leading v, got: $version" >&2
  exit 1
fi

: "${R2_BUCKET:?R2_BUCKET is required}"
: "${AWS_ACCESS_KEY_ID:?AWS_ACCESS_KEY_ID is required}"
: "${AWS_SECRET_ACCESS_KEY:?AWS_SECRET_ACCESS_KEY is required}"
export AWS_DEFAULT_REGION="${AWS_DEFAULT_REGION:-auto}"

endpoint_url="${R2_ENDPOINT_URL:-${R2_ENDPOINT:-}}"
: "${endpoint_url:?R2_ENDPOINT_URL or R2_ENDPOINT is required}"

publish_root="${staged_root%/}/${version}"
if [[ ! -d "$publish_root" ]]; then
  echo "staged release root not found: $publish_root" >&2
  exit 1
fi

prefix="pure-simdjson/${version}"
existing_count="$(
  aws --endpoint-url "$endpoint_url" s3api list-objects-v2 \
    --bucket "$R2_BUCKET" \
    --prefix "${prefix}/" \
    --max-keys 1 \
    --query 'KeyCount' \
    --output text
)"

if [[ "$existing_count" != "0" && "$existing_count" != "None" ]]; then
  echo "Refusing to overwrite immutable release prefix: ${prefix}/" >&2
  exit 1
fi

aws --endpoint-url "$endpoint_url" s3 cp --recursive \
  "$publish_root" \
  "s3://${R2_BUCKET}/${prefix}/"

echo "Uploaded immutable release payload to s3://${R2_BUCKET}/${prefix}/"
