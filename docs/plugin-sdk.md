# Plugin SDK

Third-party plugins should import `github.com/jusanchez/gitwatch/pkg/plugin`
only. The package contains the versioned manifest and newline-delimited JSON
message contract; it has no dependency on gitwatch internals.

Plugins run as separate processes and communicate through stdin/stdout. They
must treat all host payloads as untrusted and must not assume that a message
will be followed by another message after a failure.

Buildable examples live under `examples/plugin-command`,
`examples/plugin-panel`, and `examples/plugin-widget`. Each example imports
only `pkg/plugin`, has a matching API-1 manifest, and demonstrates one
extension capability. Build them with, for example:

```sh
go build ./examples/plugin-command
go build ./examples/plugin-panel
go build ./examples/plugin-widget
```

The examples intentionally echo protocol payloads rather than invoking Git or
accessing gitwatch internals. Compatibility tests in `pkg/plugin` enforce the
wire shape, API version, capability validation, and bounded message fields.
