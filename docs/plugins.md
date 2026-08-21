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
payloads are opaque JSON and are bounded by the SDK limits. Runtime transport,
timeouts, crash containment, and permission prompts are owned by the plugin
runtime tasks.

Compatibility guarantee: API version 1 is additive within the declared JSON
fields. Hosts ignore optional fields they do not use, while plugins must not
depend on undeclared capabilities or private gitwatch packages. A future
breaking wire change requires a new API version.
