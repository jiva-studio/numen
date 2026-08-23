/**
 * What the window decides about an open note, asked without a browser.
 *
 * The keyboard is the one to watch: a note opened is owed it until there is an
 * editor to take it, and an editor is registered as it is drawn, a moment
 * before it can take anything.
 */
import { describe, expect, it, vi } from 'vitest'
import { nextTick, ref } from 'vue'
import { noting, type Called, type Drawn } from './kind'
import type { drawn } from '../drawn'
import type { editing } from '../editing'
import type { State } from '../tab'

/** A vault that answers with the heading written into each note. */
const vault = (titles: Record<string, string> = {}): Called => ({
  neighbourhood: async (path) => {
    if (titles[path] === undefined) throw new Error('not reached')
    return { focus: { title: titles[path] } }
  },
})

/** The notes of a window, as far as anything here reads them. */
const notes = (states: Record<string, State> = {}) => {
  const open = ref<string[]>([])
  const shut: string[] = []
  const said: string[] = []
  let goes = true
  const store = {
    open: (path: string) => {
      if (!open.value.includes(path)) open.value = [...open.value, path]
    },
    shut: async (path: string) => {
      shut.push(path)
      return goes
    },
    all: () => open.value,
    shown: (path: string) => ({ path, body: '', state: states[path] ?? 'clean', refusal: null }),
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

/** A window with its notes, and what it was told to close. */
const window = (titles: Record<string, string> = {}, states: Record<string, State> = {}) => {
  const store = notes(states)
  const drawing = drawings()
  const closed: string[] = []
  const noted = noting(vault(titles), store.store, drawing.store, {
    closes: (path) => closed.push(path),
    makes: async () => 'Made.md',
  })
  return { noted, closed, ...store, drawings: drawing }
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
    one.noted.opens('Note.md')

    await nextTick()
    await nextTick()

    expect(one.noted.called('Note.md')).toBe('Untitled note')
  })
})

describe('the word a note tab carries', () => {
  it('is what its state is worth', () => {
    const one = window({}, { 'Note.md': 'unsaved', 'Other.md': 'clean' })
    one.noted.opens('Note.md')
    one.noted.opens('Other.md')

    expect(one.noted.marked('Note.md')).toBe('unsaved')
    expect(one.noted.marked('Other.md')).toBeUndefined()
  })
})

describe('a note tab closing', () => {
  it('writes what it owes, and the window closes it when the note has gone', async () => {
    const one = window()
    one.noted.calls('Note.md', 'A note')
    const held = one.noted.opens('Note.md')

    held.shuts()
    await nextTick()

    expect(one.shut).toEqual(['Note.md'])
    expect(one.drawings.shut).toEqual(['Note.md'])
    await vi.waitFor(() => expect(one.closed).toEqual(['Note.md']))
    expect(one.noted.called('Note.md')).toBe('Note.md')
  })

  it('stays open while the note is not done with it', async () => {
    const one = window()
    one.noted.calls('Note.md', 'A note')
    const held = one.noted.opens('Note.md')
    one.holds()

    held.shuts()
    await nextTick()
    await nextTick()

    expect(one.closed).toEqual([])
    expect(one.noted.called('Note.md')).toBe('A note')
  })
})
