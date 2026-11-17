import type { LinkDirection } from '@numen/plugin-sdk'
import type { NameRelation } from './types'

/** The texts the plugin shows to a person. */
export const WORDS = {
  /** The title a note shows where it has none. */
  untitled: 'Untitled',
  /** What a name is to a person using assistive technology: its title, and how it is related to the note it is named under. */
  nameLabel: (title: string, owner: string, _relation?: NameRelation, _direction?: LinkDirection) =>
    `${title}, name under ${owner}`,
}
