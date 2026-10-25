/**
 * The commands, from the palette and from the keystrokes that reach them.
 *
 * A command is offered over what is in front, asks for what it needs, and is
 * carried out over the note the window holds it by. What each one evaluates to is
 * asked of the window drawn, since the palette is where a person meets it.
 */
import { describe, expect, it } from 'vitest'
// The mocked modules stand before the plugins that reach them are imported.
import {
  requests,
  cards,
  DEBOUNCE,
  mountWindow,
  mountWindowWithPalette,
  nameAnswer,
  passageAnswer,
  said,
  settle,
  sourceAnswer,
} from '@/testing/window'
import { Palette, type PaletteGroup } from '@numen/ui'
import { WORDS } from '@/shared/words'

describe('the palette', () => {
  /** A keystroke taken on the window, and whether the window took it. */
  const pressKey = (key: string) => {
    const event = new KeyboardEvent('keydown', { key, ctrlKey: true, cancelable: true })
    globalThis.dispatchEvent(event)
    return event
  }

  const getGroups = (window: Awaited<ReturnType<typeof mountWindow>>) =>
    (window.findComponent(Palette).props('groups') as readonly { id: string }[]).map(
      (one) => one.id,
    )

  it('opens on the search under its own keystroke', async () => {
    const window = await mountWindow()

    pressKey('k')
    await settle()

    expect(window.findComponent(Palette).props('isOpen')).toBe(true)
    expect(getGroups(window)).toStrictEqual([])
  })

  /** The words typed into the search, with the answers back. */
  const runSearch = async (window: Awaited<ReturnType<typeof mountWindowWithPalette>>) => {
    pressKey('k')
    await settle()
    window.findComponent(Palette).vm.$emit('update:modelValue', 'ani')
    await new Promise((isDone) => setTimeout(isDone, DEBOUNCE))
    await settle()
  }

  /** The mark each row draws, by the name Lucide files it under. */
  const marks = () =>
    [...document.body.querySelectorAll('[data-palette="list"] [role="option"]')].map(
      (row) =>
        /(?:^|\s)lucide-([a-z-]+?)(?:-icon)?(?:\s|$)/.exec(
          row.querySelector('svg')?.getAttribute('class') ?? '',
        )?.[1] ?? '',
    )

  it('draws every kind of note a name turned up as what it is', async () => {
    said.names = [
      nameAnswer('Ants.md', 'Ants'),
      nameAnswer('Animals.md', 'Animals', 'deck'),
      nameAnswer('Animal.md', 'Animal', 'stencil'),
    ]
    const window = await mountWindowWithPalette()

    await runSearch(window)

    expect(marks()).toStrictEqual(['file-text', 'layers', 'layout-template'])
  })

  it('draws a passage as the note it was read out of', async () => {
    said.names = []
    said.passages = [
      passageAnswer('Animals.md', 'Animals', 'deck'),
      passageAnswer('Ants.md', 'Ants'),
    ]
    const window = await mountWindowWithPalette()

    await runSearch(window)

    // Both halves of the search answer with the same two passages here.
    expect(marks()).toStrictEqual(['layers', 'file-text', 'layers', 'file-text'])
  })

  it('draws a passage out of a book and one out of a recording as what each is', async () => {
    said.names = []
    said.passages = [
      sourceAnswer('Ants.epub', 'book'),
      sourceAnswer('730709BG.LON.mp3', 'recording'),
    ]
    const window = await mountWindowWithPalette()

    await runSearch(window)

    expect(marks()).toStrictEqual(['book-open', 'audio-lines', 'book-open', 'audio-lines'])
    // Every row of the list carries a mark, and none of them keeps empty room.
    expect(marks().every(Boolean)).toBe(true)
  })

  /** The groups standing, by the name each carries. */
  const groupTitles = () =>
    [...document.body.querySelectorAll('[data-palette="title"]')].map((one) =>
      one.textContent?.trim(),
    )

  it('draws no group for a search that answered with nothing, and says so once', async () => {
    said.embedded = 4
    const window = await mountWindowWithPalette()

    await runSearch(window)

    expect(groupTitles()).toStrictEqual([WORDS.creating])
    expect(document.body.querySelector('[data-palette="silence"]')).toBeNull()
  })

  it('keeps the group that could not be requested, with what it has to say', async () => {
    said.embedded = 0
    const window = await mountWindowWithPalette()

    await runSearch(window)

    expect(groupTitles()).toStrictEqual([WORDS.creating, WORDS.meaning])
    expect(document.body.querySelector('[data-palette="silence"]')?.textContent?.trim()).toBe(
      WORDS.notEmbedded,
    )
  })
})

/**
 * The keystrokes drawn on a command's row. Each is asked for on the window, as
 * a person presses it with the palette nowhere in sight.
 */
describe('a command reached by its own keystroke', () => {
  const field = () => document.body.querySelector<HTMLInputElement>('[data-palette="field"]')

  /** A keystroke taken on the window, and whether the window took it. */
  const pressKey = (key: string, overrides: Partial<KeyboardEventInit> = {}) => {
    const event = new KeyboardEvent('keydown', {
      key,
      ctrlKey: true,
      cancelable: true,
      ...overrides,
    })
    globalThis.dispatchEvent(event)
    return event
  }

  const type = async (text: string) => {
    const into = field()
    if (!into) return
    into.value = text
    into.dispatchEvent(new Event('input'))
    await settle()
  }

  const press = async (key: string) => {
    field()?.dispatchEvent(new KeyboardEvent('keydown', { key, bubbles: true, cancelable: true }))
    await settle()
  }

  /** The row of a command in the list of commands, by the identity it is drawn under. */
  const getRow = (window: Awaited<ReturnType<typeof mountWindow>>, id: string) => {
    const groups = window.findComponent(Palette).props('groups') as readonly PaletteGroup[]
    return groups.flatMap((group) => group.items).find((one) => one.id === id)
  }

  it('draws the keystroke on its row, written for the keyboard in hand', async () => {
    const window = await mountWindow()

    pressKey('p')
    await settle()

    expect(getRow(window, 'note')?.keys).toEqual({ icons: ['control'], letter: 'N' })
    expect(getRow(window, 'goto')?.keys).toEqual({ icons: ['control'], letter: 'G' })
  })

  it('draws no keystroke on the rows no keystroke reaches', async () => {
    const window = await mountWindow()

    pressKey('p')
    await settle()

    expect(getRow(window, 'destroy')?.keys).toBeUndefined()
    expect(getRow(window, 'eraseVault')?.keys).toBeUndefined()
  })

  it('makes a note under the name typed, on the keystroke the new note draws', async () => {
    await mountWindowWithPalette()

    const event = pressKey('n')
    await settle()
    await type('Entropy')
    await press('Enter')
    await settle()

    expect(event.defaultPrevented).toBe(true)
    expect(requests.made).toStrictEqual(['Entropy'])
  })

  it('opens the step that picks a note, on the keystroke going to one draws', async () => {
    said.names = [nameAnswer('physics/Entropy.md', 'Entropy')]
    const window = await mountWindowWithPalette()

    const event = pressKey('g')
    await settle()

    expect(event.defaultPrevented).toBe(true)
    expect(window.findComponent(Palette).props('crumb')).toBe('Go to a note')
  })

  it('leaves a keystroke alone while Alt is retained with it', async () => {
    const window = await mountWindow()

    const event = pressKey('n', { altKey: true })
    await settle()

    expect(event.defaultPrevented).toBe(false)
    expect(window.findComponent(Palette).props('isOpen')).toBe(false)
    expect(requests.made).toStrictEqual([])
  })

  it('leaves a keystroke a pane has already answered alone', async () => {
    const window = await mountWindow()

    const event = new KeyboardEvent('keydown', { key: 'n', ctrlKey: true, cancelable: true })
    event.preventDefault()
    globalThis.dispatchEvent(event)
    await settle()

    expect(window.findComponent(Palette).props('isOpen')).toBe(false)
  })
})

describe('a command reached by a keystroke holding Shift', () => {
  /** A keystroke taken on the window, and whether the window took it. */
  const pressKey = (key: string, overrides: Partial<KeyboardEventInit> = {}) => {
    const event = new KeyboardEvent('keydown', {
      key,
      ctrlKey: true,
      cancelable: true,
      ...overrides,
    })
    globalThis.dispatchEvent(event)
    return event
  }

  it('no longer puts the palette up on the letter that puts it up alone', async () => {
    const window = await mountWindow()

    const event = pressKey('K', { shiftKey: true })
    await settle()

    expect(event.defaultPrevented).toBe(false)
    expect(window.findComponent(Palette).props('isOpen')).toBe(false)
  })

  it('no longer puts the commands up on the letter that puts them up alone', async () => {
    const window = await mountWindow()

    pressKey('P', { shiftKey: true })
    await settle()

    // The plex the window opened with is the one it is standing on, which is
    // what showing the note in the plex leaves in front.
    expect(window.findComponent(Palette).props('groups')).toStrictEqual([])
  })
})

/**
 * A vault still being read draws no picture, so a command over it is reached by
 * the keyboard: this is the window a person meets while a vault is opening.
 */
describe('a command requested for while the vault is being read', () => {
  const field = () => document.body.querySelector<HTMLInputElement>('[data-palette="field"]')

  const type = async (text: string) => {
    const into = field()
    if (!into) return
    into.value = text
    into.dispatchEvent(new Event('input'))
    await settle()
  }

  const press = async (key: string) => {
    field()?.dispatchEvent(new KeyboardEvent('keydown', { key, bubbles: true, cancelable: true }))
    await settle()
  }

  /** The keystroke for a new note, which is a command over the window. */
  const pressNewNote = async () => {
    globalThis.dispatchEvent(new KeyboardEvent('keydown', { key: 'n', ctrlKey: true }))
    await settle()
  }

  it('carries the command out, and goes on saying the vault is being read', async () => {
    said.isReady = false
    said.opening = null
    const window = await mountWindowWithPalette()

    await pressNewNote()
    await type('Entropy')
    await press('Enter')

    expect(requests.made).toStrictEqual(['Entropy'])
    expect(cards(window)).toStrictEqual(['indexing the vault…'])
  })
})

/** How the window is drawn, walked as a person walks it, with nothing stubbed. */
