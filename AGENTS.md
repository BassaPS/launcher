# BASSAPS UI development rules

- Read DESIGN.md and docs/design/reference/ before UI changes. They are the authoritative visual reference.
- Use the material-design-3-ui skill for hierarchy, interaction, adaptive layout, motion and accessibility; BASSAPS owns the appearance.
- Prefer the canonical packages/design-system implementation. Never redesign global chrome per service or edit public/bassaps snapshots directly.
- Do not add arbitrary colors, fonts, radii, gradients or icon families. Update semantic tokens and regenerate all six snapshots.
- Check all services when shared UI changes. Resolve conflicts with the supplied design before implementation.
- Render the affected screens; verify light/dark, narrow/wide, keyboard and accessibility. A build alone is insufficient.
- Preserve APIs, auth, sessions, game/launcher/update/file operations, routes, user data and security settings.
- No push to production branches, deployment, DNS, Access, secret or database changes without explicit approval for that action.
