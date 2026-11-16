/** The tags of a note as its frontmatter holds them: the key `tags:` with one list item for each tag. */

const FRONTMATTER = /^---\r?\n([\s\S]*?)\r?\n---/

/** The text of a note with the key `tags` in its frontmatter, which is made when the note has none. */
export const addTags = (text: string, tags: readonly string[]): string => {
  const block = `tags:\n${tags.map((tag) => `  - ${JSON.stringify(tag)}`).join('\n')}`
  const found = FRONTMATTER.exec(text)
  if (!found) return `---\n${block}\n---\n\n${text}`
  if (/^tags:/m.test(found[1])) throw new Error('the note has the tags key already: write every tag of the note in the table')
  return `---\n${block}\n${found[1]}\n---${text.slice(found[0].length)}`
}

/** The tags the table names for each note, in the order of the rows: a note may be named once for each tag. */
export const groupTags = (rows: readonly (readonly string[])[]): Map<string, string[]> => {
  const [header = [], ...body] = rows
  if (header.join('|') !== 'note|tag') throw new Error('the table must start with the header "| note | tag |"')
  const grouped = new Map<string, string[]>()
  for (const [note = '', tag = ''] of body) grouped.set(note, [...(grouped.get(note) ?? []), tag])
  return grouped
}
