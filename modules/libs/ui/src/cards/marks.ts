/**
 * The few marks a stencil and a deck are drawn with, declared once.
 *
 * Each is drawn on twelve units and takes the ink of whatever holds it.
 */

/** What a mark shows. */
export type Mark = 'grip' | 'cross' | 'plus' | 'bin'

/** One mark: what it is drawn from, and whether it is filled or stroked. */
export interface Drawing {
  readonly path: string
  readonly filled: boolean
}

export const MARKS: Readonly<Record<Mark, Drawing>> = {
  grip: {
    path:
      'M3.2 1.8h1.6v1.6H3.2z M7.2 1.8h1.6v1.6H7.2z' +
      ' M3.2 5.2h1.6v1.6H3.2z M7.2 5.2h1.6v1.6H7.2z' +
      ' M3.2 8.6h1.6v1.6H3.2z M7.2 8.6h1.6v1.6H7.2z',
    filled: true,
  },
  cross: { path: 'M3 3 9 9 M9 3 3 9', filled: false },
  plus: { path: 'M6 2.5V9.5 M2.5 6H9.5', filled: false },
  bin: {
    path:
      'M2.6 3.6H9.4 M4.9 3.6V2.5h2.2v1.1' +
      ' M3.7 3.6l.5 6.3h3.6l.5-6.3' +
      ' M5.2 5.3v2.9 M6.8 5.3v2.9',
    filled: false,
  },
}
