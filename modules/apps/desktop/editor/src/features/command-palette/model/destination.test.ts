/**
 * Where something chosen in the palette takes the person, asked without a
 * window.
 *
 * Each of the three lands somewhere else, and taking the wrong one is a person
 * who asked for a passage and was given the top of a note.
 */
import { describe, expect, it } from 'vitest'
import { openDestination, type DestinationDeps } from './destination'
import type { SearchDestination } from './search'

/** A window that writes down where it was taken. */
const window = () => {
  const travelled: string[] = []
  const opened: string[] = []
  const shown: string[] = []
  const places: DestinationDeps = {
    travel: async (path) => void travelled.push(path),
    opensAt: async (path, run) => void opened.push(`${path} ${run.from} ${run.to}`),
    opens: (path, title, line) =>
      void shown.push(`${path} ${title}${line === undefined ? '' : ` ${line}`}`),
  }
  return { places, travelled, opened, shown }
}

const landing = (over: Partial<SearchDestination>): SearchDestination => ({
  at: 'file',
  path: 'Note.md',
  title: 'A note',
  ...over,
})

describe('a name chosen', () => {
  it('travels in the plex the person is looking at, and opens no tab', async () => {
    const one = window()

    await openDestination(landing({ at: 'plex' }), one.places)

    expect(one.travelled).toStrictEqual(['Note.md'])
    expect(one.shown).toStrictEqual([])
  })
})

describe('a file chosen', () => {
  it('opens under the name it is called by, naming no editor of its own', async () => {
    const one = window()

    await openDestination(landing({}), one.places)

    expect(one.shown).toStrictEqual(['Note.md A note'])
  })

  it('is called by the path it is filed at where it has no name', async () => {
    const one = window()

    await openDestination(landing({ title: '' }), one.places)

    expect(one.shown).toStrictEqual(['Note.md Note.md'])
  })

  it('carries the line the heading was found at', async () => {
    const one = window()

    await openDestination(landing({ line: 12 }), one.places)

    expect(one.shown).toStrictEqual(['Note.md A note 12'])
  })

  it('carries the top of the file for the line it opens with', async () => {
    const one = window()

    await openDestination(landing({ line: 0 }), one.places)

    expect(one.shown).toStrictEqual(['Note.md A note 0'])
  })
})

describe('a passage chosen', () => {
  it('opens the document it stands in, at the span it names', async () => {
    const one = window()

    await openDestination(
      landing({ at: 'document', path: 'library/mahabharata.epub', start: 40_512, length: 31 }),
      one.places,
    )

    expect(one.opened).toStrictEqual(['library/mahabharata.epub 40512 40543'])
    expect(one.shown).toStrictEqual([])
  })

  it('opens it at its first page where it names no span', async () => {
    const one = window()

    await openDestination(landing({ at: 'document', path: 'library/mahabharata.epub' }), one.places)

    expect(one.opened).toStrictEqual(['library/mahabharata.epub 0 0'])
  })
})

describe('nothing chosen', () => {
  it('takes the person nowhere at all', async () => {
    const one = window()

    await openDestination(null, one.places)

    expect([one.travelled, one.opened, one.shown]).toStrictEqual([[], [], []])
  })
})
