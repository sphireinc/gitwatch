#!/bin/sh
set -eu

binary=${1:?usage: pty-cherry-pick-progress-smoke.sh /path/to/gitwatch [evidence-dir]}
evidence=${2:-}
test -x "$binary"
command -v tmux >/dev/null

root=$(mktemp -d "${TMPDIR:-/tmp}/gitwatch-cherry-pty.XXXXXX")
repo="$root/repository"
socket="/tmp/gitwatch-cherry-pty-$$.sock"
session="gitwatch-cherry-pty-$$"
capture="$root/pane.txt"
cleanup() {
	tmux -S "$socket" kill-session -t "$session" >/dev/null 2>&1 || true
	rm -f "$socket"
	rm -rf "$root"
}
trap cleanup EXIT HUP INT TERM

mkdir -p "$repo"
git -C "$repo" init --quiet
git -C "$repo" symbolic-ref HEAD refs/heads/main
git -C "$repo" config user.name gitwatch-fixture
git -C "$repo" config user.email fixture@example.invalid
git -C "$repo" config commit.gpgsign false
printf '%s\n' base >"$repo/shared.txt"
git -C "$repo" add -- shared.txt
git -C "$repo" commit --quiet -m base
git -C "$repo" switch --quiet -c feature
printf '%s\n' first >"$repo/first.txt"
git -C "$repo" add -- first.txt
git -C "$repo" commit --quiet -m first
first=$(git -C "$repo" rev-parse HEAD)
printf '%s\n' feature >"$repo/shared.txt"
git -C "$repo" add -- shared.txt
git -C "$repo" commit --quiet -m second
second=$(git -C "$repo" rev-parse HEAD)
printf '%s\n' third >"$repo/third.txt"
git -C "$repo" add -- third.txt
git -C "$repo" commit --quiet -m third
third=$(git -C "$repo" rev-parse HEAD)
git -C "$repo" switch --quiet main
printf '%s\n' main >"$repo/shared.txt"
git -C "$repo" add -- shared.txt
git -C "$repo" commit --quiet -m main
original=$(git -C "$repo" rev-parse HEAD)
if git -C "$repo" cherry-pick "$first" "$second" "$third" >/dev/null 2>&1; then
	echo 'fixture did not produce a middle cherry-pick conflict' >&2
	exit 1
fi
test -f "$repo/.git/CHERRY_PICK_HEAD"

binary=$(cd "$(dirname "$binary")" && pwd -P)/$(basename "$binary")
repo=$(cd "$repo" && pwd -P)
[ -z "$evidence" ] || mkdir -p "$evidence"
tmux -S "$socket" new-session -d -s "$session" -x 80 -y 24 -c "$repo" \
	"NO_COLOR=1 GITWATCH_WATCH=poll GITWATCH_MOTION=off '$binary' --watch poll --motion off"

ready=0
i=0
while [ "$i" -lt 80 ]; do
	tmux -S "$socket" capture-pane -p -t "$session" >"$capture" 2>/dev/null || true
	if grep -q 'CHERRY-PICK' "$capture"; then ready=1; break; fi
	i=$((i + 1)); sleep 0.1
done
if [ "$ready" -ne 1 ]; then cat "$capture" >&2; exit 1; fi
tmux -S "$socket" send-keys -t "$session" C

progress_ready=0
i=0
while [ "$i" -lt 80 ]; do
	tmux -S "$socket" capture-pane -p -t "$session" >"$capture"
	if grep -q 'Cherry-pick progress' "$capture" && grep -q 'Target: main' "$capture" && grep -q '1 completed' "$capture" && grep -q 'conflicted' "$capture" && grep -q '\[s\] skip' "$capture"; then progress_ready=1; break; fi
	i=$((i + 1)); sleep 0.1
done
if [ "$progress_ready" -ne 1 ]; then
	cat "$capture" >&2
	for name in head todo todo.backup done abort-safety; do
		file="$repo/.git/sequencer/$name"
		if [ -f "$file" ]; then
			printf '%s\n' "sequencer/$name:" >&2
			sed -n '1,12p' "$file" >&2
		fi
	done
	exit 1
fi
if [ -n "$evidence" ]; then
	cp "$capture" "$evidence/cherry-pick-80x24-pane.txt"
	printf '%s\n' 'NO_COLOR=1; motion=off; dimensions=80x24; externally started three-commit pick, middle conflict' >"$evidence/cherry-pick-environment.txt"
fi

tmux -S "$socket" send-keys -t "$session" 1
sleep 0.2
tmux -S "$socket" send-keys -t "$session" C
i=0
reopened=0
while [ "$i" -lt 80 ]; do
	tmux -S "$socket" capture-pane -p -t "$session" >"$capture"
	if grep -q 'Cherry-pick progress' "$capture" && grep -q '\[s\] skip' "$capture"; then reopened=1; break; fi
	i=$((i + 1)); sleep 0.1
done
if [ "$reopened" -ne 1 ]; then cat "$capture" >&2; exit 1; fi
tmux -S "$socket" send-keys -t "$session" s
i=0
completed=0
while [ "$i" -lt 80 ]; do
	if [ ! -f "$repo/.git/CHERRY_PICK_HEAD" ] && [ ! -d "$repo/.git/sequencer" ]; then completed=1; break; fi
	i=$((i + 1)); sleep 0.1
done
result=$(git -C "$repo" rev-parse HEAD)
if [ "$completed" -ne 1 ] || [ "$result" = "$original" ] || [ "$(git -C "$repo" branch --show-current)" != main ] || [ ! -f "$repo/first.txt" ] || [ ! -f "$repo/third.txt" ] || [ "$(sed -n '1p' "$repo/shared.txt")" != main ]; then
	cat "$capture" >&2
	echo 'cherry-pick skip did not complete the ordered selection' >&2
	exit 1
fi
tmux -S "$socket" send-keys -t "$session" q
i=0
while [ "$i" -lt 40 ] && tmux -S "$socket" has-session -t "$session" 2>/dev/null; do i=$((i + 1)); sleep 0.1; done
if tmux -S "$socket" has-session -t "$session" 2>/dev/null; then
	tmux -S "$socket" send-keys -t "$session" q
	i=0
	while [ "$i" -lt 40 ] && tmux -S "$socket" has-session -t "$session" 2>/dev/null; do i=$((i + 1)); sleep 0.1; done
	if tmux -S "$socket" has-session -t "$session" 2>/dev/null; then cat "$capture" >&2; exit 1; fi
fi
echo 'cherry-pick PTY progress passed: middle conflict, 80x24, navigation/reopen, skip, result HEAD, and clean quit'
