/**
 * Parses a plugin's README.md into a summary and expandable detail around `<!-- more -->`,
 * strips a leading `# Title` heading, and rewrites relative `assets/` links to `/plugin/<id>/assets/`.
 */
export interface PluginReadmeParts {
  readonly summary: string
  readonly more: string
}

const LEADING_H1 = /^\s*#\s+[^\n]+(?:\n+|$)/
const MORE_MARKER = /<!--\s*more\s*-->/i
const RELATIVE_ASSET_LINK = /\]\((?:\.\/)?assets\//g

/** Splits a plugin README on `<!-- more -->` with fallback to the manifest description. */
export function parsePluginReadme(
  raw: string,
  pluginId: string,
  fallbackDescription: string,
): PluginReadmeParts {
  const rewritten = raw.replace(RELATIVE_ASSET_LINK, `](/plugin/${pluginId}/assets/`)
  const body = rewritten.replace(LEADING_H1, '').trim()
  if (body === '') {
    return { summary: fallbackDescription.trim(), more: '' }
  }
  const match = MORE_MARKER.exec(body)
  if (!match) {
    return { summary: body, more: '' }
  }
  const summary = body.slice(0, match.index).trim() || fallbackDescription.trim()
  const more = body.slice(match.index + match[0].length).trim()
  return { summary, more }
}

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
