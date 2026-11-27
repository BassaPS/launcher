# Visual regression

From the workspace: npm ci; npm run sync; build each app; npm run preview. Ports 4310-4315 map to Portal/Register/Download/Docs/Status/Launcher. node tools/reference-server.cjs serves supplied HTML on 4316. Use npm run capture:reference and npm run test:visual. Chromium must be installed for the pinned Playwright version (npx playwright install chromium).

evidence/before contains original screenshots; evidence/reference contains the three rendered references; evidence/after and evidence/verification contain implemented views. Browser fixtures are local and explicitly synthetic, never real accounts or successful production connection evidence.

The visual verifier checks both themes and compact/desktop widths, menu keyboard handling, preference persistence, overflow, browser exceptions and axe WCAG A/AA. It compares shared header/footer PNG hashes after removing only current-page emphasis. The real footer is captured at a shared viewport origin using only fully painted pixel rows; the fractional boundary containing the unrelated page background is excluded. Natural footer dimensions and typography remain measured. The separate breakpoint check covers boundary widths and browser text zoom. Register's existing browser tests exercise submission, authorization and admin publishing through mocks; Launcher tests exercise existing runtime boundaries without game launch.

Each repository's check-design.mjs verifies its pinned manifest with SHA-256 in CI. Cross-repository visual verification runs from this workspace so all services are built from the same canonical revision. Any unexplained common-chrome difference blocks completion.
