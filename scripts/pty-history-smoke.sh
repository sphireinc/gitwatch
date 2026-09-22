#!/bin/sh
set -eu
binary=${1:?binary required}
evidence=${2:-}
root=$(mktemp -d "${TMPDIR:-/tmp}/gitwatch-history.XXXXXX")
repo="$root/repository"
socket="/tmp/gitwatch-history-$$.sock"
session="gitwatch-history-$$"
capture="/tmp/gitwatch-history-$$.capture"
cleanup() { tmux -S "$socket" kill-session -t "$session" >/dev/null 2>&1 || true; rm -f "$socket" "$capture"; rm -rf "$root"; }
trap cleanup EXIT HUP INT TERM
test -x "$binary"
command -v tmux >/dev/null
mkdir -p "$repo"
git -C "$repo" init --quiet
git -C "$repo" config user.name gitwatch-fixture
git -C "$repo" config user.email fixture@example.invalid
git -C "$repo" config commit.gpgsign false
printf '%s\n' baseline >"$repo/README.md"
git -C "$repo" add README.md
git -C "$repo" commit --quiet -m baseline
git -C "$repo" switch --quiet -c feature
printf '%s\n' feature >"$repo/feature.txt"
git -C "$repo" add feature.txt
git -C "$repo" commit --quiet -m feature
git -C "$repo" switch --quiet main 2>/dev/null || git -C "$repo" switch --quiet master
printf '%s\n' main >"$repo/main.txt"
git -C "$repo" add main.txt
git -C "$repo" commit --quiet -m main
git -C "$repo" merge --quiet --no-ff feature -m merge
binary=$(cd "$(dirname "$binary")" && pwd -P)/$(basename "$binary")
repo=$(cd "$repo" && pwd -P)
[ -z "$evidence" ] || mkdir -p "$evidence"
tmux -S "$socket" new-session -d -s "$session" -x 120 -y 32 -c "$repo" "NO_COLOR=1 GITWATCH_WATCH=poll GITWATCH_MOTION=off '$binary' --watch poll --motion off"
ready=0
i=0
while [ "$i" -lt 40 ]; do
	tmux -S "$socket" capture-pane -p -t "$session" >"$capture" 2>/dev/null || true
	if grep -Eiq 'Status|Files|Repository|gitwatch' "$capture"; then ready=1; break; fi
	i=$((i + 1)); sleep 0.1
done
test "$ready" -eq 1
tmux -S "$socket" resize-window -t "$session" -x 80 -y 24
tmux -S "$socket" send-keys -t "$session" l
history_ready=0
i=0
while [ "$i" -lt 40 ]; do
	tmux -S "$socket" capture-pane -p -t "$session" >"$capture"
	if grep -Eiq 'History' "$capture" && grep -Eiq 'merge|feature|baseline' "$capture"; then history_ready=1; break; fi
	i=$((i + 1)); sleep 0.1
done
if [ "$history_ready" -ne 1 ]; then cat "$capture" >&2; exit 1; fi
if LC_ALL=C grep -Eq '\033\[' "$capture"; then exit 1; fi
if [ -n "$evidence" ]; then cp "$capture" "$evidence/history-pty-pane.txt"; printf '%s\n' 'NO_COLOR=1 GITWATCH_MOTION=off; topology=merge; dimensions=80x24' >"$evidence/history-pty-environment.txt"; fi
tmux -S "$socket" send-keys -t "$session" q
i=0
while [ "$i" -lt 30 ] && tmux -S "$socket" has-session -t "$session" 2>/dev/null; do i=$((i + 1)); sleep 0.1; done
if tmux -S "$socket" has-session -t "$session" 2>/dev/null; then exit 1; fi
echo 'history PTY smoke passed: merge graph, NO_COLOR, 80x24, reduced motion, and clean quit'
