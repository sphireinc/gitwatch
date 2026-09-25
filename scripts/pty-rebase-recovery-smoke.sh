#!/bin/sh
set -eu

binary=${1:?usage: pty-rebase-recovery-smoke.sh /path/to/gitwatch [evidence-dir]}
evidence=${2:-}
test -x "$binary"
command -v tmux >/dev/null

root=$(mktemp -d "${TMPDIR:-/tmp}/gitwatch-rebase-pty.XXXXXX")
repo="$root/repository"
socket="/tmp/gitwatch-rebase-pty-$$.sock"
session="gitwatch-rebase-pty-$$"
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
printf '%s\n' feature >"$repo/shared.txt"
git -C "$repo" add -- shared.txt
git -C "$repo" commit --quiet -m feature
git -C "$repo" switch --quiet main
printf '%s\n' main >"$repo/shared.txt"
git -C "$repo" add -- shared.txt
git -C "$repo" commit --quiet -m main
main_head=$(git -C "$repo" rev-parse HEAD)
git -C "$repo" switch --quiet feature
if git -C "$repo" rebase main >/dev/null 2>&1; then
	echo 'fixture did not produce a rebase conflict' >&2
	exit 1
fi
test -d "$repo/.git/rebase-merge"

binary=$(cd "$(dirname "$binary")" && pwd -P)/$(basename "$binary")
repo=$(cd "$repo" && pwd -P)
[ -z "$evidence" ] || mkdir -p "$evidence"
tmux -S "$socket" new-session -d -s "$session" -x 80 -y 24 -c "$repo" \
	"NO_COLOR=1 GITWATCH_WATCH=poll GITWATCH_MOTION=off '$binary' --watch poll --motion off"

ready=0
i=0
while [ "$i" -lt 80 ]; do
	tmux -S "$socket" capture-pane -p -t "$session" >"$capture" 2>/dev/null || true
	if grep -q 'REBASE' "$capture"; then ready=1; break; fi
	i=$((i + 1)); sleep 0.1
done
if [ "$ready" -ne 1 ]; then cat "$capture" >&2; exit 1; fi
tmux -S "$socket" send-keys -t "$session" C

recovery_ready=0
i=0
while [ "$i" -lt 80 ]; do
	tmux -S "$socket" capture-pane -p -t "$session" >"$capture"
	if grep -q 'Rebase recovery' "$capture" && grep -q 'Progress:' "$capture" && grep -q 'Current commit:' "$capture"; then recovery_ready=1; break; fi
	i=$((i + 1)); sleep 0.1
done
if [ "$recovery_ready" -ne 1 ]; then cat "$capture" >&2; exit 1; fi
if LC_ALL=C grep -Eq '\033\[' "$capture"; then
	echo 'NO_COLOR recovery pane contained terminal controls' >&2
	exit 1
fi
if [ -n "$evidence" ]; then
	cp "$capture" "$evidence/rebase-recovery-80x24-pane.txt"
	printf '%s\n' 'NO_COLOR=1; motion=off; dimensions=80x24; externally started conflict rebase' >"$evidence/rebase-recovery-environment.txt"
fi

tmux -S "$socket" send-keys -t "$session" s
completed=0
i=0
while [ "$i" -lt 80 ]; do
	if [ ! -d "$repo/.git/rebase-merge" ]; then completed=1; break; fi
	i=$((i + 1)); sleep 0.1
done
if [ "$completed" -ne 1 ] || [ "$(git -C "$repo" rev-parse HEAD)" != "$main_head" ] || [ "$(git -C "$repo" branch --show-current)" != feature ]; then
	cat "$capture" >&2
	echo 'rebase skip did not complete to the expected branch and HEAD' >&2
	exit 1
fi
tmux -S "$socket" send-keys -t "$session" q
i=0
while [ "$i" -lt 40 ] && tmux -S "$socket" has-session -t "$session" 2>/dev/null; do i=$((i + 1)); sleep 0.1; done
if tmux -S "$socket" has-session -t "$session" 2>/dev/null; then
	# The first q can return from the recovery workspace to Status.
	tmux -S "$socket" send-keys -t "$session" q
	i=0
	while [ "$i" -lt 40 ] && tmux -S "$socket" has-session -t "$session" 2>/dev/null; do i=$((i + 1)); sleep 0.1; done
	if tmux -S "$socket" has-session -t "$session" 2>/dev/null; then
		tmux -S "$socket" capture-pane -p -t "$session" >"$capture"
		cat "$capture" >&2
		echo 'gitwatch did not exit after rebase recovery' >&2
		exit 1
	fi
fi
echo 'rebase PTY recovery passed: external conflict, 80x24 NO_COLOR progress, skip, authoritative HEAD, and clean quit'
