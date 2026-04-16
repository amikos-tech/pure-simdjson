#!/usr/bin/env bash
set -euo pipefail

workflow_name="phase3-go-wrapper-smoke"
required_jobs=(
  "linux-amd64-go-race"
  "linux-arm64-go-race"
  "darwin-amd64-go-race"
  "darwin-arm64-go-race"
  "windows-amd64-go-race"
)

branch="$(git rev-parse --abbrev-ref HEAD)"
pushed_after="$(date -u +"%Y-%m-%dT%H:%M:%SZ")"

gh auth status -h github.com >/dev/null
git remote get-url origin >/dev/null

git push origin "${branch}"

run_id=""
for attempt in $(seq 1 30); do
  run_id="$(
    gh run list \
      --branch "${branch}" \
      --event push \
      --json databaseId,createdAt,workflowName \
      --limit 20 \
    | python3 -c '
import json
import sys
from datetime import datetime, timezone

created_after = datetime.fromisoformat(sys.argv[1].replace("Z", "+00:00"))
workflow_name = sys.argv[2]
raw = sys.stdin.read().strip()
if not raw:
    sys.exit(0)
runs = json.loads(raw)

def parse_created(value: str) -> datetime:
    return datetime.fromisoformat(value.replace("Z", "+00:00")).astimezone(timezone.utc)

matching = [
    run
    for run in runs
    if run.get("workflowName") == workflow_name and parse_created(run["createdAt"]) >= created_after
]
matching.sort(key=lambda run: parse_created(run["createdAt"]), reverse=True)
if matching:
    print(matching[0]["databaseId"])
' "${pushed_after}" "${workflow_name}"
  )"

  if [[ -n "${run_id}" ]]; then
    break
  fi

  sleep 2
done

if [[ -z "${run_id}" ]]; then
  echo "failed to locate push run id for ${branch} after ${pushed_after}" >&2
  exit 1
fi

run_status=""
for attempt in $(seq 1 180); do
  run_status="$(gh run view "${run_id}" --json status --jq .status)"
  if [[ "${run_status}" == "completed" ]]; then
    break
  fi
  sleep 5
done

if [[ "${run_status}" != "completed" ]]; then
  echo "timed out waiting for run ${run_id} to complete" >&2
  exit 1
fi

gh run view "${run_id}" --json conclusion,jobs \
| python3 -c '
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
        details = ", ".join(missing)
        print(f"run {run_id} missing jobs: {details}", file=sys.stderr)
    if failed:
        details = ", ".join(f"{name}={jobs.get(name)!r}" for name in failed)
        print(f"run {run_id} required jobs not successful: {details}", file=sys.stderr)
    sys.exit(1)

print(f"run {run_id} verified")
' "${run_id}" "${required_jobs[@]}"
