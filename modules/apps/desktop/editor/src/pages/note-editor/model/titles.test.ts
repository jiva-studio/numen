/**
 * What an open note is called, asked without a browser.
 *
 * A tab under the wrong name is a person opening the note they did not mean:
 * the title is written into the note itself, and it changes as they type.
 */
import { describe, expect, it, vi } from 'vitest'
import { nextTick, ref } from 'vue'
import { noteTitles, type NoteTitlesDeps } from './titles'
import type { openNotes, State } from '@/entities/note'

/** A vault that answers with the heading written into each note. */
const vault = (titles: Record<string, string> = {}): NoteTitlesDeps => ({
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
    getOpenIds: () => Object.keys(states.value),
    getOpenNote: (path: string) => ({ state: states.value[path] ?? 'loading' }),
    getPath: (path: string) => at.value[path] ?? path,
  }
  return {
    store: store as unknown as ReturnType<typeof openNotes>,
    setState: (path: string, state: State) => {
      states.value = { ...states.value, [path]: state }
    },
    moveNote: (path: string, to: string) => {
      at.value = { ...at.value, [path]: to }
    },
  }
}

describe('what a note is called', () => {
  it('is the path it is filed at while nothing has named it', () => {
    const names = noteTitles(vault(), notes().store)

    expect(names.getTitle('Deep/Note.md')).toBe('Deep/Note.md')
  })

  it('is what the window called it', () => {
    const names = noteTitles(vault(), notes().store)

    names.setTitle('Deep/Note.md', 'A note')

    expect(names.getTitle('Deep/Note.md')).toBe('A note')
  })

  it('is the heading the vault reads out of it once what was typed has landed', async () => {
    const store = notes()
    const names = noteTitles(vault({ 'Note.md': 'What it is about' }), store.store)
    names.setTitle('Note.md', 'Untitled note')

    store.setState('Note.md', 'clean')
    await nextTick()

    await vi.waitFor(() => expect(names.getTitle('Note.md')).toBe('What it is about'))
  })

  it('is not asked for again while the note is still being written', async () => {
    const store = notes()
    const names = noteTitles(vault({ 'Note.md': 'What it is about' }), store.store)
    names.setTitle('Note.md', 'Untitled note')

    store.setState('Note.md', 'unsaved')
    await nextTick()
    await nextTick()

    expect(names.getTitle('Note.md')).toBe('Untitled note')
  })

  it('is the name it had when the vault cannot answer', async () => {
    const store = notes()
    const names = noteTitles(vault(), store.store)
    names.setTitle('Note.md', 'Untitled note')

    store.setState('Note.md', 'clean')
    await nextTick()
    await nextTick()

    expect(names.getTitle('Note.md')).toBe('Untitled note')
  })

  it('is asked for again at the file a note moved to, and follows the rename', async () => {
    const store = notes()
    const said = vault({ 'Note.md': 'What it is about', 'Renamed.md': 'Renamed' })
    const names = noteTitles(said, store.store)
    store.setState('Note.md', 'clean')
    await nextTick()
    await vi.waitFor(() => expect(names.getTitle('Note.md')).toBe('What it is about'))

    store.moveNote('Note.md', 'Renamed.md')
    await nextTick()

    await vi.waitFor(() => expect(names.getTitle('Note.md')).toBe('Renamed'))
  })

  it('is asked for at the file a note moved to once it settles there', async () => {
    const store = notes()
    const names = noteTitles(vault({ 'Renamed.md': 'Renamed' }), store.store)
    names.setTitle('Note.md', 'Untitled note')
    store.setState('Note.md', 'unsaved')
    store.moveNote('Note.md', 'Renamed.md')
    await nextTick()
    expect(names.getTitle('Note.md')).toBe('Untitled note')

    store.setState('Note.md', 'clean')
    await nextTick()

    await vi.waitFor(() => expect(names.getTitle('Note.md')).toBe('Renamed'))
  })

  it('is forgotten with the note, so a name is not left behind it', () => {
    const names = noteTitles(vault(), notes().store)
    names.setTitle('Note.md', 'A note')

    names.forgetTab('Note.md')

    expect(names.getTitle('Note.md')).toBe('Note.md')
  })
})
