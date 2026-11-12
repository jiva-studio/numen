/** The README.md a plugin serves, and an empty text where none can be fetched or the server answers a page. */
export async function readPluginReadme(pluginId: string): Promise<string> {
  try {
    const response = await fetch(`/plugin/${pluginId}/README.md`)
    if (!response.ok) return ''
    if (response.headers.get('content-type')?.includes('text/html')) return ''
    return await response.text()
  } catch {
    return ''
  }
}
