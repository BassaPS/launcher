# Accessibility

Target normal text contrast 4.5:1, large text/non-text controls 3:1. Brand pink/blue alone are not normal-text colors on light surfaces; use on-primary, link and semantic roles. Status combines text/icons and color. Inputs have labels, placeholders remain readable, and links within text are underlined. Disabled controls are distinguishable and remain natively disabled.

Shared mobile navigation exposes aria-expanded, removes hidden links from focus, closes on Escape with focus return, dismisses on outside click and resets on resize/navigation. Native theme select preserves keyboard behavior even when rendered as the reference's half-circle control. Native dialogs retain focus management. Status history bars have named image roles inside a labeled group.

Web appearance is owned by theme.js, not separate React/Vue implementations. Preference is stored locally. Shared cross-service navigation transfers a validated light/dark/system preference in bps-appearance, then removes that parameter. No authentication cookie, account token or new cross-domain cookie is used. Direct visits retain local/system choice. Launcher keeps its existing local Zustand preference and Wails behavior.
