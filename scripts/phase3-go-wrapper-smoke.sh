#!/usr/bin/env bash
set -euo pipefail

workflow_file="phase3-go-wrapper-smoke.yml"
required_jobs=(
  "linux-amd64-go-race"
  "linux-arm64-go-race"
  "darwin-amd64-go-race"
  "darwin-arm64-go-race"
  "windows-amd64-go-race"
)

branch="$(git rev-parse --abbrev-ref HEAD)"
dispatched_after="$(date -u +"%Y-%m-%dT%H:%M:%SZ")"

gh auth status -h github.com >/dev/null
git remote get-url origin >/dev/null

git push origin "${branch}"
gh workflow run "${workflow_file}" --ref "${branch}"

run_id=""
for attempt in $(seq 1 30); do
  run_id="$(
    gh run list \
      --workflow "${workflow_file}" \
      --branch "${branch}" \
      --event workflow_dispatch \
      --json databaseId,createdAt \
      --limit 20 \
    | python3 - "${dispatched_after}" <<'PY'
import json
import sys
from datetime import datetime, timezone

created_after = datetime.fromisoformat(sys.argv[1].replace("Z", "+00:00"))
runs = json.load(sys.stdin)

def parse_created(value: str) -> datetime:
    return datetime.fromisoformat(value.replace("Z", "+00:00")).astimezone(timezone.utc)

matching = [run for run in runs if parse_created(run["createdAt"]) >= created_after]
matching.sort(key=lambda run: parse_created(run["createdAt"]), reverse=True)
if matching:
    print(matching[0]["databaseId"])
PY
  )"

  if [[ -n "${run_id}" ]]; then
    break
  fi

  sleep 2
done

if [[ -z "${run_id}" ]]; then
  echo "failed to locate workflow_dispatch run id for ${branch} after ${dispatched_after}" >&2
  exit 1
fi

gh run watch "${run_id}" || true

gh run view "${run_id}" --json conclusion,jobs \
| python3 - "${run_id}" "${required_jobs[@]}" <<'PY'
import json
import sys

run_id = sys.argv[1]
required_jobs = sys.argv[2:]
payload = json.load(sys.stdin)
jobs = {job["name"]: job.get("conclusion") for job in payload.get("jobs", [])}

missing = [name for name in required_jobs if name not in jobs]
failed = [name for name in required_jobs if jobs.get(name) != "success"]

if payload.get("conclusion") != "success" or missing or failed:
    if missing:
        print(f"run {run_id} missing jobs: {', '.join(missing)}", file=sys.stderr)
    if failed:
        details = ", ".join(f"{name}={jobs.get(name)!r}" for name in failed)
        print(f"run {run_id} required jobs not successful: {details}", file=sys.stderr)
    sys.exit(1)

print(f"run {run_id} verified")
PY
