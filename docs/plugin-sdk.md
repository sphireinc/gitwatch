# Plugin SDK

Third-party plugins should import `github.com/sphireinc/git-watch/pkg/plugin`
only. The package contains the versioned manifest and newline-delimited JSON
message contract; it has no dependency on gitwatch internals.

## Compatibility fixtures

API version 1 has checked-in request, response, and status-widget fixtures in
`pkg/plugin/testdata/v1/`. `TestV1WireFixturesRemainDecodable` loads every
fixture, decodes it through the public SDK, and validates handshake API
versions. The v1 compatibility matrix is API-1 hosts and API-1 plugins using
the documented message types. A future API version must add a new fixture
directory and an explicit negotiation test before it is considered compatible.

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

SDK helpers such as `NewHandshake`, `NewCommand`, `NewPanel`, `NewWidget`, and
`NewEvent` construct API-1 messages without importing Bubble Tea or internal
packages. Use `Encode` for newline-delimited transport and `Decode` for every
incoming record. Errors should use bounded `WireError` codes/messages and must
not include tokens, full remote URLs, or repository file contents.
