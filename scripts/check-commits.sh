#!/usr/bin/env bash
# Usage: scripts/check-commits.sh BASE HEAD
# Fails if any commit in BASE..HEAD has a subject that isn't a Conventional
# Commit, or is a fixup/squash/WIP commit. PRs are rebase-merged, so every
# commit lands on main as written.
set -euo pipefail
base=${1:?base}; head=${2:?head}
types='feat|fix|docs|style|refactor|perf|test|build|ci|chore|revert'
pattern="^($types)(\([a-z0-9./_-]+\))?!?: [^ ].{0,99}$"
bad=0
while IFS=$'\t' read -r sha subject; do
  if [[ ! $subject =~ $pattern ]] || [[ $subject =~ ^(fixup!|squash!|amend!|WIP) ]]; then
    echo "::error::${sha:0:7} \"$subject\" is not a Conventional Commit subject (type(scope): summary, at most 100 characters)"
    bad=1
  else
    echo "ok ${sha:0:7} $subject"
  fi
done < <(git log --format='%H%x09%s' --no-merges "$base..$head")
exit $bad
