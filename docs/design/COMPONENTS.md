# Components

Global Web chrome: bassaps-header and bassaps-footer, implemented once in chrome.js with isolated Shadow DOM styling. React's generated PlatformHeader/PlatformFooter only bridge attributes/navigation. Vue uses the same custom elements. Primary buttons use the supplied pink-blue gradient, lavender offset edge and asymmetric radius. Secondary buttons are outlined, with the reference's rounded shape. Inputs keep explicit labels, 48px targets and visible focus/error states.

Existing native buttons, forms, tables, tabs, dialogs, notices, status badges, progress bars and loaders inherit canonical CSS and semantic roles. They retain their existing state machines and API calls. Do not replace a real form with a demo. Do not invent an account menu or language switcher on a service that does not have one.

Docs navigation uses native details/summary with the existing navigation data, avoiding nested button roles in the pinned VitePress sidebar. Previous/next reading links are derived from the same existing navigation order in page metadata. Launcher uses its existing React controls, Wails titlebar and sidebar, themed with the same assets and component tokens. Its language/account/settings/window actions are unchanged.
