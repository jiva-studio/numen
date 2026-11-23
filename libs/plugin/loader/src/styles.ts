/** A document or a shadow root that can adopt a constructed sheet. */
export type StyleRoot = Document | ShadowRoot

function canConstructSheets(): boolean {
  return (
    typeof CSSStyleSheet !== 'undefined' &&
    typeof CSSStyleSheet.prototype.replaceSync === 'function' &&
    // Stryker disable next-line ConditionalExpression,StringLiteral: JSDOM always defines document
    typeof document !== 'undefined' &&
    'adoptedStyleSheets' in document
  )
}

/** A constructed copy of the text, and nothing where the engine cannot construct one. */
export function createSheet(text: string, baseURL?: string): CSSStyleSheet | undefined {
  if (!canConstructSheets()) return undefined
  try {
    const sheet = new CSSStyleSheet(baseURL ? { baseURL } : {})
    sheet.replaceSync(text)
    return sheet
  } catch {
    return undefined
  }
}

/**
 * Adds a sheet to a root until the returned release is called. A root holds a sheet once however
 * many panes adopt it.
 */
function createSheetAdopter() {
  const adoptions = new WeakMap<StyleRoot, Map<CSSStyleSheet, number>>()
  return (root: StyleRoot, sheet: CSSStyleSheet): (() => void) => {
    const counts = adoptions.get(root) ?? new Map<CSSStyleSheet, number>()
    adoptions.set(root, counts)
    const count = counts.get(sheet) ?? 0
    if (!root.adoptedStyleSheets.includes(sheet)) {
      root.adoptedStyleSheets = [...root.adoptedStyleSheets, sheet]
    }
    counts.set(sheet, count + 1)
    let isReleased = false
    return () => {
      if (isReleased) return
      isReleased = true
      const remaining = (counts.get(sheet) ?? 1) - 1
      if (remaining > 0) {
        counts.set(sheet, remaining)
        return
      }
      counts.delete(sheet)
      root.adoptedStyleSheets = root.adoptedStyleSheets.filter((one) => one !== sheet)
    }
  }
}

/**
 * The stylesheet each loaded plugin's frontend carries, by plugin id, owned by the window that loads
 * the plugins.
 *
 * The loader reads each stylesheet once into a constructed sheet, so a pane adopts it as it mounts:
 * a `<link>` added with the pane is applied only after it loads, and the pane is drawn unstyled
 * until then.
 */
export function createPluginStyles() {
  const urls = new Map<string, string>()
  const sheets = new Map<string, CSSStyleSheet>()

  const register = (pluginId: string, url: string): void => {
    urls.set(pluginId, url)
    sheets.delete(pluginId)
  }

  const adoptSheet = createSheetAdopter()

  return {
    register,
    unregister: (pluginId: string): void => {
      urls.delete(pluginId)
      sheets.delete(pluginId)
    },
    getUrl: (pluginId: string): string | undefined => urls.get(pluginId),
    clear: (): void => {
      urls.clear()
      sheets.clear()
    },
    /**
     * Registers the stylesheet of a plugin and reads it into a sheet its panes adopt. A stylesheet
     * that cannot be read is still registered, and the panes link it instead.
     */
    async load(pluginId: string, url: string, fetchFn: typeof fetch): Promise<void> {
      register(pluginId, url)
      if (!canConstructSheets()) return
      try {
        const res = await fetchFn(url)
        if (!res.ok) return
        const sheet = createSheet(await res.text(), new URL(url, document.baseURI).href)
        if (sheet && urls.get(pluginId) === url) sheets.set(pluginId, sheet)
      } catch {
        sheets.delete(pluginId)
      }
    },
    /** The sheet the plugin's panes adopt, and nothing where its stylesheet was not read into one. */
    getSheet: (pluginId: string): CSSStyleSheet | undefined => sheets.get(pluginId),
    adoptSheet,
    /**
     * Adopts the plugin's sheet into the root, and gives back the release; nothing where the plugin
     * has no sheet, and the pane links its stylesheet instead.
     */
    adopt(pluginId: string, root: StyleRoot): (() => void) | undefined {
      const sheet = sheets.get(pluginId)
      if (!sheet || !('adoptedStyleSheets' in root)) return undefined
      return adoptSheet(root, sheet)
    },
  }
}

export type PluginStyles = ReturnType<typeof createPluginStyles>
