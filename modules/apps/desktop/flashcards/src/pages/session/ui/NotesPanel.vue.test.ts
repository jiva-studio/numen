// @vitest-environment jsdom
import { mount } from '@vue/test-utils'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import { nextTick, ref } from 'vue'

import NotesPanel from './NotesPanel.vue'
import { useNotesPanel } from '../model/notes'
import { WORDS as words } from '../lib/notesWords'
import type { DeckNeighbourhood, Neighbour } from '../api/notes'
import type { NotesPanelState } from '../model/notes'

/** One note as the window hands it over. */
const createNeighbour = (more: Partial<Neighbour> = {}): Neighbour => ({
  written: 'Leaf mould',
  path: 'notes/Leaf mould.md',
  title: 'Leaf mould',
  body: 'Compost made of fallen leaves alone.',
  label: '',
  points: true,
  ambiguous: false,
  error: '',
  ...more,
})

/** The panel over one deck, with the session around it standing in for it. */
const createPanel = (around: DeckNeighbourhood): NotesPanelState => {
  const open = ref(false)
  return useNotesPanel({
    open: () => open.value,
    showPanel: (up) => {
      open.value = up
    },
    vault: () => 'one',
    deck: () => 'decks/Words.md',
    getDeckNeighbourhood: async () => around,
    showNotice: () => {},
  })
}

/** The panel, opened and drawn. */
const mountPanel = async (around: DeckNeighbourhood) => {
  const panel = createPanel(around)
  await panel.openPanel()
  return mount(NotesPanel, { props: { held: panel } })
}

describe('the panel the deck is read in', () => {
  it('names every note it read, one under another', async () => {
    const one = await mountPanel({
      notes: [createNeighbour(), createNeighbour({ title: 'Humus', path: 'notes/Humus.md' })],
      unread: 0,
    })
    expect(one.findAll('.reading__name').map((it) => it.text())).toEqual(['Leaf mould', 'Humus'])
  })

  // What the deck was made from and what has since been hung off it are read
  // differently, so which way the link runs is said.
  it('says which of them point at the deck rather than being pointed at', async () => {
    const one = await mountPanel({
      notes: [
        createNeighbour(),
        createNeighbour({ title: 'Humus', path: 'notes/Humus.md', points: false }),
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
    const one = await mountPanel({ notes: [createNeighbour({ ambiguous: true })], unread: 0 })
    expect(one.text()).toContain(words.ambiguous)
    expect(one.text()).not.toContain(words.dangling)
  })

  it('says why a note it could reach has no text', async () => {
    const one = await mountPanel({
      notes: [createNeighbour({ body: '', error: 'that note is not in the vault' })],
      unread: 0,
    })
    expect(one.text()).toContain('that note is not in the vault')
  })

  // The bound is where a person can see it, rather than a list that quietly
  // stops.
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

  // The panel comes in while the notes are still being asked for, so what was
  // named is waited for rather than looked for once and given up on.
  it('scrolls to it once the notes have arrived', async () => {
    const panel = createPanel({
      notes: [createNeighbour(), createNeighbour({ title: 'Humus', path: 'notes/Humus.md' })],
      unread: 0,
    })
    const one = mount(NotesPanel, { props: { held: panel } })

    await panel.openPanel('notes/Humus.md')
    await nextTick()
    await nextTick()

    expect(scrolled).toHaveBeenCalledTimes(1)
    expect(one.findAll('.reading__note')[1]?.element).toBe(scrolled.mock.instances[0])
    // What was sought has been scrolled to, and the next card does not seek it.
    expect(panel.at.value).toBe('')
  })
})
