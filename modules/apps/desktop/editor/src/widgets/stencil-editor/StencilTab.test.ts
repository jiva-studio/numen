/**
 * A stencil tab drawn: where a mark lands, and what a gesture in the editor
 * reaches.
 *
 * What is wrong with a face is drawn inside that face, and what is wrong with a
 * field inside that field's row, so this asks those and not the tab.
 */
// @vitest-environment jsdom
import { enableAutoUnmount, mount } from '@vue/test-utils'
import { afterEach, describe, expect, it } from 'vitest'
import type { Cards, VaultFace, Problem } from '../../entities/deck/cards'
import { fileOpeners } from '../../entities/tab/openers'
import { useWindowTabs } from '../../entities/tab/windowTabs'
import { STENCIL } from '../../entities/tab/workspace'
import StencilTab from './StencilTab.vue'
import { useStencilTabs, type StencilTabState } from './useStencilTabs'
import { WORDS as words } from '../../entities/deck/words'

/** The one place a file is opened from. Nothing here opens one. */
const puts = () => fileOpeners({ fileKinds: async () => new Map() })

/** A moment for whatever the tab asked the vault for to come back. */
const settles = () => new Promise((done) => setTimeout(done, 0))

// Each tab is drawn into the page, so the one before it goes before the next
// stands: a mark is teleported to the row a selector finds in the whole page.
enableAutoUnmount(afterEach)

/** A window with one stencil open, drawn. */
const drawn = async (problems: readonly Problem[] = []) => {
  /** Each field rename the editor asked the vault for. */
  const renamed: string[] = []
  let fields: readonly string[] = ['Height', 'Life span']
  let faces: readonly VaultFace[] = [
    { name: 'Recognise', preamble: '', front: '{{Life span}}', back: '{{Height}}' },
    { name: 'Name it', preamble: '', front: '{{Life span}}', back: '' },
  ]

  const core: Cards = {
    stencils: async () => ({ stencils: [], held: 0 }),
    createDeck: async (title) => ({ path: `${title}.md`, error: null }),
    createStencil: async (title) => ({ path: `${title}.md`, error: null }),
    // The vault writes the name in the fields and in the braces of every face.
    renameField: async (path, from, to) => {
      renamed.push(`${path} ${from} ${to}`)
      fields = fields.map((one) => (one === from ? to : one))
      const braces = (text: string) => text.split(`{{${from}}}`).join(`{{${to}}}`)
      faces = faces.map((face) => ({
        ...face,
        front: braces(face.front),
        back: braces(face.back),
      }))
      return { decks: [], cards: 0, notWritten: [], error: null, changed: false, at: 'renamed' }
    },
    readDeck: async () => ({ deck: null, error: 'missing', at: '', bound: 0 }),
    writeDeck: async () => ({ error: null, changed: false, at: '', bound: 0 }),
    readStencil: async (path) => ({
      stencil: { path, title: 'Animal', fields, preamble: '', faces, tail: '', problems },
      error: null,
      at: 'read',
    }),
    writeStencil: async () => ({ error: null, changed: false, at: 'written' }),
  }

  const held = useWindowTabs()
  const stencils = useStencilTabs(core, held.handle, puts())
  held.declares([stencils.kind])
  const id = await held.opens(STENCIL, 'Animal.md')
  await settles()
  const tab = held.handle.holds<StencilTabState>(STENCIL, id) as StencilTabState
  // A mark is teleported into the face or the row it is about, so the editor
  // has to stand in the document for those to be found.
  const window = mount(StencilTab, { props: { state: tab }, attachTo: document.body })
  await settles()
  return { window, tab, renamed }
}

const missing: Problem = {
  fault: 'faceMissingASide',
  card: null,
  face: 1,
  field: '',
  text: 'this face has no back',
}

const twice: Problem = {
  fault: 'fieldDeclaredTwice',
  card: null,
  face: null,
  field: 'Height',
  text: 'declared twice',
}

describe('a stencil drawn', () => {
  it('draws a row for every field and one for every face', async () => {
    const { window } = await drawn()

    expect(window.findAll('[data-field]')).toHaveLength(2)
    expect(window.findAll('[data-face]')).toHaveLength(2)
  })
})

describe('a mark on a face', () => {
  it('is drawn inside the face it was read against', async () => {
    const { window, tab } = await drawn([missing])
    const second = tab.stencil.value.faces[1]?.id ?? ''

    expect(window.find(`[data-face="${second}"]`).find('[data-wrong]').text()).toBe(
      'this face has no back',
    )
  })

  it('is drawn under the name of that face, where the editor draws it', async () => {
    const { window, tab } = await drawn([missing])
    const second = tab.stencil.value.faces[1]?.id ?? ''

    expect(window.find(`[data-face="${second}"] header [data-wrong]`).text()).toBe(
      'this face has no back',
    )
  })

  it('is drawn on no other face', async () => {
    const { window, tab } = await drawn([missing])
    const first = tab.stencil.value.faces[0]?.id ?? ''

    expect(window.find(`[data-face="${first}"]`).find('[data-wrong]').exists()).toBe(false)
  })

  it('follows the face when another is taken out beside it', async () => {
    const { window, tab } = await drawn([missing])
    const first = tab.stencil.value.faces[0]?.id ?? ''
    const second = tab.stencil.value.faces[1]?.id ?? ''

    tab.removeFace(first)
    await settles()

    expect(window.find(`[data-face="${second}"]`).find('[data-wrong]').text()).toBe(
      'this face has no back',
    )
  })

  it('is nowhere at all where the vault reported nothing wrong', async () => {
    const { window } = await drawn()

    expect(window.findAll('[data-wrong]')).toHaveLength(0)
  })
})

describe('a mark on a field', () => {
  it('is drawn inside the row of the field it names', async () => {
    const { window } = await drawn([twice])

    expect(window.find('[data-field="Height"]').find('[data-wrong]').text()).toBe(
      'declared twice',
    )
  })

  it('is drawn on no other row', async () => {
    const { window } = await drawn([twice])

    expect(window.find('[data-field="Life span"]').find('[data-wrong]').exists()).toBe(false)
  })

  it('is drawn nowhere above the editor, which is for what stands against neither', async () => {
    const { window } = await drawn([twice])

    expect(window.find(`[aria-label="${words.problems}"]`).exists()).toBe(false)
  })
})

describe('a gesture in the editor', () => {
  it('asks the vault to rename the field the box belongs to', async () => {
    const { window, renamed } = await drawn()
    const box = window.find('[data-field="Height"]').find('input')

    await box.setValue('Shoulder')
    await box.trigger('change')
    await settles()

    expect(renamed).toStrictEqual(['Animal.md Height Shoulder'])
  })

  it('shows the field and the braces as the vault left them', async () => {
    const { window, tab } = await drawn()
    const box = window.find('[data-field="Height"]').find('input')

    await box.setValue('Shoulder')
    await box.trigger('change')
    await settles()
    await settles()

    expect(tab.stencil.value.fields).toStrictEqual(['Shoulder', 'Life span'])
    expect(tab.stencil.value.faces[0]?.back).toBe('{{Shoulder}}')
  })

  it('leaves the braces of every other field where they are', async () => {
    const { window, tab } = await drawn()
    const box = window.find('[data-field="Height"]').find('input')

    await box.setValue('Shoulder')
    await box.trigger('change')
    await settles()
    await settles()

    expect(tab.stencil.value.faces[1]?.front).toBe('{{Life span}}')
  })
})
