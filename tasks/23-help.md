# Task 23 — Discoverable keymap and command palette

**Priority:** P0

Use Bubbles help concepts for compact contextual footer and full `?` overlay. Centralize key bindings so help and behavior cannot drift. Add a command palette (`:` or Ctrl-P) if implementation remains simple; otherwise full help is mandatory and palette is P1.

**Acceptance:** Every P0 action is discoverable without README lookup.

**Status:** Complete — centralized default bindings include the click-equivalent `enter/d` diff action, mutation scope actions, navigation, filtering, refresh, help, modal close, and quit; help lines derive from the same binding table.
