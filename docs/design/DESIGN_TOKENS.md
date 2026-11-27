# Tokens and single source

Canonical source: packages/design-system/tokens.json. Generated files: tokens.css, fonts.css, chrome.js, chrome.css, foundation.css, web.css, launcher.css, fonts, notices and SVGs. Web Component chrome is the same JavaScript in React and VitePress; no service owns a separate global header/footer.

Run npm run sync from the design workspace. npm run check:sync fails if any of the six app snapshots differs. Each public/bassaps/manifest.json records every file's SHA-256 and revision. A repository builds independently from its pinned snapshot; builds never import a parent folder. The canonical generator and all snapshots must be reviewed together. Do not hand-edit a snapshot.

Reference colors become semantic roles: background, surface, elevated, text, muted, outline, primary/on-primary, link, success, warning, error, info and scrim. Legacy framework variables are adapters to these roles. Hex colors belong only in the token source and approved SVG assets. Raw reference HTML/CSS is retained unchanged and excluded from production Docs routes.
