/** Reads the README.md of a plugin from the server that serves its files. */

/** The README.md a plugin serves, and an empty text where none can be fetched or the server answers a page. */
export async function fetchPluginReadme(pluginId: string): Promise<string> {
  try {
    if (typeof fetch !== 'function') return ''
    const response = await fetch(`/plugin/${pluginId}/README.md`)
    if (!response.ok) return ''
    const contentType = response.headers?.get?.('content-type') ?? ''
    if (contentType.includes('text/html')) return ''
    return await response.text()
  } catch {
    return ''
  }
}
