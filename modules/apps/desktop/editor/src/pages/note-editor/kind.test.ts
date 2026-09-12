/**
 * What the window decides about an open note, asked without a browser.
 *
 * The keyboard is the one to watch: a note opened is owed it until there is an
 * editor to take it, and an editor is registered as it is drawn, a moment
 * before it can take anything.
 */
import { describe, expect, it, vi } from 'vitest'
import { flushPromises } from '@vue/test-utils'
import { nextTick, ref } from 'vue'
import type { PlexDestination } from '@numen/ui'
import { useNoteTab, type NoteTabDeps, type NoteTabState } from './kind'
import { fileOpeners } from '@/entities/tab'
import type { EditorHandle } from './model/keyboard'
import type { noteChanges } from './model/changes'
import type { openNotes, State } from '@/entities/note'
import { useWindowTabs } from '@/entities/tab'
import { NOTE } from '@/entities/tab'

/** A vault that answers with the heading written into each note. */
const vault = (
  titles: Record<string, string> = {},
  notes: Record<string, string> = {},
): NoteTabDeps => ({
  neighbourhood: async (path) => {
    if (titles[path] === undefined) throw new Error('not reached')
    return { focus: { title: titles[path] } }
  },
  resolve: async (_from, written) =>
    new Map(written.filter((one) => notes[one]).map((one) => [one, notes[one]!])),
})

/**
 * The notes of a window, as far as anything here reads them. Each is filed
 * under the identity it opened under, and stands at a file of its own.
 */
const notes = (states: Record<string, State> = {}) => {
  /** The identity of every open note. */
  const open = ref<string[]>([])
  /** Where each open note stands now, under the identity it opened under. */
  const at = ref<Record<string, string>>({})
  const shut: string[] = []
  const said: string[] = []
  let goes = true
  const where = (id: string) => at.value[id] ?? id
  const store = {
    open: (id: string, path: string = id) => {
      if (open.value.includes(id)) return
      open.value = [...open.value, id]
      at.value = { ...at.value, [id]: path }
    },
    shut: async (id: string) => {
      shut.push(where(id))
      if (!goes) return false
      open.value = open.value.filter((one) => one !== id)
      return true
    },
    all: () => open.value,
    has: (id: string) => open.value.includes(id),
    getOpenNote: (id: string) => ({
      path: where(id),
      body: '',
      state: states[where(id)] ?? 'clean',
      error: null,
    }),
    where,
    address: () => null,
    cues: () => [],
    copy: () => '',
    getErrorMessage: () => '',
    typed: () => {},
    save: () => {},
    keep: () => said.push('keep'),
    take: () => said.push('take'),
  }
  return {
    store: store as unknown as ReturnType<typeof openNotes>,
    shut,
    said,
    /** The identity of the note standing at a file, for a test that has its name. */
    idOf: (path: string) => open.value.find((id) => where(id) === path) ?? '',
    /** The note standing at a file moved to another, the way a rename moves one. */
    moves: (path: string, to: string) => {
      const id = open.value.find((one) => where(one) === path)
      if (id) at.value = { ...at.value, [id]: to }
    },
    holds: () => {
      goes = false
    },
  }
}

/** What is being typed, which nothing here reads beyond letting a tab go. */
const drawings = () => {
  const shut: string[] = []
  const store = { shown: () => 0, shut: (path: string) => shut.push(path) }
  return { store: store as unknown as ReturnType<typeof noteChanges>, shut }
}

/** An editor that says whether it took what it was handed. */
const editor = (takes = true) => {
  const focused: number[] = []
  const drawn: EditorHandle = {
    focus: () => {
      focused.push(-1)
      return takes
    },
    measure: () => {},
    reveal: (line: number) => {
      focused.push(line)
      return takes
    },
  }
  return { drawn, focused }
}

/** A window holding notes, and what it draws while it holds them. */
const window = (
  titles: Record<string, string> = {},
  states: Record<string, State> = {},
  reaches: Record<string, string> = {},
) => {
  const store = notes(states)
  const drawing = drawings()
  const held = useWindowTabs()
  const tabOpeners = fileOpeners({
    fileKinds: async (paths) =>
      new Map(paths.map((path) => [path, { kind: 'note' as const, type: 'note' as const }])),
  })
  const noted = useNoteTab(vault(titles, reaches), store.store, drawing.store, held.handle, tabOpeners)
  held.registerKinds([noted.kind])
  /** Every note tab the window holds now. */
  const open = () => held.tabs.value.map((tab) => tab.id)
  /**
   * A note put in front of the person. It reaches the tab the only way anything
   * does, which is through the one place a file is opened from.
   */
  const openNote = (path: string, title = '', how: PlexDestination = 'here') =>
    tabOpeners.openNewFile(path, title, 'note', how)
  return { noted, held, open, openNote, ...store, drawings: drawing }
}

describe('a link in the prose followed', () => {
  /** A window holding one note, and the tab that note stands in. */
  const createNoteWindow = async (reaches: Record<string, string> = {}) => {
    const one = window({}, {}, reaches)
    one.openNote('Note.md')
    await flushPromises()
    return { ...one, state: one.noted.openTab(one.noted.kept.holding('Note.md') ?? 'Note.md') }
  }

  it('opens the note it names, in a tab beside the one it was written in', async () => {
    const one = await createNoteWindow({ 'name://Entropy': 'physics/Entropy.md' })

    one.state.followLink('name://Entropy')
    await flushPromises()

    expect(one.open()).toHaveLength(2)
    expect(one.noted.kept.holding('physics/Entropy.md')).not.toBeNull()
  })

  it('opens nothing where no note answers to it', async () => {
    const one = await createNoteWindow()

    one.state.followLink('name://Nowhere')
    await flushPromises()

    expect(one.open()).toHaveLength(1)
  })

  it('opens nothing for an address that names no note at all', async () => {
    const one = await createNoteWindow({ 'https://example.com': 'physics/Entropy.md' })

    one.state.followLink('https://example.com')
    await flushPromises()

    expect(one.open()).toHaveLength(1)
  })
})

describe('a note opened', () => {
  it('is owed the keyboard until there is an editor to take it', async () => {
    const one = window()
    const state = one.noted.openTab('Note.md')
    await nextTick()

    const drew = editor()
    state.setEditor(drew.drawn)
    await nextTick()

    expect(drew.focused).toEqual([-1])
  })

  it('is revealed at the line it was asked for', async () => {
    const one = window()
    const state = one.noted.openTab('Note.md', 12)
    const drew = editor()

    state.setEditor(drew.drawn)
    await nextTick()

    expect(drew.focused).toEqual([12])
  })

  it('stays owed while the editor could not take it, and is given it again', async () => {
    const one = window()
    const state = one.noted.openTab('Note.md')
    const early = editor(false)
    state.setEditor(early.drawn)
    await nextTick()

    const drew = editor()
    state.setEditor(drew.drawn)
    await nextTick()

    expect(early.focused).toContain(-1)
    expect(drew.focused).toEqual([-1])
  })

  it('is owed nothing once an editor has taken it', async () => {
    const one = window()
    const state = one.noted.openTab('Note.md')
    const drew = editor()
    state.setEditor(drew.drawn)
    await nextTick()

    state.measure()

    expect(drew.focused).toEqual([-1])
  })

  it('takes the keyboard again when it is asked for while it is already open', async () => {
    const one = window()
    const state = one.noted.openTab('Note.md')
    const drew = editor()
    state.setEditor(drew.drawn)
    await nextTick()

    one.noted.focusLine('Note.md')
    await nextTick()

    expect(drew.focused).toEqual([-1, -1])
  })

  it('is called what it was opened under', () => {
    const one = window()
    one.noted.setTitle('Deep/Note.md', 'A note')

    expect(one.noted.getTitle('Deep/Note.md')).toBe('A note')
  })

  it('is called by the file it is filed under while nothing has named it', () => {
    const one = window()
    one.noted.openTab('Deep/Note.md')

    expect(one.noted.getTitle('Deep/Note.md')).toBe('Deep/Note.md')
  })
})

describe('a note that was renamed', () => {
  it('is shown in the tab already holding it, and no second tab is opened on it', async () => {
    const one = window()
    one.openNote('Note.md', 'A note')
    await nextTick()
    const [tab] = one.open()
    one.moves('Note.md', 'Renamed.md')
    await nextTick()

    one.openNote('Renamed.md')
    await nextTick()

    expect(one.open()).toEqual([tab])
  })

  it('is called what the window calls it under the name it now has', async () => {
    const one = window()
    one.openNote('Note.md', 'A note')
    await nextTick()
    one.moves('Note.md', 'Renamed.md')
    await nextTick()

    one.noted.setTitle('Renamed.md', 'Renamed')

    expect(one.noted.getTitle('Renamed.md')).toBe('Renamed')
    expect(one.noted.kind.getTitle?.(one.noted.createNoteTabState(one.idOf('Renamed.md')))).toBe('Renamed')
  })

  it('leaves the name it had free, so a note made under it opens a tab of its own', async () => {
    const one = window()
    one.openNote('Foo.md', 'The first')
    await nextTick()
    const first = one.open()[0]
    one.moves('Foo.md', 'Bar.md')
    await nextTick()

    one.noted.setTitle('Foo.md', 'The second')
    one.openNote('Foo.md')
    await nextTick()

    const open = one.open()
    expect(open).toHaveLength(2)
    expect(open[0]).toBe(first)
    expect(one.idOf('Bar.md')).not.toBe(one.idOf('Foo.md'))
    expect(one.noted.getTitle('Bar.md')).toBe('The first')
    expect(one.noted.getTitle('Foo.md')).toBe('The second')
  })

  it('answers to the store under the identity it opened with, at the name it now has', async () => {
    const one = window()
    one.openNote('Note.md')
    await nextTick()
    const id = one.noted.kept.holding('Note.md')
    one.moves('Note.md', 'Renamed.md')
    await nextTick()

    expect(one.noted.kept.holding('Renamed.md')).toBe(id)
    expect(one.noted.kept.holding('Note.md')).toBeNull()
    expect(one.noted.createNoteTabState(id ?? '').note.value.path).toBe('Renamed.md')
  })

  it('is called by the file it now stands at while nothing has named it', async () => {
    const one = window()
    one.openNote('Note.md')
    await nextTick()
    one.moves('Note.md', 'Renamed.md')
    await nextTick()

    expect(one.held.tabs.value.map((tab) => tab.title)).toEqual(['Renamed.md'])
  })

  it('takes the keyboard in the tab holding it, under the name it now has', async () => {
    const one = window()
    const state = one.noted.openTab('Note.md')
    const drew = editor()
    state.setEditor(drew.drawn)
    await nextTick()
    one.moves('Note.md', 'Renamed.md')
    await nextTick()

    one.noted.focusLine('Renamed.md', 4)
    await nextTick()

    expect(drew.focused).toEqual([-1, 4])
  })
})

describe('what a note is called', () => {
  it('is the heading the vault reads out of it once what was typed has landed', async () => {
    const one = window({ 'Note.md': 'What it is about' })
    one.noted.setTitle('Note.md', 'Untitled note')
    one.noted.openTab('Note.md')

    await nextTick()
    await vi.waitFor(() => expect(one.noted.getTitle('Note.md')).toBe('What it is about'))
  })

  it('is the name it had when the vault cannot answer', async () => {
    const one = window()
    one.noted.setTitle('Note.md', 'Untitled note')
    one.openNote('Note.md')

    await nextTick()
    await nextTick()

    expect(one.noted.getTitle('Note.md')).toBe('Untitled note')
  })

  it('is what a note was called before its tab opened, kept by the tab that opens', async () => {
    const one = window()
    one.noted.setTitle('Made.md', 'A new note')

    one.openNote('Made.md')
    await nextTick()

    expect(one.noted.getTitle('Made.md')).toBe('A new note')
    expect(one.held.tabs.value.map((tab) => tab.title)).toEqual(['A new note'])
  })
})

describe('the word a note tab carries', () => {
  it('is what its state is worth', () => {
    const one = window({}, { 'Note.md': 'unsaved', 'Other.md': 'clean' })
    one.noted.openTab('Note.md')
    one.noted.openTab('Other.md')

    const getMark = (path: string) => one.noted.kind.getMark?.(one.noted.createNoteTabState(path))

    expect(getMark('Note.md')).toBe('unsaved')
    expect(getMark('Other.md')).toBeUndefined()
  })
})

describe('the window going', () => {
  it('leaves an open note alone, since the quit is what writes what it owes', () => {
    const one = window()
    const state = one.noted.openTab('Note.md')

    one.noted.kind.onDestroy?.(state, 'Note.md')

    expect(one.shut).toEqual([])
    expect(one.drawings.shut).toEqual([])
  })
})

describe('a note tab closing', () => {
  it('writes what it owes, and goes when the note says it is done', async () => {
    const one = window()
    one.openNote('Note.md', 'A note')
    await nextTick()
    const [id] = one.open()

    one.held.shut(id ?? '')
    await nextTick()

    expect(one.shut).toEqual(['Note.md'])
    expect(one.drawings.shut).toEqual(['Note.md'])
    await vi.waitFor(() => expect(one.open()).toEqual([]))
    expect(one.noted.getTitle('Note.md')).toBe('Note.md')
  })

  it('stays open while the note is not done with it', async () => {
    const one = window()
    one.openNote('Note.md', 'A note')
    await nextTick()
    const [id] = one.open()
    one.holds()

    one.held.shut(id ?? '')
    await nextTick()
    await nextTick()

    expect(one.open()).toEqual([id])
    expect(one.noted.getTitle('Note.md')).toBe('A note')
  })
})

describe('a note the window is told to let go of', () => {
  it('is let go of by the tab holding it, under the identity it opened under', async () => {
    const one = window()
    one.openNote('Note.md', 'A note')
    await nextTick()

    one.noted.closeTab(one.idOf('Note.md'))
    await nextTick()

    expect(one.shut).toEqual(['Note.md'])
    await vi.waitFor(() => expect(one.open()).toEqual([]))
  })

  it('is let go of at the name it now has, wherever its file went', async () => {
    const one = window()
    one.openNote('Note.md', 'A note')
    await nextTick()
    const id = one.idOf('Note.md')
    one.moves('Note.md', 'Moved.md')
    await nextTick()

    one.noted.closeTab(id)
    await nextTick()

    expect(one.shut).toEqual(['Moved.md'])
  })

  it('is nothing to a window holding no tab of it', async () => {
    const one = window()
    one.openNote('Note.md', 'A note')
    await nextTick()

    one.noted.closeTab('never opened')
    await nextTick()

    expect(one.shut).toEqual([])
    expect(one.open()).toHaveLength(1)
  })
})

describe('what a note is called under the identity it opened under', () => {
  it('is what the window calls it, wherever its file went', async () => {
    const one = window()
    one.openNote('Note.md', 'A note')
    await nextTick()
    const id = one.idOf('Note.md')
    one.moves('Note.md', 'Moved.md')
    await nextTick()

    expect(one.noted.kept.getTitle(id)).toBe('A note')
  })

  it('is the file it stands at while nothing has named it', async () => {
    const one = window()
    one.openNote('Note.md')
    await nextTick()

    expect(one.noted.kept.getTitle(one.idOf('Note.md'))).toBe('Note.md')
  })
})

/** What the one note tab of a window holds. */
const stateOf = (one: ReturnType<typeof window>): NoteTabState =>
  one.held.getTabStateIn<NoteTabState>(one.held.tabs.value[0]?.id ?? '', NOTE)!

describe('what a command asked over a note tab is over', () => {
  it('is the note it holds, at the file it stands at and under the name it carries', async () => {
    const one = window()
    one.openNote('Note.md', 'A note')
    await nextTick()

    expect(one.noted.kind.over!(stateOf(one))).toStrictEqual({
      path: 'Note.md',
      title: 'A note',
    })
  })

  it('is the file it went to, where the note moved under it', async () => {
    const one = window()
    one.openNote('Note.md', 'A note')
    await nextTick()
    one.moves('Note.md', 'Moved.md')
    await nextTick()

    expect(one.noted.kind.over!(stateOf(one)).path).toBe('Moved.md')
  })
})

describe('what a note tab holds, as whoever answers for the person is told it', () => {
  it('is the file it stands at', async () => {
    const one = window()
    one.openNote('Note.md', 'A note')
    await nextTick()

    expect(one.noted.kind.getOpenTab!(stateOf(one))).toStrictEqual({ path: 'Note.md' })
  })
})
