import type { Row } from '@numen/ui'

/** The folders and files of the vault the demo draws, with every folder open. */
export const FILE_ROWS: readonly Row[] = [
  {
    id: 'physics',
    name: 'Physics',
    hasChildren: true,
    rows: [
      { id: 'physics/Entropy.md', name: 'Entropy.md', hasChildren: false, rows: [] },
      {
        id: 'physics/Entropy of mixing.md',
        name: 'Entropy of mixing.md',
        hasChildren: false,
        rows: [],
      },
      { id: 'physics/The second law.md', name: 'The second law.md', hasChildren: false, rows: [] },
      { id: 'physics/Free energy.md', name: 'Free energy.md', hasChildren: false, rows: [] },
    ],
  },
  {
    id: 'computation',
    name: 'Computation',
    hasChildren: true,
    rows: [
      {
        id: 'computation/Shannon entropy.md',
        name: 'Shannon entropy.md',
        hasChildren: false,
        rows: [],
      },
      {
        id: "computation/Landauer's principle.md",
        name: "Landauer's principle.md",
        hasChildren: false,
        rows: [],
      },
    ],
  },
  {
    id: 'reading',
    name: 'Reading',
    hasChildren: true,
    rows: [
      {
        id: 'reading/Boltzmann 1877.pdf',
        name: 'Boltzmann 1877.pdf',
        hasChildren: false,
        rows: [],
      },
      { id: 'reading/Shannon 1948.pdf', name: 'Shannon 1948.pdf', hasChildren: false, rows: [] },
    ],
  },
  {
    id: 'lectures',
    name: 'Lectures',
    hasChildren: true,
    rows: [
      { id: 'lectures/Lecture 4.mp3', name: 'Lecture 4.mp3', hasChildren: false, rows: [] },
      { id: 'lectures/Feynman 1964.md', name: 'Feynman 1964.md', hasChildren: false, rows: [] },
    ],
  },
  {
    id: 'decks',
    name: 'Decks',
    hasChildren: true,
    rows: [
      { id: 'decks/Thermodynamics.md', name: 'Thermodynamics.md', hasChildren: false, rows: [] },
    ],
  },
  { id: 'Inbox.md', name: 'Inbox.md', hasChildren: false, rows: [] },
  { id: 'Reading list.md', name: 'Reading list.md', hasChildren: false, rows: [] },
]

/** The row selected when the tab opens. */
export const SELECTED_ROW = 'physics/Entropy.md'
