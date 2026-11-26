const ARROW_STEP: Readonly<Record<string, number>> = { ArrowRight: 1, ArrowLeft: -1 }

function selectTab(tabs: HTMLElement[], selected: HTMLElement): void {
  for (const tab of tabs) {
    const isSelected = tab === selected
    tab.setAttribute('aria-selected', String(isSelected))
    tab.setAttribute('tabindex', isSelected ? '0' : '-1')
    const panel = document.getElementById(tab.getAttribute('aria-controls') ?? '')
    if (panel) panel.hidden = !isSelected
  }
}

/**
 * Wires the Overview, Access & tests and Version history tabs of a plugin page; the first tab
 * starts selected.
 */
export function initPluginTabs(): void {
  const tabs = [...document.querySelectorAll<HTMLElement>('[data-plugin-tab]')]
  const first = tabs[0]
  if (!first) return
  for (const [index, tab] of tabs.entries()) {
    tab.addEventListener('click', () => selectTab(tabs, tab))
    tab.addEventListener('keydown', (event) => {
      const step = ARROW_STEP[event.key]
      if (step === undefined) return
      event.preventDefault()
      const next = tabs[(index + step + tabs.length) % tabs.length]
      if (!next) return
      selectTab(tabs, next)
      next.focus()
    })
  }
  selectTab(tabs, first)
}
