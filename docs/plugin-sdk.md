# Plugin SDK

Third-party plugins should import `github.com/jusanchez/gitwatch/pkg/plugin`
only. The package contains the versioned manifest and newline-delimited JSON
message contract; it has no dependency on gitwatch internals.

Plugins run as separate processes and communicate through stdin/stdout. They
must treat all host payloads as untrusted and must not assume that a message
will be followed by another message after a failure.
