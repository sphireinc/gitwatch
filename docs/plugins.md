# Plugin contract

Plugins are separate processes. A plugin declares a JSON manifest with an
identifier, version, API version, executable, and capabilities. gitwatch
validates the manifest and negotiates only capabilities supported by the host.

The v1 contract does not permit a plugin to execute Git directly inside the
gitwatch process or mutate TUI state. Startup begins with a newline-delimited
`handshake` message containing the API version and the capabilities granted by
the host; the plugin must return a matching `handshake` response. Runtime
transport, timeouts, crash containment, and permission prompts are owned by the
plugin runtime tasks.
