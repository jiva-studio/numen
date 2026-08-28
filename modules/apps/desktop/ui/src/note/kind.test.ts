/**
 * What the window decides about an open note, asked without a browser.
 *
 * The keyboard is the one to watch: a note opened is owed it until there is an
 * editor to take it, and an editor is registered as it is drawn, a moment
 * before it can take anything.
 */
import { describe, expect, it, vi } from 'vitest'
import { nextTick, ref } from 'vue'
import type { PlexShowing } from '@numen/ui'
import { noting, type Called } from './kind'
import { putting } from '../putting'
import type { Drawn } from './entering'
import type { drawn } from './drawn'
import type { editing } from './editing'
import type { State } from './tab'
import { windowing } from '../windowing'

/** A vault that answers with the heading written into each note. */
const vault = (titles: Record<string, string> = {}): Called => ({
  neighbourhood: async (path) => {
    if (titles[path] === undefined) throw new Error('not reached')
    return { focus: { title: titles[path] } }
  },
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
    shown: (id: string) => ({
      path: where(id),
      body: '',
      state: states[where(id)] ?? 'clean',
      refusal: null,
    }),
    where,
    saying: () => '',
    typed: () => {},
    save: () => {},
    keep: () => said.push('keep'),
    take: () => said.push('take'),
  }
  return {
    store: store as unknown as ReturnType<typeof editing>,
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
  return { store: store as unknown as ReturnType<typeof drawn>, shut }
}

/** An editor that says whether it took what it was handed. */
const editor = (takes = true) => {
  const focused: number[] = []
  const drawn: Drawn = {
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
const window = (titles: Record<string, string> = {}, states: Record<string, State> = {}) => {
  const store = notes(states)
  const drawing = drawings()
  const held = windowing()
  const puts = putting({ types: async () => new Map() })
  const noted = noting(vault(titles), store.store, drawing.store, held.host, puts)
  held.declares([noted.kind])
  /** Every note tab the window holds now. */
  const open = () => held.tabs.value.map((tab) => tab.id)
  /**
   * A note put in front of the person. It reaches the tab the only way anything
   * does, which is through the one place a file is opened from.
   */
  const shows = (path: string, title = '', showing: PlexShowing = 'here') =>
    puts.made(path, title, 'note', showing)
  return { noted, held, open, shows, ...store, drawings: drawing }
}

describe('a note opened', () => {
  it('is owed the keyboard until there is an editor to take it', async () => {
    const one = window()
    const held = one.noted.opens('Note.md')
    await nextTick()

    const drew = editor()
    held.drew(drew.drawn)
    await nextTick()

    expect(drew.focused).toEqual([-1])
  })

  it('is revealed at the line it was asked for', async () => {
    const one = window()
    const held = one.noted.opens('Note.md', 12)
    const drew = editor()

    held.drew(drew.drawn)
    await nextTick()

    expect(drew.focused).toEqual([12])
  })

  it('stays owed while the editor could not take it, and is given it again', async () => {
    const one = window()
    const held = one.noted.opens('Note.md')
    const early = editor(false)
    held.drew(early.drawn)
    await nextTick()

    const drew = editor()
    held.drew(drew.drawn)
    await nextTick()

    expect(early.focused).toContain(-1)
    expect(drew.focused).toEqual([-1])
  })

  it('is owed nothing once an editor has taken it', async () => {
    const one = window()
    const held = one.noted.opens('Note.md')
    const drew = editor()
    held.drew(drew.drawn)
    await nextTick()

    held.measure()

    expect(drew.focused).toEqual([-1])
  })

  it('takes the keyboard again when it is asked for while it is already open', async () => {
    const one = window()
    const held = one.noted.opens('Note.md')
    const drew = editor()
    held.drew(drew.drawn)
    await nextTick()

    one.noted.entersAt('Note.md')
    await nextTick()

    expect(drew.focused).toEqual([-1, -1])
  })

  it('is called what it was opened under', () => {
    const one = window()
    one.noted.calls('Deep/Note.md', 'A note')

    expect(one.noted.called('Deep/Note.md')).toBe('A note')
  })

  it('is called by the file it is filed under while nothing has named it', () => {
    const one = window()
    one.noted.opens('Deep/Note.md')

    expect(one.noted.called('Deep/Note.md')).toBe('Deep/Note.md')
  })
})

describe('a note that was renamed', () => {
  it('is shown in the tab already holding it, and no second tab is opened on it', async () => {
    const one = window()
    one.shows('Note.md', 'A note')
    await nextTick()
    const [tab] = one.open()
    one.moves('Note.md', 'Renamed.md')
    await nextTick()

    one.shows('Renamed.md')
    await nextTick()

    expect(one.open()).toEqual([tab])
  })

  it('is called what the window calls it under the name it now has', async () => {
    const one = window()
    one.shows('Note.md', 'A note')
    await nextTick()
    one.moves('Note.md', 'Renamed.md')
    await nextTick()

    one.noted.calls('Renamed.md', 'Renamed')

    expect(one.noted.called('Renamed.md')).toBe('Renamed')
    expect(one.noted.kind.called(one.noted.held(one.idOf('Renamed.md')))).toBe('Renamed')
  })

  it('leaves the name it had free, so a note made under it opens a tab of its own', async () => {
    const one = window()
    one.shows('Foo.md', 'The first')
    await nextTick()
    const first = one.open()[0]
    one.moves('Foo.md', 'Bar.md')
    await nextTick()

    one.noted.calls('Foo.md', 'The second')
    one.shows('Foo.md')
    await nextTick()

    const open = one.open()
    expect(open).toHaveLength(2)
    expect(open[0]).toBe(first)
    expect(one.idOf('Bar.md')).not.toBe(one.idOf('Foo.md'))
    expect(one.noted.called('Bar.md')).toBe('The first')
    expect(one.noted.called('Foo.md')).toBe('The second')
  })

  it('answers to the store under the identity it opened with, at the name it now has', async () => {
    const one = window()
    one.shows('Note.md')
    await nextTick()
    const id = one.noted.holding('Note.md')
    one.moves('Note.md', 'Renamed.md')
    await nextTick()

    expect(one.noted.holding('Renamed.md')).toBe(id)
    expect(one.noted.holding('Note.md')).toBeNull()
    expect(one.noted.held(id ?? '').shown().path).toBe('Renamed.md')
  })

  it('is called by the file it now stands at while nothing has named it', async () => {
    const one = window()
    one.shows('Note.md')
    await nextTick()
    one.moves('Note.md', 'Renamed.md')
    await nextTick()

    expect(one.held.tabs.value.map((tab) => tab.title)).toEqual(['Renamed.md'])
  })

  it('takes the keyboard in the tab holding it, under the name it now has', async () => {
    const one = window()
    const held = one.noted.opens('Note.md')
    const drew = editor()
    held.drew(drew.drawn)
    await nextTick()
    one.moves('Note.md', 'Renamed.md')
    await nextTick()

    one.noted.entersAt('Renamed.md', 4)
    await nextTick()

    expect(drew.focused).toEqual([-1, 4])
  })
})

describe('what a note is called', () => {
  it('is the heading the vault reads out of it once what was typed has landed', async () => {
    const one = window({ 'Note.md': 'What it is about' })
    one.noted.calls('Note.md', 'Untitled note')
    one.noted.opens('Note.md')

    await nextTick()
    await vi.waitFor(() => expect(one.noted.called('Note.md')).toBe('What it is about'))
  })

  it('is the name it had when the vault cannot answer', async () => {
    const one = window()
    one.noted.calls('Note.md', 'Untitled note')
    one.shows('Note.md')

    await nextTick()
    await nextTick()

    expect(one.noted.called('Note.md')).toBe('Untitled note')
  })

  it('is what a note was called before its tab opened, kept by the tab that opens', async () => {
    const one = window()
    one.noted.calls('Made.md', 'A new note')

    one.shows('Made.md')
    await nextTick()

    expect(one.noted.called('Made.md')).toBe('A new note')
    expect(one.held.tabs.value.map((tab) => tab.title)).toEqual(['A new note'])
  })
})

describe('the word a note tab carries', () => {
  it('is what its state is worth', () => {
    const one = window({}, { 'Note.md': 'unsaved', 'Other.md': 'clean' })
    one.noted.opens('Note.md')
    one.noted.opens('Other.md')

    const marked = (path: string) => one.noted.kind.marked?.(one.noted.held(path))

    expect(marked('Note.md')).toBe('unsaved')
    expect(marked('Other.md')).toBeUndefined()
  })
})

describe('the window going', () => {
  it('leaves an open note alone, since the quit is what writes what it owes', () => {
    const one = window()
    const held = one.noted.opens('Note.md')

    one.noted.kind.gone?.(held, 'Note.md')

    expect(one.shut).toEqual([])
    expect(one.drawings.shut).toEqual([])
  })
})

describe('a note tab closing', () => {
  it('writes what it owes, and goes when the note says it is done', async () => {
    const one = window()
    one.shows('Note.md', 'A note')
    await nextTick()
    const [id] = one.open()

    one.held.shut(id ?? '')
    await nextTick()

    expect(one.shut).toEqual(['Note.md'])
    expect(one.drawings.shut).toEqual(['Note.md'])
    await vi.waitFor(() => expect(one.open()).toEqual([]))
    expect(one.noted.called('Note.md')).toBe('Note.md')
  })

  it('stays open while the note is not done with it', async () => {
    const one = window()
    one.shows('Note.md', 'A note')
    await nextTick()
    const [id] = one.open()
    one.holds()

    one.held.shut(id ?? '')
    await nextTick()
    await nextTick()

    expect(one.open()).toEqual([id])
    expect(one.noted.called('Note.md')).toBe('A note')
  })
})

describe('a note the window is told to let go of', () => {
  it('is let go of by the tab holding it, under the identity it opened under', async () => {
    const one = window()
    one.shows('Note.md', 'A note')
    await nextTick()

    one.noted.shuts(one.idOf('Note.md'))
    await nextTick()

    expect(one.shut).toEqual(['Note.md'])
    await vi.waitFor(() => expect(one.open()).toEqual([]))
  })

  it('is let go of at the name it now has, wherever its file went', async () => {
    const one = window()
    one.shows('Note.md', 'A note')
    await nextTick()
    const id = one.idOf('Note.md')
    one.moves('Note.md', 'Moved.md')
    await nextTick()

    one.noted.shuts(id)
    await nextTick()

    expect(one.shut).toEqual(['Moved.md'])
  })

  it('is nothing to a window holding no tab of it', async () => {
    const one = window()
    one.shows('Note.md', 'A note')
    await nextTick()

    one.noted.shuts('never opened')
    await nextTick()

    expect(one.shut).toEqual([])
    expect(one.open()).toHaveLength(1)
  })
})

describe('what a note is called under the identity it opened under', () => {
  it('is what the window calls it, wherever its file went', async () => {
    const one = window()
    one.shows('Note.md', 'A note')
    await nextTick()
    const id = one.idOf('Note.md')
    one.moves('Note.md', 'Moved.md')
    await nextTick()

    expect(one.noted.titled(id)).toBe('A note')
  })

  it('is the file it stands at while nothing has named it', async () => {
    const one = window()
    one.shows('Note.md')
    await nextTick()

    expect(one.noted.titled(one.idOf('Note.md'))).toBe('Note.md')
  })
})
