/** A small tree: a folder with a folder in it, an empty folder and a loose row. */
import type { Row } from '../lib/row'

export const WORK_ROWS: readonly Row[] = [
  {
    id: 'work',
    name: 'Work',
    hasChildren: true,
    rows: [
      {
        id: 'plans',
        name: 'Plans',
        hasChildren: true,
        rows: [{ id: 'friday', name: 'Friday', hasChildren: false }],
      },
      { id: 'pages', name: 'Pages', hasChildren: false },
    ],
  },
  { id: 'empty', name: 'Empty', hasChildren: true },
  { id: 'loose', name: 'Loose', hasChildren: false },
]
