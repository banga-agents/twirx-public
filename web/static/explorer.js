// Provenance explorer enhancement.
//
// The page already contains the complete proof: every field, every panel,
// expanded, in source order. This module does one thing — turn that list into a
// tab set so a reader can compare fields without scrolling past the others.
//
// No dependency, no framework, no network call, no telemetry, no storage.
// If this file fails to load, the page loses nothing but convenience.

const root = document.querySelector('[data-explorer]');

if (root) {
  const list = root.querySelector('.explorer__list ol');
  const links = [...root.querySelectorAll('[data-field-link]')];
  const panels = [...root.querySelectorAll('[data-field]')];

  if (list && links.length && links.length === panels.length) {
    const byId = new Map(panels.map((p) => [p.dataset.field, p]));

    list.setAttribute('role', 'tablist');
    list.setAttribute('aria-orientation', 'vertical');
    for (const li of list.children) li.setAttribute('role', 'presentation');

    links.forEach((link, i) => {
      const id = link.dataset.fieldLink;
      const panel = byId.get(id);
      if (!panel) return;

      link.setAttribute('role', 'tab');
      link.id = `tab-${id}`;
      link.setAttribute('aria-controls', panel.id);
      panel.setAttribute('role', 'tabpanel');
      panel.setAttribute('aria-labelledby', link.id);
      panel.tabIndex = 0;

      link.addEventListener('click', (event) => {
        event.preventDefault();
        select(i, true);
      });

      link.addEventListener('keydown', (event) => {
        const step = { ArrowDown: 1, ArrowRight: 1, ArrowUp: -1, ArrowLeft: -1 }[event.key];
        let next = null;
        if (step) next = (i + step + links.length) % links.length;
        else if (event.key === 'Home') next = 0;
        else if (event.key === 'End') next = links.length - 1;
        if (next === null) return;
        event.preventDefault();
        select(next, true);
        links[next].focus();
      });
    });

    function select(index, updateHash) {
      links.forEach((link, i) => {
        const on = i === index;
        link.setAttribute('aria-selected', String(on));
        link.tabIndex = on ? 0 : -1;
        const panel = byId.get(link.dataset.fieldLink);
        if (panel) panel.hidden = !on;
      });
      if (updateHash && history.replaceState) {
        history.replaceState(null, '', `#field-${links[index].dataset.fieldLink}`);
      }
    }

    // Honour a deep link such as /proof/explorer/#field-price_currency.
    const wanted = links.findIndex(
      (link) => `#field-${link.dataset.fieldLink}` === location.hash,
    );
    select(wanted >= 0 ? wanted : 0, false);

    // The grid lives on .explorer, not on the section that carries the hook.
    root.querySelector('.explorer')?.classList.add('explorer--enhanced');
  }
}
