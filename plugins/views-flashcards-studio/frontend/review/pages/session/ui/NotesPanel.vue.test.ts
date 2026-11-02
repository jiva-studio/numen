// @vitest-environment jsdom
import { mount } from '@vue/test-utils'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import { nextTick, ref } from 'vue'

import NotesPanel from './NotesPanel.vue'
import { useNotesPanel } from '../model/notes'
import { WORDS as words } from '../lib/notesWords'
import type { DeckNeighbourhood, Neighbour } from '../api/notes'
import type { NotesPanelState } from '../model/notes'

/** One note as the window receives it. */
const createNeighbour = (more: Partial<Neighbour> = {}): Neighbour => ({
  written: 'Leaf mould',
  path: 'notes/Leaf mould.md',
  title: 'Leaf mould',
  body: 'Compost made of fallen leaves alone.',
  label: '',
  hasPoints: true,
  isAmbiguous: false,
  error: '',
  ...more,
})

/** The panel over one deck, with the session around it standing in for it. */
const createPanel = (neighbourhood: DeckNeighbourhood): NotesPanelState => {
  const open = ref(false)
  return useNotesPanel({
    open: () => open.value,
    showPanel: (up) => {
      open.value = up
    },
    vault: () => 'one',
    deck: () => 'decks/Words.md',
    getDeckNeighbourhood: async () => neighbourhood,
    showNotice: () => undefined,
    resolveLinks: async () => [],
  })
}

/** The panel, opened and drawn. */
const mountPanel = async (neighbourhood: DeckNeighbourhood) => {
  const panel = createPanel(neighbourhood)
  await panel.openPanel()
  return mount(NotesPanel, { props: { panel } })
}

describe('the panel the deck is read in', () => {
  it('names every note it read, one under another', async () => {
    const one = await mountPanel({
      notes: [createNeighbour(), createNeighbour({ title: 'Humus', path: 'notes/Humus.md' })],
      unread: 0,
    })
    expect(one.findAll('.reading__name').map((it) => it.text())).toEqual(['Leaf mould', 'Humus'])
  })

  // The notes the deck was made from and the notes that were later linked to it
  // are read differently, so the direction of the link is reported.
  it('says which of them point at the deck and not being pointed at', async () => {
    const one = await mountPanel({
      notes: [
        createNeighbour(),
        createNeighbour({ title: 'Humus', path: 'notes/Humus.md', hasPoints: false }),
      ],
      unread: 0,
    })
    expect(one.text()).toContain(words.pointsHere)
    expect(one.findAll('.reading__note')[0]?.text()).not.toContain(words.pointsHere)
  })

  // A vault where a name has come loose is a vault with a question in it, and
  // hiding the question answers it wrongly.
  it('names a link that reached no note, and reads nothing under it', async () => {
    const one = await mountPanel({
      notes: [createNeighbour({ path: '', title: '', body: '', written: 'Mould' })],
      unread: 0,
    })
    expect(one.find('.reading__name').text()).toBe('Mould')
    expect(one.text()).toContain(words.dangling)
  })

  it('says where a name several notes answer to was read as the nearest', async () => {
    const one = await mountPanel({ notes: [createNeighbour({ isAmbiguous: true })], unread: 0 })
    expect(one.text()).toContain(words.ambiguous)
    expect(one.text()).not.toContain(words.dangling)
  })

  it('says why a note it could reach has no text', async () => {
    const one = await mountPanel({
      notes: [createNeighbour({ body: '', error: 'note not in the vault' })],
      unread: 0,
    })
    expect(one.text()).toContain('note not in the vault')
  })

  // The limit is visible to the user and is not a silently truncated list.
  it('says how many at the end it did not read', async () => {
    const one = await mountPanel({ notes: [createNeighbour()], unread: 4 })
    expect(one.text()).toContain(words.named(4))
  })

  it('says quietly that a deck is joined to nothing', async () => {
    const one = await mountPanel({ notes: [], unread: 0 })
    expect(one.text()).toContain(words.nothing)
  })
})

describe('a reading opened on one note', () => {
  // jsdom does no scrolling of its own, so this is what is watched for.
  const scrolled = vi.fn()
  beforeEach(() => {
    scrolled.mockClear()
    Element.prototype.scrollIntoView = scrolled
  })

  // The panel opens while the notes are still being requested, so the requested
  // note is awaited and not looked up once and abandoned.
  it('scrolls to it once the notes have arrived', async () => {
    const panel = createPanel({
      notes: [createNeighbour(), createNeighbour({ title: 'Humus', path: 'notes/Humus.md' })],
      unread: 0,
    })
    const one = mount(NotesPanel, { props: { panel } })

    await panel.openPanel('notes/Humus.md')
    await nextTick()
    await nextTick()

    expect(scrolled).toHaveBeenCalledTimes(1)
    expect(one.findAll('.reading__note')[1]?.element).toBe(scrolled.mock.instances[0])
    // What was sought has been scrolled to, and the next card does not seek it.
    expect(panel.col.value).toBe('')
  })
})
