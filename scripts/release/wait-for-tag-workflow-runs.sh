#!/usr/bin/env bash
set -euo pipefail

if [ "$#" -lt 3 ]; then
  echo "usage: $0 <tag> <head-sha> <workflow> [<workflow> ...]" >&2
  exit 1
fi

tag_name="$1"
head_sha="$2"
shift 2

poll_interval="${POLL_INTERVAL_SECONDS:-3}"
lookup_timeout="${WAIT_FOR_RUN_TIMEOUT_SECONDS:-180}"
lookup_deadline="$(($(date +%s) + lookup_timeout))"
api_retry_limit="${API_RETRY_LIMIT:-10}"
api_retry_delay="${API_RETRY_DELAY_SECONDS:-15}"

# Transient GitHub API errors (for example HTTP 502) must not be mistaken for a
# missing or failed run, so lookups report failure separately from "not found".
find_run_id() {
  local workflow="$1"
  local runs

  if ! runs="$(
    gh run list \
      --workflow "${workflow}" \
      --limit 100 \
      --json databaseId,event,headBranch,headSha
  )"; then
    echo "gh run list failed for ${workflow}; retrying" >&2
    return 0
  fi

  jq -r \
    --arg tag_name "${tag_name}" \
    --arg head_sha "${head_sha}" \
    '.[] | select(.event == "push" and .headBranch == $tag_name and .headSha == $head_sha) | .databaseId' \
    <<<"${runs}" \
    | head -n1
}

# gh run watch exits non-zero both when the run fails and when the API call
# behind it errors, so confirm the outcome from the run itself before failing.
wait_for_run() {
  local workflow="$1"
  local run_id="$2"
  local api_failures=0
  local state status conclusion

  while true; do
    if gh run watch "${run_id}" --exit-status; then
      return 0
    fi

    if state="$(gh run view "${run_id}" --json status,conclusion --jq '"\(.status) \(.conclusion)"')"; then
      read -r status conclusion <<<"${state}"
      if [ "${status}" = "completed" ]; then
        if [ "${conclusion}" = "success" ]; then
          return 0
        fi
        echo "${workflow} run ${run_id} for ${tag_name} concluded ${conclusion}" >&2
        return 1
      fi
      echo "${workflow} run ${run_id} is still ${status}; resuming watch" >&2
    else
      echo "gh run view failed for ${workflow} run ${run_id}" >&2
    fi

    api_failures="$((api_failures + 1))"
    if [ "${api_failures}" -ge "${api_retry_limit}" ]; then
      echo "giving up on ${workflow} run ${run_id} after ${api_failures} interrupted watches" >&2
      return 1
    fi

    sleep "${api_retry_delay}"
  done
}

for workflow in "$@"; do
  run_id=""

  while [ -z "${run_id}" ]; do
    run_id="$(find_run_id "${workflow}")"
    if [ -n "${run_id}" ]; then
      break
    fi

    if [ "$(date +%s)" -ge "${lookup_deadline}" ]; then
      echo "timed out waiting for ${workflow} on tag ${tag_name} (${head_sha})" >&2
      exit 1
    fi

    sleep "${poll_interval}"
  done

  echo "watching ${workflow} run ${run_id} for ${tag_name}" >&2
  wait_for_run "${workflow}" "${run_id}"
done
