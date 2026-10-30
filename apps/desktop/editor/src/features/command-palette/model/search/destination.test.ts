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
  const openedTab: string[] = []
  const displayedItems: string[] = []
  const places: DestinationDeps = {
    travel: async (path) => {
      travelled.push(path)
    },
    openFile: (path, title, line, selector) => {
      if (selector) {
        openedTab.push(`${path} ${JSON.stringify(selector)}`)
        return
      }
      displayedItems.push(`${path} ${title}${line === undefined ? '' : ` ${line}`}`)
    },
  }
  return { places, travelled, opened: openedTab, openedFiles: displayedItems }
}

const landing = (overrides: Partial<SearchDestination>): SearchDestination => ({
  at: 'file',
  path: 'Note.md',
  title: 'A note',
  ...overrides,
})

describe('a name chosen', () => {
  it('is put in front where the plugin that shows notes puts it, and opens no tab', async () => {
    const one = window()

    await openDestination(landing({ at: 'locate' }), one.places)

    expect(one.travelled).toStrictEqual(['Note.md'])
    expect(one.openedFiles).toStrictEqual([])
  })
})

describe('a file chosen', () => {
  it('opens under the name it is called by, naming no editor of its own', async () => {
    const one = window()

    await openDestination(landing({}), one.places)

    expect(one.openedFiles).toStrictEqual(['Note.md A note'])
  })

  it('is called by the path it is filed at where it has no name', async () => {
    const one = window()

    await openDestination(landing({ title: '' }), one.places)

    expect(one.openedFiles).toStrictEqual(['Note.md Note.md'])
  })

  it('carries the line the heading was found at', async () => {
    const one = window()

    await openDestination(landing({ line: 12 }), one.places)

    expect(one.openedFiles).toStrictEqual(['Note.md A note 12'])
  })

  it('carries the top of the file for the line it opens with', async () => {
    const one = window()

    await openDestination(landing({ line: 0 }), one.places)

    expect(one.openedFiles).toStrictEqual(['Note.md A note 0'])
  })
})

describe('a passage chosen', () => {
  it('opens the file it was found in at the selector it names', async () => {
    const one = window()

    await openDestination(landing({ path: 'library/a.bin', selector: { page: 3 } }), one.places)

    expect(one.opened).toStrictEqual(['library/a.bin {"page":3}'])
    expect(one.openedFiles).toStrictEqual([])
  })
})

describe('nothing chosen', () => {
  it('takes the person nowhere at all', async () => {
    const one = window()

    await openDestination(null, one.places)

    expect([one.travelled, one.opened, one.openedFiles]).toStrictEqual([[], [], []])
  })
})
