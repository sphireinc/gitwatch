# Plugin contract

Plugins are separate processes. A plugin declares a JSON manifest with an
identifier, version, API version, executable, capabilities, and optional
configuration schema. gitwatch validates the manifest and negotiates only
capabilities supported by the host. The public SDK also defines stable event,
command, panel, status-widget, and lifecycle payload types.

The v1 contract does not permit a plugin to execute Git directly inside the
gitwatch process or mutate TUI state. Startup begins with a newline-delimited
`handshake` message containing the API version and the capabilities granted by
the host; the plugin must return a matching `handshake` response. A host
rejects an unsupported API version, unknown capability, or capability that was
not granted. Lifecycle events are `start`, `stop`, and `failure`; message
payloads are opaque JSON and are bounded by the SDK limits. The runtime owns
transport, timeout and cancellation handling, bounded restart supervision,
crash containment, and capability validation.

Compatibility guarantee: API version 1 is additive within the declared JSON
fields. Hosts ignore optional fields they do not use, while plugins must not
depend on undeclared capabilities or private gitwatch packages. A future
breaking wire change requires a new API version.

API-1 policy: new optional fields and new message payload fields are additive;
existing meanings and limits remain stable. A breaking envelope, capability
meaning, or required-field change requires API 2 plus migration fixtures. The
host grants capabilities explicitly, limits each process output and runtime,
and never lets a plugin write terminal control sequences directly. Plugin
state is opt-in, private, and can be reset by deleting the state file.

## Versioned contributions

The dependency-free SDK exposes API 2 as an additive negotiation path. A
plugin may advertise `versions: [2, 1]`; the host selects the highest mutually
supported version. Unknown capabilities are omitted from the grant so a newer
plugin can degrade to the surfaces the host understands. The API-1
`api_version` handshake and strict capability negotiation remain unchanged for
existing plugins.

API-2 contributions are data-only schemas: contextual actions, bounded tables,
detail fields, notifications, and read-only repository metadata. The host
renders these schemas and does not load plugin UI code. Contribution builders
reject terminal control characters and enforce bounded row, field, column, and
text sizes. Process, network, and Git-mutation permissions are separate
capabilities and must be granted explicitly by the host.
