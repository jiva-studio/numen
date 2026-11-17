/** The texts the plugin shows to a person. */
export const WORDS = {
  /** The title a note shows where it has none. */
  untitled: 'Untitled',
  /** What a name is to a person using assistive technology: its title, and the note it is named under. */
  nameLabel: (title: string, owner: string) => `${title}, name under ${owner}`,
}
