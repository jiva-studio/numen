/**
 * The commands, from the palette and from the keystrokes that reach them.
 *
 * A command is offered over what is in front, asks for what it needs, and is
 * carried out over the note the window holds it by. What each one comes to is
 * asked of the window drawn, since the palette is where a person meets it.
 */
import { describe, expect, it } from 'vitest'
import { type VueWrapper } from '@vue/test-utils'
import {
  branch,
  Notices,
  pane,
  Palette,
  Tree,
  WorkspaceLayout,
  type PaletteGroup,
} from '@numen/ui'
import AgentTab from './agent/AgentTab.vue'
import DeckTab from './cards/DeckTab.vue'
import FilesTab from './files/FilesTab.vue'
import { NEW_DECK, NEW_STENCIL } from './files/menu'
import NoteTab from './note/NoteTab.vue'
import PlexTab from './plex/PlexTab.vue'
import { WORDS as plexWords } from './plex/words'
import {
  asked,
  cards,
  maker,
  DEBOUNCE,
  drawn,
  drawnWithPalette,
  nameSaid,
  nodeInPlex,
  paneKinds,
  passageSaid,
  said,
  settles,
  sourceSaid,
} from './testing/window'
import { REFUSED, WORDS } from './words'
import { plexCalled } from './tabs/workspace'

describe('the palette', () => {
  /** A keystroke taken on the window, and whether the window took it. */
  const pressed = (key: string) => {
    const event = new KeyboardEvent('keydown', { key, ctrlKey: true, cancelable: true })
    globalThis.dispatchEvent(event)
    return event
  }

  const groupsOf = (window: Awaited<ReturnType<typeof drawn>>) =>
    (window.findComponent(Palette).props('groups') as readonly { id: string }[]).map((one) => one.id)

  it('opens on the commands for what is in front, and prints nothing', async () => {
    const window = await drawn()

    const event = pressed('p')
    await settles()

    expect(event.defaultPrevented).toBe(true)
    expect(window.findComponent(Palette).props('open')).toBe(true)
    expect(groupsOf(window)).toStrictEqual(['note', 'window', 'vault'])
  })

  it('opens on the search under its own keystroke', async () => {
    const window = await drawn()

    pressed('k')
    await settles()

    expect(window.findComponent(Palette).props('open')).toBe(true)
    expect(groupsOf(window)).toStrictEqual([])
  })

  /** A name the search turned up, chosen to be read. */
  const reads = async (window: Awaited<ReturnType<typeof drawn>>, path: string) => {
    pressed('k')
    await settles()
    window.findComponent(Palette).vm.$emit('update:modelValue', 'ani')
    await new Promise((done) => setTimeout(done, DEBOUNCE))
    window.findComponent(Palette).vm.$emit('choose', path, 'note')
    await settles()
    await settles()
  }

  /** The words typed into the search, with the answers back. */
  const searched = async (window: Awaited<ReturnType<typeof drawnWithPalette>>) => {
    pressed('k')
    await settles()
    window.findComponent(Palette).vm.$emit('update:modelValue', 'ani')
    await new Promise((done) => setTimeout(done, DEBOUNCE))
    await settles()
  }

  /** The mark each row draws, by the name Lucide files it under. */
  const marks = () =>
    [...document.body.querySelectorAll('[data-palette="list"] [role="option"]')].map(
      (row) =>
        /lucide-([a-z-]+)-icon/.exec(row.querySelector('svg')?.getAttribute('class') ?? '')?.[1] ??
        '',
    )

  it('draws every kind of note a name turned up as what it is', async () => {
    said.names = [
      nameSaid('Ants.md', 'Ants'),
      nameSaid('Animals.md', 'Animals', 'deck'),
      nameSaid('Animal.md', 'Animal', 'stencil'),
    ]
    const window = await drawnWithPalette()

    await searched(window)

    expect(marks()).toStrictEqual(['file-text', 'layers', 'layout-template'])
  })

  it('draws a passage as the note it was read out of', async () => {
    said.names = []
    said.passages = [passageSaid('Animals.md', 'Animals', 'deck'), passageSaid('Ants.md', 'Ants')]
    const window = await drawnWithPalette()

    await searched(window)

    // Both halves of the search answer with the same two passages here.
    expect(marks()).toStrictEqual(['layers', 'file-text', 'layers', 'file-text'])
  })

  it('draws a passage out of a book and one out of a recording as what each is', async () => {
    said.names = []
    said.passages = [sourceSaid('Ants.epub', 'book'), sourceSaid('730709BG.LON.mp3', 'recording')]
    const window = await drawnWithPalette()

    await searched(window)

    expect(marks()).toStrictEqual(['book-open', 'audio-lines', 'book-open', 'audio-lines'])
    // Every row of the list carries a mark, and none of them keeps empty room.
    expect(marks().every(Boolean)).toBe(true)
  })

  /** The groups standing, by the name each carries. */
  const groupTitles = () =>
    [...document.body.querySelectorAll('[data-palette="title"]')].map((one) => one.textContent?.trim())

  it('draws no group for a search that answered with nothing, and says so once', async () => {
    said.embedded = 4
    const window = await drawnWithPalette()

    await searched(window)

    expect(groupTitles()).toStrictEqual([WORDS.creating])
    expect(document.body.querySelector('[data-palette="silence"]')).toBeNull()
  })

  it('keeps the group that could not be asked, with what it has to say', async () => {
    said.embedded = 0
    const window = await drawnWithPalette()

    await searched(window)

    expect(groupTitles()).toStrictEqual([WORDS.creating, WORDS.meaning])
    expect(document.body.querySelector('[data-palette="silence"]')?.textContent?.trim()).toBe(
      WORDS.notEmbedded,
    )
  })

  it('opens a deck it turned up in the editor of its cards', async () => {
    said.names = [nameSaid('Animals.md', 'Animals', 'deck')]
    said.types = { 'Animals.md': 'deck' }
    const window = await drawn()

    await reads(window, 'Animals.md')

    expect(paneKinds(window).flat()).toContain('deck')
    expect(window.findComponent(NoteTab).exists()).toBe(false)
  })

  it('opens a stencil it turned up in the editor of its fields and faces', async () => {
    said.names = [nameSaid('Animal.md', 'Animal', 'stencil')]
    said.types = { 'Animal.md': 'stencil' }
    const window = await drawn()

    await reads(window, 'Animal.md')

    expect(paneKinds(window).flat()).toContain('stencil')
    expect(window.findComponent(NoteTab).exists()).toBe(false)
  })

  it('opens an ordinary note the same search turned up in a note tab', async () => {
    said.names = [nameSaid('Animals.md', 'Animals')]
    const window = await drawn()

    await reads(window, 'Animals.md')

    expect(paneKinds(window).flat()).not.toContain('deck')
    expect(window.findComponent(NoteTab).exists()).toBe(true)
  })

  it('turns from the search to the commands on the character that means them', async () => {
    const window = await drawn()
    pressed('k')
    await settles()

    window.findComponent(Palette).vm.$emit('update:modelValue', '>')
    await settles()

    expect(groupsOf(window)).toStrictEqual(['note', 'window', 'vault'])
  })

  /** The note the commands are over, which the step that renames one opens on. */
  const overNote = async (window: Awaited<ReturnType<typeof drawn>>) => {
    window.findComponent(Palette).vm.$emit('choose', 'title', 'title')
    await settles()
    return window.findComponent(Palette).props('modelValue')
  }

  /** The tab of the plex standing on that note, as the window calls it. */
  const plexTab = (window: Awaited<ReturnType<typeof drawn>>, note: string): string =>
    (window.findComponent(WorkspaceLayout).props('tabs') as readonly { id: string; title: string }[])
      .find((one) => one.title === plexCalled(plexWords.plex, note))
      ?.id ?? ''

  it('is over the plex in the tab in front, not the plex last put in front', async () => {
    said.names = [nameSaid('physics/Entropy.md', 'Entropy')]
    const window = await drawn()

    // A second plex, standing on a note of its own, put in front last.
    pressed('p')
    await settles()
    window.findComponent(Palette).vm.$emit('choose', 'plex', 'plex')
    await settles()
    pressed('k')
    await settles()
    window.findComponent(Palette).vm.$emit('update:modelValue', 'en')
    await new Promise((done) => setTimeout(done, DEBOUNCE))
    window.findComponent(Palette).vm.$emit('choose', 'physics/Entropy.md', 'plex')
    await settles()

    // Each of the two dragged into a pane of its own. The pane the person is in
    // is the one holding the plex on the note the vault opens with.
    window.findComponent(WorkspaceLayout).vm.$emit('update:modelValue', {
      root: branch(
        'root',
        [
          pane('main', [plexTab(window, 'Root')]),
          pane('aside', [plexTab(window, 'physics/Entropy')]),
        ],
        [0.5, 0.5],
      ),
      axis: 'horizontal',
      focus: 'main',
    })
    await settles()

    pressed('p')
    await settles()

    expect(await overNote(window)).toBe('Root')
  })
})

/**
 * The keystrokes drawn on a command's row. Each is asked for on the window, as
 * a person presses it with the palette nowhere in sight.
 */
describe('a command reached by its own keystroke', () => {
  const field = () => document.body.querySelector<HTMLInputElement>('[data-palette="field"]')

  /** A keystroke taken on the window, and whether the window took it. */
  const pressed = (key: string, over: Partial<KeyboardEventInit> = {}) => {
    const event = new KeyboardEvent('keydown', { key, ctrlKey: true, cancelable: true, ...over })
    globalThis.dispatchEvent(event)
    return event
  }

  const type = async (text: string) => {
    const into = field()
    if (!into) return
    into.value = text
    into.dispatchEvent(new Event('input'))
    await settles()
  }

  const press = async (key: string) => {
    field()?.dispatchEvent(new KeyboardEvent('keydown', { key, bubbles: true, cancelable: true }))
    await settles()
  }

  /** The row of a command in the list of commands, by the identity it is drawn under. */
  const rowOf = (window: Awaited<ReturnType<typeof drawn>>, id: string) => {
    const groups = window.findComponent(Palette).props('groups') as readonly PaletteGroup[]
    return groups.flatMap((group) => group.items).find((one) => one.id === id)
  }

  it('draws the keystroke on its row, written for the keyboard in hand', async () => {
    const window = await drawn()

    pressed('p')
    await settles()

    expect(rowOf(window, 'note')?.keys).toEqual({ icons: ['control'], letter: 'N' })
    expect(rowOf(window, 'goto')?.keys).toEqual({ icons: ['control'], letter: 'G' })
  })

  it('draws no keystroke on the rows no keystroke reaches', async () => {
    const window = await drawn()

    pressed('p')
    await settles()

    expect(rowOf(window, 'destroy')?.keys).toBeUndefined()
    expect(rowOf(window, 'eraseVault')?.keys).toBeUndefined()
  })

  it('makes a note under the name typed, on the keystroke the new note draws', async () => {
    await drawnWithPalette()

    const event = pressed('n')
    await settles()
    await type('Entropy')
    await press('Enter')
    await settles()

    expect(event.defaultPrevented).toBe(true)
    expect(asked.made).toStrictEqual(['Entropy'])
  })

  it('makes a deck under the name typed, and opens it in the editor of its cards', async () => {
    const window = await drawnWithPalette()

    globalThis.dispatchEvent(new KeyboardEvent('keydown', { key: 'p', ctrlKey: true }))
    await settles()
    await type('New deck')
    await press('Enter')
    await type('Animals')
    await press('Enter')
    await settles()

    expect(asked.cards).toStrictEqual(['deck / Animals'])
    expect(paneKinds(window).flat()).toContain('deck')
  })

  it('opens the deck where the vault filed it, and at no path of its own making', async () => {
    const window = await drawnWithPalette()

    globalThis.dispatchEvent(new KeyboardEvent('keydown', { key: 'p', ctrlKey: true }))
    await settles()
    await type('New deck')
    await press('Enter')
    await type('Animals')
    await press('Enter')
    await settles()

    const deck = window.findComponent(DeckTab).props('state') as {
      shown: { value: { path: string } }
    }
    expect(deck.shown.value.path).toBe('Animals.note')
    expect(deck.shown.value.path).not.toBe('Animals.md')
  })

  it('says the refusal and opens nothing where the name is taken already', async () => {
    const window = await drawnWithPalette()

    const makes = async () => {
      globalThis.dispatchEvent(new KeyboardEvent('keydown', { key: 'p', ctrlKey: true }))
      await settles()
      await type('New deck')
      await press('Enter')
      await type('Animals')
      await press('Enter')
      await settles()
    }
    await makes()
    await makes()

    expect(paneKinds(window).flat().filter((kind) => kind === 'deck')).toHaveLength(1)
    expect(window.text()).toContain(REFUSED.occupied)
  })

  it('makes a stencil the same way, and opens no deck', async () => {
    const window = await drawnWithPalette()

    globalThis.dispatchEvent(new KeyboardEvent('keydown', { key: 'p', ctrlKey: true }))
    await settles()
    await type('New stencil')
    await press('Enter')
    await type('Animal')
    await press('Enter')
    await settles()

    expect(asked.cards).toStrictEqual(['stencil / Animal [Field 1]'])
    expect(paneKinds(window).flat()).not.toContain('deck')
  })

  it('makes the stencil carrying the field its cards are named by, and not none', async () => {
    await drawnWithPalette()

    globalThis.dispatchEvent(new KeyboardEvent('keydown', { key: 'p', ctrlKey: true }))
    await settles()
    await type('New stencil')
    await press('Enter')
    await type('Animal')
    await press('Enter')
    await settles()

    expect(asked.cards).not.toStrictEqual(['stencil / Animal []'])
  })

  it('opens the step that picks a note, on the keystroke going to one draws', async () => {
    said.names = [nameSaid('physics/Entropy.md', 'Entropy')]
    const window = await drawnWithPalette()

    const event = pressed('g')
    await settles()

    expect(event.defaultPrevented).toBe(true)
    expect(window.findComponent(Palette).props('crumb')).toBe('Go to a note')
  })

  it('travels to the note picked on that step', async () => {
    said.names = [nameSaid('physics/Entropy.md', 'Entropy')]
    const window = await drawnWithPalette()

    pressed('g')
    await settles()
    await type('en')
    await new Promise((done) => setTimeout(done, DEBOUNCE))
    await press('Enter')
    await settles()

    const plex = window.findComponent(PlexTab).props('state') as {
      view: { here: { value: string } }
    }
    expect(plex.view.here.value).toBe('physics/Entropy.md')
  })

  it('leaves a keystroke alone while Alt is held with it', async () => {
    const window = await drawn()

    const event = pressed('n', { altKey: true })
    await settles()

    expect(event.defaultPrevented).toBe(false)
    expect(window.findComponent(Palette).props('open')).toBe(false)
    expect(asked.made).toStrictEqual([])
  })

  it('leaves a keystroke a pane has already answered alone', async () => {
    const window = await drawn()

    const event = new KeyboardEvent('keydown', { key: 'n', ctrlKey: true, cancelable: true })
    event.preventDefault()
    globalThis.dispatchEvent(event)
    await settles()

    expect(window.findComponent(Palette).props('open')).toBe(false)
  })
})

/**
 * The keystrokes that hold Shift. The letter each holds is spoken for on its
 * own, so what the window does with it turns on Shift alone.
 */
describe('a command reached by a keystroke holding Shift', () => {
  /** A keystroke taken on the window, and whether the window took it. */
  const pressed = (key: string, over: Partial<KeyboardEventInit> = {}) => {
    const event = new KeyboardEvent('keydown', { key, ctrlKey: true, cancelable: true, ...over })
    globalThis.dispatchEvent(event)
    return event
  }

  const tabs = (window: Awaited<ReturnType<typeof drawn>>) =>
    (window.findComponent(WorkspaceLayout).props('tabs') as readonly { id: string }[]) ?? []

  it('no longer puts the palette up on the letter that puts it up alone', async () => {
    const window = await drawn()

    const event = pressed('K', { shiftKey: true })
    await settles()

    expect(event.defaultPrevented).toBe(false)
    expect(window.findComponent(Palette).props('open')).toBe(false)
  })

  it('no longer puts the commands up on the letter that puts them up alone', async () => {
    const window = await drawn()

    pressed('P', { shiftKey: true })
    await settles()

    // The plex the window opened with is the one it is standing on, which is
    // what showing the note in the plex leaves in front.
    expect(window.findComponent(Palette).props('groups')).toStrictEqual([])
  })

  it('opens an agent in a tab of its own', async () => {
    const window = await drawn()
    const before = window.findAllComponents(AgentTab).length

    const event = pressed('A', { shiftKey: true })
    await settles()

    expect(event.defaultPrevented).toBe(true)
    expect(window.findAllComponents(AgentTab).length).toBe(before + 1)
  })

  it('closes the tab in front', async () => {
    const window = await drawn()
    const before = tabs(window).length

    const event = pressed('W', { shiftKey: true })
    await settles()

    expect(event.defaultPrevented).toBe(true)
    expect(tabs(window).length).toBe(before - 1)
  })

  it('makes a child note of the note in front, under the name typed', async () => {
    const window = await drawnWithPalette()

    const event = pressed('C', { shiftKey: true })
    await settles()
    expect(window.findComponent(Palette).props('crumb')).toBe('New child note')

    const field = document.body.querySelector<HTMLInputElement>('[data-palette="field"]')
    if (field) {
      field.value = 'Entropy'
      field.dispatchEvent(new Event('input'))
      await settles()
      field.dispatchEvent(new KeyboardEvent('keydown', { key: 'Enter', bubbles: true }))
      await settles()
    }

    expect(event.defaultPrevented).toBe(true)
    expect(asked.made).toStrictEqual(['Entropy'])
  })

  it('shows the note in front in the plex', async () => {
    const window = await drawn()

    const event = pressed('P', { shiftKey: true })
    await settles()

    expect(event.defaultPrevented).toBe(true)
    const plex = window.findComponent(PlexTab).props('state') as {
      view: { here: { value: string } }
    }
    expect(plex.view.here.value).toBe('Root.md')
  })

  it('draws every keystroke that holds Shift on the row that names it', async () => {
    const window = await drawn()

    pressed('p')
    await settles()

    const groups = window.findComponent(Palette).props('groups') as readonly PaletteGroup[]
    const drawnKeys = Object.fromEntries(
      groups.flatMap((group) => group.items).map((one) => [one.id, one.keys]),
    )
    expect(drawnKeys['travel']).toEqual({ icons: ['control', 'shift'], letter: 'P' })
    expect(drawnKeys['child']).toEqual({ icons: ['control', 'shift'], letter: 'C' })
    expect(drawnKeys['agent']).toEqual({ icons: ['control', 'shift'], letter: 'A' })
    expect(drawnKeys['close']).toEqual({ icons: ['control', 'shift'], letter: 'W' })
  })
})

describe('a command asked for on a node of the plex', () => {
  /** The menu on a node, and an item of it chosen. */
  const chose = async (window: Awaited<ReturnType<typeof drawn>>, id: string) => {
    const plex = window.findComponent(PlexTab).props('state') as {
      asks: (one: unknown) => void
      chose: (id: string) => void
    }
    plex.asks({ node: nodeInPlex(window), at: { x: 0, y: 0 }, from: null, opening: 'below' })
    plex.chose(id)
    await settles()
  }

  it('says in the tab what that tab could not show', async () => {
    const window = await drawn()
    const tab = window.findComponent(PlexTab)
    const state = tab.props('state') as { view: { trouble: { value: string } } }

    state.view.trouble.value = 'Gone.md is not in the vault'
    await settles()

    expect(tab.find('.caution').text()).toBe('Gone.md is not in the vault')
    expect(cards(window)).toStrictEqual([])
  })

  it('says nothing where it was taken up', async () => {
    const window = await drawn()

    await chose(window, 'title')

    expect(cards(window)).toStrictEqual([])
    expect(window.findComponent(Palette).props('crumb')).toBe('Change title')
  })
})

/**
 * A vault still being read draws no picture, so a command over it is reached by
 * the keyboard: this is the window a person meets while a vault is opening.
 */
describe('a command asked for while the vault is being read', () => {
  /** The keystroke for a new note, which is a command over the window. */
  const askedFor = async () => {
    globalThis.dispatchEvent(new KeyboardEvent('keydown', { key: 'n', ctrlKey: true }))
    await settles()
  }

  it('says the vault is still being read, and carries no command out', async () => {
    said.ready = false
    said.opening = null
    const window = await drawn()

    await askedFor()

    expect(cards(window)).toStrictEqual(['reading the vault…', 'The vault is still being read'])
    expect(asked.made).toStrictEqual([])
  })

  it('lets a person put away what it told them, and forgets it', async () => {
    said.ready = false
    said.opening = null
    const window = await drawn()

    await askedFor()
    await window.findAll('article.notice button')[1]!.trigger('click')
    await settles()

    expect(cards(window)).toStrictEqual(['reading the vault…'])
    // Put away is put away for good: the window stops handing the corner a card
    // it has been told the person is finished with.
    expect(window.findComponent(Notices).props('notices')).toHaveLength(1)
  })
})

/** The palette drawn as a person meets it, with nothing of it stubbed. */
/** The note goes to the vault's .trash folder, so nothing is asked over it. */
describe('the keyboard on the command that removes a note', () => {
  const field = () => document.body.querySelector<HTMLInputElement>('[data-palette="field"]')

  const press = async (key: string) => {
    field()?.dispatchEvent(new KeyboardEvent('keydown', { key, bubbles: true, cancelable: true }))
    await settles()
  }

  const type = async (text: string) => {
    const into = field()
    if (!into) return
    into.value = text
    into.dispatchEvent(new Event('input'))
    await settles()
  }

  /** The commands open, with the one that removes a note lit and taken. */
  const overRemove = async () => {
    const window = await drawnWithPalette()
    globalThis.dispatchEvent(new KeyboardEvent('keydown', { key: 'p', ctrlKey: true }))
    await settles()
    await type('remove')
    await press('Enter')
    return window
  }

  it('removes the note the moment the command is chosen', async () => {
    await overRemove()

    expect(asked.removed).toStrictEqual(['Root.md false'])
  })

  it('stands on no step, and the palette goes with the choice', async () => {
    const window = await overRemove()

    expect(document.body.textContent).not.toContain('Keep the note')
    expect(window.findComponent(Palette).props('open')).toBe(false)
  })
})

/** A deck and a stencil answer a removal the way a note does. */
describe('a file the window has open in an editor of cards, removed from the tree', () => {
  const field = () => document.body.querySelector<HTMLInputElement>('[data-palette="field"]')

  const press = async (key: string) => {
    field()?.dispatchEvent(new KeyboardEvent('keydown', { key, bubbles: true, cancelable: true }))
    await settles()
  }

  const type = async (text: string) => {
    const into = field()
    if (!into) return
    into.value = text
    into.dispatchEvent(new Event('input'))
    await settles()
  }

  /** The window with one deck or one stencil made and put in front. */
  const holding = async (command: string, name: string) => {
    const window = await drawnWithPalette()
    globalThis.dispatchEvent(new KeyboardEvent('keydown', { key: 'p', ctrlKey: true }))
    await settles()
    await type(command)
    await press('Enter')
    await type(name)
    await press('Enter')
    await settles()
    return window
  }

  /** A row taken out of the vault, as the tree asks for it. */
  const removes = async (window: VueWrapper, path: string) => {
    window.findComponent(Tree).vm.$emit('remove', [path])
    await settles()
    await settles()
  }

  it('lets go of the tab holding a deck', async () => {
    const window = await holding('New deck', 'Animals')
    expect(paneKinds(window).flat()).toContain('deck')

    await removes(window, 'Animals.note')

    expect(asked.removed).toStrictEqual(['Animals.note false'])
    expect(paneKinds(window).flat()).not.toContain('deck')
  })

  it('lets go of the tab holding a stencil', async () => {
    const window = await holding('New stencil', 'Animal')
    expect(paneKinds(window).flat()).toContain('stencil')

    await removes(window, 'Animal.note')

    expect(asked.removed).toStrictEqual(['Animal.note false'])
    expect(paneKinds(window).flat()).not.toContain('stencil')
  })

  it('writes the card nobody had saved before the file goes', async () => {
    const window = await holding('New deck', 'Animals')
    const state = window.findComponent(DeckTab).props('state') as {
      adds(
        stencil: string,
        values: readonly { field: string; text: string }[],
        section: string | null,
      ): void
    }

    state.adds('Animal', [{ field: 'Name', text: 'Vicuña' }], null)
    await removes(window, 'Animals.note')

    // Making the deck is no write, so the only one is what the person added.
    expect(asked.wrote).toStrictEqual(['Vicuña'])
  })
})

/** The tree makes one where the row stands, and the vault may answer nothing. */
describe('a deck or a stencil the file tree asked the vault for', () => {
  /** What a row of the tree asks for, on the row the menu was opened on. */
  const asksFor = async (stencil: boolean) => {
    const window = await drawn()
    const tree = window.findComponent(FilesTab).props('state') as {
      asks(asked: { path: string | null; at: { x: number; y: number } }): void
      chose(id: string): void
    }
    maker.breaks()
    tree.asks({ path: null, at: { x: 0, y: 0 } })
    tree.chose(stencil ? NEW_STENCIL : NEW_DECK)
    await settles()
    return window
  }

  it('says why no deck was made, where the vault could not be reached', async () => {
    const window = await asksFor(false)

    expect(cards(window).join(' ')).toContain('numen did not answer')
  })

  it('says why no stencil was made, the same way', async () => {
    const window = await asksFor(true)

    expect(cards(window).join(' ')).toContain('numen did not answer')
  })
})

/** How the window is drawn, walked as a person walks it, with nothing stubbed. */
