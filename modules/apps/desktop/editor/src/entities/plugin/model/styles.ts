/**
 * Scoped stylesheet registry for plugin views and Shadow DOM style containment.
 */
const stylesRegistry = new Map<string, string>()

export function registerPluginStyle(pluginId: string, url: string): void {
  stylesRegistry.set(pluginId, url)
}

export function unregisterPluginStyle(pluginId: string): void {
  stylesRegistry.delete(pluginId)
}

export function getPluginStyle(pluginId: string): string | undefined {
  return stylesRegistry.get(pluginId)
}

export function clearPluginStyles(): void {
  stylesRegistry.clear()
}

export const PANE_LAYOUT_STYLE =
  ':host, .plugin-view-container { display: flex !important; flex-direction: column !important; width: 100% !important; height: 100% !important; min-width: 0 !important; min-height: 0 !important; flex: 1 1 0% !important; }'

export const PANE_HOST_STYLE: Record<string, string> = {
  display: 'flex',
  flexDirection: 'column',
  width: '100%',
  height: '100%',
  minWidth: '0',
  minHeight: '0',
  flex: '1 1 0%',
}

export function adoptHostStyles(shadowRoot: ShadowRoot): void {
  if (typeof document === 'undefined') return
  const hostStyles = document.querySelectorAll<HTMLStyleElement | HTMLLinkElement>(
    'head style:not([data-plugin-id]), head link[rel="stylesheet"]:not([data-plugin-id])',
  )
  for (const node of hostStyles) {
    if (node.tagName.toLowerCase() === 'style' && node.textContent) {
      const copy = document.createElement('style')
      copy.textContent = node.textContent
      shadowRoot.appendChild(copy)
    } else if (node.tagName.toLowerCase() === 'link') {
      const copy = document.createElement('link')
      copy.rel = 'stylesheet'
      copy.href = (node as HTMLLinkElement).href
      shadowRoot.appendChild(copy)
    }
  }
}

export function attachPluginShadow(el: HTMLElement | null): ShadowRoot | null {
  if (!el) return null
  if (el.shadowRoot) return el.shadowRoot
  if (typeof el.attachShadow === 'function') {
    try {
      const shadow = el.attachShadow({ mode: 'open' })
      adoptHostStyles(shadow)
      return shadow
    } catch {
      return el.shadowRoot
    }
  }
  return null
}
