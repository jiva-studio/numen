import type { Row } from '@numen/ui'

/** The vault the window is drawn over: notes on entropy, the papers they cite, two lectures and a deck. */
export const VAULT_NAME = 'Physics'

const createFileRow = (path: string): Row => ({
  id: path,
  name: path.split('/').pop() ?? path,
  hasChildren: false,
})

const createFolderRow = (path: string, names: readonly string[]): Row => ({
  id: path,
  name: path,
  hasChildren: true,
  rows: names.map((name) => createFileRow(`${path}/${name}`)),
})

export const ROWS: readonly Row[] = [
  createFolderRow('Physics', [
    'Entropy.md',
    'Entropy of mixing.md',
    'Free energy.md',
    "Maxwell's demon.md",
    'The second law.md',
  ]),
  createFolderRow('Computation', ["Landauer's principle.md", 'Shannon entropy.md']),
  createFolderRow('Reading', ['Boltzmann 1877.pdf', 'On Heat and Order.epub', 'Shannon 1948.pdf']),
  createFolderRow('Lectures', ['Lecture 4.mp3', 'Lecture 5.mp3']),
  createFolderRow('Decks', ['Sanskrit roots.md']),
  createFileRow('Inbox.md'),
  createFileRow('Reading list.md'),
]

export const OPEN = ['Physics', 'Computation', 'Reading', 'Lectures', 'Decks']

/** The note a window shows when the story is drawn somewhere other than a tab. */
export const NOTE = {
  title: 'Entropy',
  text: `How many arrangements a state could have been made of, counted the same way in a gas and in a message. Boltzmann counted the microstates a temperature leaves open; Shannon counted the messages a channel could have sent, and the two counts came out as one expression.

## Where the two meet

A demon that sorts fast molecules from slow ones appears to lower the entropy of the gas for nothing. It does not: sorting means knowing which molecule is which, and the knowing is paid for when the memory holding it is cleared.

- Boltzmann, 1877 — *Über die Beziehung*, p. 373
- Shannon, 1948 — the same logarithm, in bits
- Landauer, 1961 — what clearing one bit costs

> The measure is not about heat. It is about how much a description leaves open.

## What is still open

Whether Free energy belongs under this note or beside it. The counting there assumes a temperature the surroundings hold fixed, and that assumption is nowhere above.
`,
}
