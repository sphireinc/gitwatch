#!/bin/sh
set -eu

binary=${1:?usage: pty-smoke.sh /path/to/gitwatch /path/to/repository [evidence-dir]}
repository=${2:?usage: pty-smoke.sh /path/to/gitwatch /path/to/repository [evidence-dir]}
evidence=${3:-}

if [ ! -x "$binary" ]; then
	echo "gitwatch binary is not executable: $binary" >&2
	exit 2
fi
if [ ! -d "$repository/.git" ]; then
	echo "repository is not initialized: $repository" >&2
	exit 2
fi
if ! command -v tmux >/dev/null 2>&1; then
	echo "tmux is required for the PTY smoke gate" >&2
	exit 2
fi

binary=$(cd "$(dirname "$binary")" && pwd -P)/$(basename "$binary")
repository=$(cd "$repository" && pwd -P)
if [ -n "$evidence" ]; then
	mkdir -p "$evidence"
fi

# Keep the tmux control socket in the repository's supported temporary root.
# macOS TMPDIR may point at a sandboxed per-process directory that tmux cannot
# create sockets in when the smoke gate is launched by a managed runner.
socket="/tmp/gitwatch-pty-$$.sock"
session="gitwatch-pty-$$"
capture="/tmp/gitwatch-pty-$$.capture"

cleanup() {
	tmux -S "$socket" kill-session -t "$session" >/dev/null 2>&1 || true
	rm -f "$socket" "$capture"
}
trap cleanup EXIT HUP INT TERM

tmux -S "$socket" new-session -d -s "$session" -x 120 -y 32 -c "$repository" \
	"GITWATCH_WATCH=poll GITWATCH_MOTION=off '$binary' --watch poll --motion off"

ready=0
i=0
while [ "$i" -lt 40 ]; do
	if tmux -S "$socket" capture-pane -p -t "$session" >"$capture" 2>/dev/null; then
		if grep -Eq 'Status|Files|Repository|gitwatch' "$capture"; then
			ready=1
			break
		fi
	fi
	i=$((i + 1))
	sleep 0.1
done
if [ "$ready" -ne 1 ]; then
	echo "PTY smoke failed: application did not render a workspace" >&2
	cat "$capture" >&2 2>/dev/null || true
	exit 1
fi

tmux -S "$socket" resize-window -t "$session" -x 80 -y 24
tmux -S "$socket" send-keys -t "$session" '?'
sleep 0.3
tmux -S "$socket" capture-pane -p -t "$session" >"$capture"
if ! grep -Eiq 'help|keyboard|quit|escape' "$capture"; then
	echo "PTY smoke failed: help view did not render at 80x24" >&2
	cat "$capture" >&2
	exit 1
fi

tmux -S "$socket" send-keys -t "$session" Escape
sleep 0.3
tmux -S "$socket" send-keys -t "$session" q
i=0
while [ "$i" -lt 30 ] && tmux -S "$socket" has-session -t "$session" 2>/dev/null; do
	i=$((i + 1))
	sleep 0.1
done
if tmux -S "$socket" has-session -t "$session" 2>/dev/null; then
	echo "PTY smoke failed: application did not exit after q" >&2
	exit 1
fi

if [ -n "$evidence" ]; then
	cp "$capture" "$evidence/pty-pane.txt"
	{
		date -u '+%Y-%m-%dT%H:%M:%SZ'
		uname -srm 2>/dev/null || true
		"$binary" --version
	} >"$evidence/pty-environment.txt"
fi
echo "PTY smoke passed: startup, 80x24 help, and clean quit"
