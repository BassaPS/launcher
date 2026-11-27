/* One framework-independent implementation for every BASSAPS website. */
import snapshot from './services.json' with { type: 'json' };
import chromeCSS from './chrome-style.js';
const chromeSheet = new CSSStyleSheet();
chromeSheet.replaceSync(chromeCSS);
const asset = name => new URL(name, import.meta.url).href;
const star = `<svg class="star" viewBox="0 0 32 32" aria-hidden="true"><use href="${asset('icons.svg')}#brand-star"/></svg>`;
const logo = `<span>BASS<b>APS</b></span>${star}`;
const origins = Object.fromEntries(Object.entries(snapshot.services).map(([id, service]) => [id, new URL(service.url).origin]));
function transferAppearance(link) {
  const target = new URL(link.href);
  const preference = window.BassapsAppearance?.read();
  if (target.origin !== location.origin && Object.values(origins).includes(target.origin) && preference) {
    target.searchParams.set('bps-appearance', preference);
    link.href = target.href;
  }
}
function href(item, service) { return item.serviceId === service ? new URL(item.url).pathname : item.url; }
function active(item, service, pathname) {
  if (item.serviceId !== service) return false;
  if (service !== 'portal') return true;
  return item.id === 'news' ? pathname.startsWith('/news') : pathname === '/';
}
function navigation(service, pathname) {
  return snapshot.navigation.map(item => `<a href="${href(item, service)}" data-origin="${origins[item.serviceId]}"${active(item, service, pathname) ? ' aria-current="location"' : ''}>${item.label}</a>`).join('');
}
class BassapsHeader extends HTMLElement {
  static observedAttributes = ['service', 'path'];
  connectedCallback() {
    if (this.shadowRoot) return;
    const shadow = this.attachShadow({ mode: 'open' });
    shadow.adoptedStyleSheets = [chromeSheet];
    shadow.innerHTML = `<link rel="stylesheet" href="${asset('chrome.css')}"><header class="header"><div class="inner"><a class="logo" href="${snapshot.services.portal.url}" aria-label="BASSAPS ホーム">${logo}</a><nav class="navigation" aria-label="グローバルナビゲーション"></nav><div class="utilities"><label class="theme-control"><svg viewBox="0 0 24 24" aria-hidden="true"><circle cx="12" cy="12" r="8" fill="none" stroke="currentColor" stroke-width="1.5"/><path d="M12 4a8 8 0 0 1 0 16Z" fill="currentColor"/></svg><select class="theme" aria-label="表示テーマ"><option value="system">システム</option><option value="light">ライト</option><option value="dark">ダーク</option></select></label><button class="menu-button" aria-label="メニューを開く" aria-expanded="false" aria-controls="mobile-menu"><svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.8" aria-hidden="true"><path d="M4 6h16M4 12h16M4 18h16"/></svg></button></div></div></header><div class="mobile" id="mobile-menu" hidden><nav aria-label="モバイルナビゲーション"></nav></div>`;
    this.controller = new AbortController();
    shadow.querySelector('link')?.remove();
    const { signal } = this.controller;
    const trigger = shadow.querySelector('.menu-button');
    const menu = shadow.querySelector('.mobile');
    this.closeMenu = focus => {
      menu.hidden = true;
      trigger.setAttribute('aria-expanded', 'false');
      trigger.setAttribute('aria-label', 'メニューを開く');
      if (focus) trigger.focus();
    };
    trigger.addEventListener('click', () => {
      const open = menu.hidden;
      menu.hidden = !open;
      trigger.setAttribute('aria-expanded', String(open));
      trigger.setAttribute('aria-label', open ? 'メニューを閉じる' : 'メニューを開く');
    }, { signal });
    this.addEventListener('keydown', event => {
      if (event.key === 'Escape' && !menu.hidden) { this.closeMenu(true); event.preventDefault(); }
    }, { signal });
    document.addEventListener('click', event => { if (!event.composedPath().includes(this)) this.closeMenu(false); }, { signal });
    this.addEventListener('focusout', event => {
      const next = event.relatedTarget;
      if (next && !this.contains(next) && !shadow.contains(next)) this.closeMenu(false);
    }, { signal });
    const wide = matchMedia('(min-width: 1200px)');
    wide.addEventListener('change', () => this.closeMenu(false), { signal });
    shadow.querySelector('.theme').addEventListener('change', event => window.BassapsAppearance?.apply(event.target.value, true), { signal });
    const reflect = () => { shadow.querySelector('.theme').value = window.BassapsAppearance?.read() || 'system'; };
    document.addEventListener('bassaps-theme-change', reflect, { signal });
    reflect();
    shadow.addEventListener('click', event => {
      const link = event.composedPath().find(node => node instanceof HTMLAnchorElement);
      if (!link) return;
      this.closeMenu(false);
      if (event.button !== 0 || event.metaKey || event.ctrlKey || event.shiftKey || event.altKey) return;
      const destination = new URL(link.href);
      transferAppearance(link);
      if (destination.origin === location.origin && this.onNavigate) {
        event.preventDefault();
        this.onNavigate(destination.pathname + destination.search + destination.hash);
      }
    }, { signal });
    this.update();
  }
  update() {
    if (!this.shadowRoot) return;
    const html = navigation(this.getAttribute('service'), this.getAttribute('path') || location.pathname);
    this.shadowRoot.querySelectorAll('nav').forEach(nav => { nav.innerHTML = html; });
    const home = this.shadowRoot.querySelector('.logo');
    home.href = this.getAttribute('service') === 'portal' ? '/' : snapshot.services.portal.url;
    this.closeMenu?.(false);
  }
  attributeChangedCallback() { this.update(); }
  disconnectedCallback() { this.controller?.abort(); }
}
class BassapsFooter extends HTMLElement {
  connectedCallback() {
    if (this.shadowRoot) return;
    const shadow = this.attachShadow({ mode: 'open' });
    shadow.adoptedStyleSheets = [chromeSheet];
    shadow.innerHTML = `<link rel="stylesheet" href="${asset('chrome.css')}"><footer class="footer"><div class="inner footer-row"><span class="system">© ${new Date().getFullYear()} BASSAPS NETWORK</span><nav aria-label="フッターナビゲーション"><a href="https://bassaps.com/projects">Projects</a><a href="https://bassaps.com/about">About</a><a href="https://bassaps.com/privacy">Privacy</a><a href="${asset('licenses.txt')}">Licenses</a><a href="mailto:support@bassaps.com">support@bassaps.com</a></nav><span class="system">DESIGNED FOR THE FUTURE</span></div><div class="inner footer-disclaimer">個人運営の非公式プロジェクト · 関連ゲームの公式運営とは無関係です。</div></footer>`;
    shadow.querySelector('link')?.remove();
    shadow.addEventListener('click', event => {
      const link = event.composedPath().find(node => node instanceof HTMLAnchorElement);
      if (link) transferAppearance(link);
    });
  }
}
if (!customElements.get('bassaps-header')) customElements.define('bassaps-header', BassapsHeader);
if (!customElements.get('bassaps-footer')) customElements.define('bassaps-footer', BassapsFooter);
