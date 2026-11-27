/* Early, external and CSP-compatible. Public appearance preference only. */
(() => {
  const valid = value => ['light', 'dark', 'system'].includes(value);
  const media = matchMedia('(prefers-color-scheme: dark)');
  function read() {
    try {
      const url = new URL(location.href);
      const transferred = url.searchParams.get('bps-appearance');
      if (valid(transferred)) {
        localStorage.setItem('bassaps-appearance', transferred);
        url.searchParams.delete('bps-appearance');
        history.replaceState(history.state, '', url);
        return transferred;
      }
      const current = localStorage.getItem('bassaps-appearance');
      if (valid(current)) return current;
      for (const key of ['bassaps-theme', 'bassaps.theme', 'vitepress-theme-appearance']) {
        const previous = localStorage.getItem(key);
        if (valid(previous)) return previous;
      }
    } catch { /* Appearance still works without persistent storage. */ }
    return 'system';
  }
  let preference = read();
  function apply(value, save = false) {
    if (!valid(value)) return;
    preference = value;
    const dark = value === 'dark' || (value === 'system' && media.matches);
    const root = document.documentElement;
    root.dataset.theme = dark ? 'dark' : 'light';
    root.dataset.appearance = value;
    root.classList.toggle('dark', dark);
    root.style.colorScheme = dark ? 'dark' : 'light';
    if (save) {
      try { localStorage.setItem('bassaps-appearance', value); } catch { /* Session-only choice. */ }
    }
    document.dispatchEvent(new CustomEvent('bassaps-theme-change', { detail: { preference: value, dark } }));
  }
  window.BassapsAppearance = { apply, read: () => preference };
  apply(preference);
  media.addEventListener('change', () => apply(preference));
  addEventListener('storage', event => {
    if (event.key === 'bassaps-appearance') apply(valid(event.newValue) ? event.newValue : 'system');
  });
})();
