# Dependency graph and execution lanes

## Foundation

```text
120
 ↓
121 → 122 → 123 → 124 → 125
```

Do not skip this. Tasks 122–125 prevent every later feature from creating one-off state, stale refresh behavior or single-repo assumptions.

## Rebase / conflict critical path

```text
126 → 127 → 128 → 129 → 130
                    └────→ 131

133 → 134 → 135 ─────────────┐
136 ─────────────────────────┤
137 → 138 → 139 → 140 → 141 ├→ 143 → 132
                         142 ┘
```

Task 143 is deliberately central. Rebase, cherry-pick, revert and merge must share conflict lifecycle behavior.

## Recovery

```text
144 → 145 → 146 → 147
```

## Bisect

```text
148 → 149
  └────→ 150 (also requires custom-command foundation 167)
```

Task 150 can be delayed until Task 167 exists.

## Submodules

```text
151 → 152
  └→ 153
152 + 153 → 154
```

## Tags / remotes

```text
155 → 156
157 → 158 → 159
          └→ 160 (also uses compare 161)
```

Task 160 may land after 161 even though numbering is lower; record the dependency exception.

## History / inspection

```text
161 → 162 → 163
126 + 131 + 162 → 164
124 → 165
142 + 161 → 166
```

## Extensibility

```text
167 → 168 → 169
```

## GitHub

```text
170 → 171
  └→ 172 → 173
```

## Multi-repo differentiation

```text
125 + 157 + 159 → 174
125 + 151 + 155 + 172 → 175
145 + 146 + 174 → 176
174 + 175 → 177
125 + 162 + 170 + 175 → 178
175 → 179 → 180
```

## Hardening / release

```text
181
182
183
  \  |  /
   184 → 185 → 186
```

Task 184 waits for all required parity lanes even if individual feature tasks complete out of numeric order.
