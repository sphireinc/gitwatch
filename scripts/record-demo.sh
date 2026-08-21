#!/bin/sh
set -eu

binary=${1:?usage: record-demo.sh /path/to/gitwatch /path/to/demo-repository}
repository=${2:?usage: record-demo.sh /path/to/gitwatch /path/to/demo-repository}

if [ ! -x "$binary" ]; then
	echo "gitwatch binary is not executable: $binary" >&2
	exit 2
fi
if [ ! -d "$repository/.git" ]; then
	echo "demo repository is not initialized: $repository" >&2
	exit 2
fi

binary=$(cd "$(dirname "$binary")" && pwd -P)/$(basename "$binary")
repository=$(cd "$repository" && pwd -P)
socket="${TMPDIR:-/tmp}/gitwatch-demo-tmux-$$.sock"
session="gitwatch-demo-$$"

cleanup() {
	tmux -S "$socket" kill-server >/dev/null 2>&1 || true
}
trap cleanup EXIT HUP INT TERM

tmux -S "$socket" new-session -d -s "$session" -x 160 -y 30 -c "$repository" \
	"GITWATCH_WATCH=fs GITWATCH_MOTION=off '$binary'"
tmux -S "$socket" set-option -t "$session" status off

(
	sleep 1
	printf '%s\n' 'Live watcher refresh arrived without pressing r.' >>"$repository/docs/notes.md"
	sleep 1
	# SGR left-button press/release at one-based terminal column 10, row 4.
	tmux -S "$socket" send-keys -t "$session" -H \
		1b 5b 3c 30 3b 31 30 3b 34 4d 1b 5b 3c 30 3b 31 30 3b 34 6d
	sleep 1
	tmux -S "$socket" send-keys -t "$session" Escape
	sleep 0.2
	tmux -S "$socket" send-keys -t "$session" -l '/notes'
	sleep 0.2
	tmux -S "$socket" send-keys -t "$session" Enter
	sleep 1
	tmux -S "$socket" send-keys -t "$session" -l '/'
	sleep 0.2
	tmux -S "$socket" send-keys -t "$session" Escape
	sleep 0.2
	tmux -S "$socket" send-keys -t "$session" Space
	sleep 1
	tmux -S "$socket" send-keys -t "$session" Space
	sleep 1
	tmux -S "$socket" send-keys -t "$session" d
	sleep 1
	tmux -S "$socket" resize-window -t "$session" -x 80 -y 24
	sleep 1
	tmux -S "$socket" send-keys -t "$session" q
) >/dev/null 2>&1 &

tmux -S "$socket" attach-session -t "$session"
