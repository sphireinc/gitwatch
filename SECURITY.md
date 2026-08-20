# Security policy

Please report suspected command injection, terminal escape injection, or data-loss bugs privately to the repository maintainers rather than opening a public exploit description. Include the version, operating system, Git version, reproduction steps, and sanitized logs.

gitwatch never invokes a shell for Git operations, treats watcher events as hints, sanitizes untrusted terminal text, and requires explicit confirmation for destructive restore actions. Do not include credentials or private repository contents in reports.
