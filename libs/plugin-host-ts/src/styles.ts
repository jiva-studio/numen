/**
 * The stylesheet each loaded plugin's frontend carries, by plugin id.
 *
 * The loader reads each stylesheet once into a constructed sheet, so a pane adopts it as it mounts:
 * a `<link>` added with the pane is applied only after it loads, and the pane is drawn unstyled
 * until then.
 */
const stylesRegistry = new Map<string, string>()
const sheets = new Map<string, CSSStyleSheet>()
const adoptions = new WeakMap<StyleRoot, Map<CSSStyleSheet, number>>()

/** A document or a shadow root that can adopt a constructed sheet. */
export type StyleRoot = Document | ShadowRoot

export function registerPluginStyle(pluginId: string, url: string): void {
  stylesRegistry.set(pluginId, url)
  sheets.delete(pluginId)
}

export function unregisterPluginStyle(pluginId: string): void {
  stylesRegistry.delete(pluginId)
  sheets.delete(pluginId)
}

export function getPluginStyle(pluginId: string): string | undefined {
  return stylesRegistry.get(pluginId)
}

export function clearPluginStyles(): void {
  stylesRegistry.clear()
  sheets.clear()
}

/** Whether this engine can build a stylesheet from text and let a document or shadow root adopt it. */
export function canConstructSheets(): boolean {
  return (
    typeof CSSStyleSheet !== 'undefined' &&
    typeof CSSStyleSheet.prototype.replaceSync === 'function' &&
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
 * Registers the stylesheet of a plugin and reads it into a sheet its panes adopt. A stylesheet that
 * cannot be read is still registered, and the panes link it instead.
 */
export async function loadPluginStyle(
  pluginId: string,
  url: string,
  fetchFn: typeof fetch,
): Promise<void> {
  registerPluginStyle(pluginId, url)
  if (!canConstructSheets()) return
  try {
    const res = await fetchFn(url)
    if (!res.ok) return
    const sheet = createSheet(await res.text(), new URL(url, document.baseURI).href)
    if (sheet && stylesRegistry.get(pluginId) === url) sheets.set(pluginId, sheet)
  } catch {
    // The panes link the stylesheet instead.
  }
}

/** The sheet the plugin's panes adopt, and nothing where its stylesheet was not read into one. */
export function getPluginSheet(pluginId: string): CSSStyleSheet | undefined {
  return sheets.get(pluginId)
}

/**
 * Adds the sheet to the root until the returned release is called. A root holds a sheet once
 * however many panes adopt it.
 */
export function adoptSheet(root: StyleRoot, sheet: CSSStyleSheet): () => void {
  const counts = adoptions.get(root) ?? new Map<CSSStyleSheet, number>()
  adoptions.set(root, counts)
  const count = counts.get(sheet) ?? 0
  if (count === 0 && !root.adoptedStyleSheets.includes(sheet)) {
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

/**
 * Adopts the plugin's sheet into the root, and gives back the release; nothing where the plugin
 * has no sheet, and the pane links its stylesheet instead.
 */
export function adoptPluginStyle(pluginId: string, root: StyleRoot): (() => void) | undefined {
  const sheet = sheets.get(pluginId)
  if (!sheet || !('adoptedStyleSheets' in root)) return undefined
  return adoptSheet(root, sheet)
}
