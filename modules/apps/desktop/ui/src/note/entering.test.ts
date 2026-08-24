/**
 * The keyboard an open note is owed, asked without a browser.
 *
 * A note that is never handed its keyboard is a note the person typed into
 * somewhere else: the tab is on screen, and what they wrote went nowhere.
 */
import { describe, expect, it } from 'vitest'
import { nextTick } from 'vue'
import { entering, ITSELF, type Drawn } from './entering'

/** An editor that says whether it took what it was handed. */
const editor = (takes = true) => {
  const focused: number[] = []
  const measured: number[] = []
  const drawn: Drawn = {
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
    const owed = entering()
    owed.owes('Note.md')
    const drew = editor()

    owed.drew('Note.md', drew.drawn)
    await nextTick()

    expect(drew.focused).toEqual([ITSELF])
  })

  it('is revealed at the line it was owed', async () => {
    const owed = entering()
    const drew = editor()
    owed.drew('Note.md', drew.drawn)

    owed.owes('Note.md', 12)
    await nextTick()

    expect(drew.focused).toEqual([12])
  })

  it('stays owed while the editor cannot take it', async () => {
    const owed = entering()
    const early = editor(false)
    owed.owes('Note.md')
    owed.drew('Note.md', early.drawn)
    await nextTick()

    const drew = editor()
    owed.drew('Note.md', drew.drawn)
    await nextTick()

    expect(drew.focused).toEqual([ITSELF])
  })

  it('is owed nothing once an editor has taken it', async () => {
    const owed = entering()
    const drew = editor()
    owed.owes('Note.md')
    owed.drew('Note.md', drew.drawn)
    await nextTick()

    owed.measure('Note.md')

    expect(drew.focused).toEqual([ITSELF])
    expect(drew.measured).toHaveLength(1)
  })

  it('is owed nothing at all once its tab has closed', async () => {
    const owed = entering()
    owed.owes('Note.md')
    owed.drops('Note.md')

    const drew = editor()
    owed.drew('Note.md', drew.drawn)
    await nextTick()

    expect(drew.focused).toEqual([])
  })

  it('is owed to the note it was owed to, and to no other', async () => {
    const owed = entering()
    const one = editor()
    const other = editor()
    owed.drew('One.md', one.drawn)
    owed.drew('Other.md', other.drawn)

    owed.owes('Other.md', 3)
    await nextTick()

    expect(one.focused).toEqual([])
    expect(other.focused).toEqual([3])
  })
})

describe('an editor that goes', () => {
  it('is not handed anything after it has', async () => {
    const owed = entering()
    const drew = editor()
    owed.drew('Note.md', drew.drawn)
    owed.drew('Note.md', null)

    owed.owes('Note.md')
    await nextTick()

    expect(drew.focused).toEqual([])
  })
})
