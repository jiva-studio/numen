/**
 * The keyboard an open note is owed, asked without a browser.
 *
 * A note that is never handed its keyboard is a note the person typed into
 * somewhere else: the tab is on screen, and what they wrote went nowhere.
 */
import { describe, expect, it } from 'vitest'
import { nextTick } from 'vue'
import { noteKeyboard, ITSELF, type EditorHandle } from './keyboard'

/** An editor that says whether it took what it was handed. */
const editor = (takes = true) => {
  const focused: number[] = []
  const measured: number[] = []
  const drawn: EditorHandle = {
    focus: () => {
      focused.push(ITSELF)
      return takes
    },
    measure: () => measured.push(1),
    reveal: (line: number) => {
      focused.push(line)
      return takes
    },
  }
  return { drawn, focused, measured }
}

describe('a note owed the keyboard', () => {
  it('takes it as soon as an editor is drawn', async () => {
    const owed = noteKeyboard()
    owed.requestFocus('Note.md')
    const drew = editor()

    owed.setEditor('Note.md', drew.drawn)
    await nextTick()

    expect(drew.focused).toEqual([ITSELF])
  })

  it('is revealed at the line it was owed', async () => {
    const owed = noteKeyboard()
    const drew = editor()
    owed.setEditor('Note.md', drew.drawn)

    owed.requestFocus('Note.md', 12)
    await nextTick()

    expect(drew.focused).toEqual([12])
  })

  it('keeps the line it was owed when the note is asked for itself', async () => {
    const owed = noteKeyboard()
    const drew = editor()

    owed.requestFocus('Note.md', 12)
    owed.requestFocus('Note.md')
    owed.setEditor('Note.md', drew.drawn)
    await nextTick()

    expect(drew.focused).toEqual([12])
  })

  it('stays owed while the editor cannot take it', async () => {
    const owed = noteKeyboard()
    const early = editor(false)
    owed.requestFocus('Note.md')
    owed.setEditor('Note.md', early.drawn)
    await nextTick()

    const drew = editor()
    owed.setEditor('Note.md', drew.drawn)
    await nextTick()

    expect(drew.focused).toEqual([ITSELF])
  })

  it('is owed nothing once an editor has taken it', async () => {
    const owed = noteKeyboard()
    const drew = editor()
    owed.requestFocus('Note.md')
    owed.setEditor('Note.md', drew.drawn)
    await nextTick()

    owed.measure('Note.md')

    expect(drew.focused).toEqual([ITSELF])
    expect(drew.measured).toHaveLength(1)
  })

  it('is owed nothing at all once its tab has closed', async () => {
    const owed = noteKeyboard()
    owed.requestFocus('Note.md')
    owed.cancelFocusRequest('Note.md')

    const drew = editor()
    owed.setEditor('Note.md', drew.drawn)
    await nextTick()

    expect(drew.focused).toEqual([])
  })

  it('is owed to the note it was owed to, and to no other', async () => {
    const owed = noteKeyboard()
    const one = editor()
    const other = editor()
    owed.setEditor('One.md', one.drawn)
    owed.setEditor('Other.md', other.drawn)

    owed.requestFocus('Other.md', 3)
    await nextTick()

    expect(one.focused).toEqual([])
    expect(other.focused).toEqual([3])
  })
})

describe('the window drawn at another size', () => {
  it('has every open editor measure again', async () => {
    const owed = noteKeyboard()
    const one = editor()
    const other = editor()
    owed.setEditor('One.md', one.drawn)
    owed.setEditor('Other.md', other.drawn)

    owed.measureAll()

    expect(one.measured).toHaveLength(1)
    expect(other.measured).toHaveLength(1)
  })

  it('has nothing to say to an editor whose tab has let go of it', async () => {
    const owed = noteKeyboard()
    const drew = editor()
    owed.setEditor('Note.md', drew.drawn)
    owed.setEditor('Note.md', null)

    owed.measureAll()

    expect(drew.measured).toEqual([])
  })
})

describe('an editor that goes', () => {
  it('is not handed anything after it has', async () => {
    const owed = noteKeyboard()
    const drew = editor()
    owed.setEditor('Note.md', drew.drawn)
    owed.setEditor('Note.md', null)

    owed.requestFocus('Note.md')
    await nextTick()

    expect(drew.focused).toEqual([])
  })
})
