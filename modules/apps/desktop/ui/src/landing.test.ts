/**
 * Where something chosen in the palette takes the person, asked without a
 * window.
 *
 * Each of the three lands somewhere else, and taking the wrong one is a person
 * who asked for a passage and was given the top of a note.
 */
import { describe, expect, it } from 'vitest'
import { lands, type Places } from './landing'
import type { Landing } from './finding'

/** A window that writes down where it was taken. */
const window = () => {
  const travelled: string[] = []
  const opened: string[] = []
  const shown: string[] = []
  const entered: string[] = []
  const places: Places = {
    travel: async (path) => void travelled.push(path),
    opensAt: async (path, run) => void opened.push(`${path} ${run.start} ${run.length}`),
    shows: (path, title) => void shown.push(`${path} ${title}`),
    entersAt: (path, line) => void entered.push(`${path} ${line}`),
  }
  return { places, travelled, opened, shown, entered }
}

const landing = (over: Partial<Landing>): Landing => ({
  at: 'note',
  path: 'Note.md',
  title: 'A note',
  ...over,
})

describe('a name chosen', () => {
  it('travels in the plex the person is looking at, and opens no tab', async () => {
    const one = window()

    await lands(landing({ at: 'plex' }), one.places)

    expect(one.travelled).toStrictEqual(['Note.md'])
    expect(one.shown).toStrictEqual([])
  })
})

describe('a note chosen', () => {
  it('opens under the name it is called by', async () => {
    const one = window()

    await lands(landing({}), one.places)

    expect(one.shown).toStrictEqual(['Note.md A note'])
    expect(one.entered).toStrictEqual([])
  })

  it('is called by the path it is filed at where it has no name', async () => {
    const one = window()

    await lands(landing({ title: '' }), one.places)

    expect(one.shown).toStrictEqual(['Note.md Note.md'])
  })

  it('stands on the line the heading was found at', async () => {
    const one = window()

    await lands(landing({ line: 12 }), one.places)

    expect(one.shown).toStrictEqual(['Note.md A note'])
    expect(one.entered).toStrictEqual(['Note.md 12'])
  })

  it('stands at the top of the note for the line it opens with', async () => {
    const one = window()

    await lands(landing({ line: 0 }), one.places)

    expect(one.entered).toStrictEqual(['Note.md 0'])
  })
})

describe('a passage chosen', () => {
  it('opens the document it stands in, at the stretch it names', async () => {
    const one = window()

    await lands(
      landing({ at: 'document', path: 'library/mahabharata.epub', start: 40_512, length: 31 }),
      one.places,
    )

    expect(one.opened).toStrictEqual(['library/mahabharata.epub 40512 31'])
    expect(one.shown).toStrictEqual([])
  })

  it('opens it at its first page where it names no stretch', async () => {
    const one = window()

    await lands(landing({ at: 'document', path: 'library/mahabharata.epub' }), one.places)

    expect(one.opened).toStrictEqual(['library/mahabharata.epub 0 0'])
  })
})

describe('nothing chosen', () => {
  it('takes the person nowhere at all', async () => {
    const one = window()

    await lands(null, one.places)

    expect([one.travelled, one.opened, one.shown, one.entered]).toStrictEqual([[], [], [], []])
  })
})
