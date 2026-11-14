/** The links of a note as its frontmatter holds them: the block `links:` with one record for each link. */

export interface NoteLink {
  readonly to: string
  readonly role: string
  readonly label: string
}

const FRONTMATTER = /^---\r?\n([\s\S]*?)\r?\n---/

const unquote = (value: string): string => value.trim().replace(/^(["'])(.*)\1$/, '$2')

/** The links a note writes, in the order of the block. */
export const readLinks = (text: string): NoteLink[] => {
  const lines = (FRONTMATTER.exec(text)?.[1] ?? '').split(/\r?\n/)
  const start = lines.findIndex((line) => /^links:\s*$/.test(line))
  if (start < 0) return []
  const records: Record<string, string>[] = []
  for (const line of lines.slice(start + 1)) {
    if (/^[A-Za-z_]/.test(line)) break
    const first = /^\s*-\s+([A-Za-z_]+):\s*(.*)$/.exec(line)
    const next = /^\s+([A-Za-z_]+):\s*(.*)$/.exec(line)
    if (first) records.push({ [first[1]]: unquote(first[2]) })
    else if (next && records.length > 0) records[records.length - 1][next[1]] = unquote(next[2])
  }
  return records.map((record) => ({ to: record['to'] ?? '', role: record['role'] ?? '', label: record['label'] ?? '' }))
}

/** The title of the note an address names: `[[notes/Quay#Dock|the quay]]` is "Quay". */
export const getLinkTitle = (address: string): string => {
  const inner = address.replace(/^\[\[|\]\]$/g, '').split('|')[0].split('#')[0]
  return (inner.split('/').pop() ?? inner).replace(/\.md$/, '')
}

/** The text of a note with the links in front of its body. */
export const addLinks = (text: string, links: readonly NoteLink[]): string => {
  if (FRONTMATTER.test(text)) throw new Error('the note has a frontmatter already: write the links in the table, not in the text')
  const records = links.map((link) => {
    const label = link.label === '' ? '' : `\n    label: ${JSON.stringify(link.label)}`
    return `  - to: ${JSON.stringify(link.to)}\n    role: ${link.role}${label}`
  })
  return `---\nlinks:\n${records.join('\n')}\n---\n\n${text}`
}
