/** The tags of a note as its frontmatter holds them now, whichever way the product spelled the list. */

const FRONTMATTER = /^---\r?\n([\s\S]*?)\r?\n---/

const unquote = (value: string): string => value.trim().replace(/^(["'])(.*)\1$/, '$2')

const splitList = (inline: string): string[] =>
  inline
    .replace(/^\[|\]$/g, '')
    .split(',')
    .map(unquote)
    .filter((tag) => tag !== '')

/** The tags of the key `tags`: a list in the flow form, a list with one item for each tag, or one tag as a text. */
export const readTags = (text: string): string[] => {
  const lines = (FRONTMATTER.exec(text)?.[1] ?? '').split(/\r?\n/)
  const header = lines.find((line) => /^tags:/.test(line))
  if (header === undefined) return []
  const rest = header.replace(/^tags:/, '').trim()
  if (rest.startsWith('[')) return splitList(rest)
  if (rest !== '') return [unquote(rest)]
  const items: string[] = []
  for (const line of lines.slice(lines.indexOf(header) + 1)) {
    const item = /^\s*-\s+(.*)$/.exec(line)
    if (!item) break
    items.push(unquote(item[1]))
  }
  return items
}
