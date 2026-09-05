/**
 * What an open note is called, asked without a browser.
 *
 * A tab under the wrong name is a person opening the note they did not mean:
 * the title is written into the note itself, and it changes as they type.
 */
import { describe, expect, it, vi } from 'vitest'
import { nextTick, ref } from 'vue'
import { naming, type NamingDeps } from './naming'
import type { editing } from './editing'
import type { State } from './tab'

/** A vault that answers with the heading written into each note. */
const vault = (titles: Record<string, string> = {}): NamingDeps => ({
  neighbourhood: async (path) => {
    if (titles[path] === undefined) throw new Error('not reached')
    return { focus: { title: titles[path] } }
  },
})

/**
 * The notes of a window, each in the state the test puts it in and at the file
 * the test moved it to.
 */
const notes = () => {
  const states = ref<Record<string, State>>({})
  const at = ref<Record<string, string>>({})
  const store = {
    all: () => Object.keys(states.value),
    shown: (path: string) => ({ state: states.value[path] ?? 'loading' }),
    where: (path: string) => at.value[path] ?? path,
  }
  return {
    store: store as unknown as ReturnType<typeof editing>,
    stands: (path: string, state: State) => {
      states.value = { ...states.value, [path]: state }
    },
    moves: (path: string, to: string) => {
      at.value = { ...at.value, [path]: to }
    },
  }
}

describe('what a note is called', () => {
  it('is the path it is filed at while nothing has named it', () => {
    const names = naming(vault(), notes().store)

    expect(names.called('Deep/Note.md')).toBe('Deep/Note.md')
  })

  it('is what the window called it', () => {
    const names = naming(vault(), notes().store)

    names.calls('Deep/Note.md', 'A note')

    expect(names.called('Deep/Note.md')).toBe('A note')
  })

  it('is the heading the vault reads out of it once what was typed has landed', async () => {
    const store = notes()
    const names = naming(vault({ 'Note.md': 'What it is about' }), store.store)
    names.calls('Note.md', 'Untitled note')

    store.stands('Note.md', 'clean')
    await nextTick()

    await vi.waitFor(() => expect(names.called('Note.md')).toBe('What it is about'))
  })

  it('is not asked for again while the note is still being written', async () => {
    const store = notes()
    const names = naming(vault({ 'Note.md': 'What it is about' }), store.store)
    names.calls('Note.md', 'Untitled note')

    store.stands('Note.md', 'unsaved')
    await nextTick()
    await nextTick()

    expect(names.called('Note.md')).toBe('Untitled note')
  })

  it('is the name it had when the vault cannot answer', async () => {
    const store = notes()
    const names = naming(vault(), store.store)
    names.calls('Note.md', 'Untitled note')

    store.stands('Note.md', 'clean')
    await nextTick()
    await nextTick()

    expect(names.called('Note.md')).toBe('Untitled note')
  })

  it('is asked for again at the file a note moved to, and follows the rename', async () => {
    const store = notes()
    const said = vault({ 'Note.md': 'What it is about', 'Renamed.md': 'Renamed' })
    const names = naming(said, store.store)
    store.stands('Note.md', 'clean')
    await nextTick()
    await vi.waitFor(() => expect(names.called('Note.md')).toBe('What it is about'))

    store.moves('Note.md', 'Renamed.md')
    await nextTick()

    await vi.waitFor(() => expect(names.called('Note.md')).toBe('Renamed'))
  })

  it('is asked for at the file a note moved to once it settles there', async () => {
    const store = notes()
    const names = naming(vault({ 'Renamed.md': 'Renamed' }), store.store)
    names.calls('Note.md', 'Untitled note')
    store.stands('Note.md', 'unsaved')
    store.moves('Note.md', 'Renamed.md')
    await nextTick()
    expect(names.called('Note.md')).toBe('Untitled note')

    store.stands('Note.md', 'clean')
    await nextTick()

    await vi.waitFor(() => expect(names.called('Note.md')).toBe('Renamed'))
  })

  it('is forgotten with the note, so a name is not left behind it', () => {
    const names = naming(vault(), notes().store)
    names.calls('Note.md', 'A note')

    names.forgets('Note.md')

    expect(names.called('Note.md')).toBe('Note.md')
  })
})
