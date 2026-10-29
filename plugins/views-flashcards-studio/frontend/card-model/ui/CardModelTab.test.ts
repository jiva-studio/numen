/**
 * A card model tab rendered: where a mark appears, and what a gesture in the
 * editor triggers.
 *
 * A problem with a template is drawn inside that template, and a problem with a
 * field inside that field's row, so this tests those and not the tab.
 */
// @vitest-environment jsdom
import { enableAutoUnmount, mount } from '@vue/test-utils'
import { afterEach, describe, expect, it } from 'vitest'
import { asFailure, asValue } from '@numen/plugin-sdk'
import type { Cards, VaultTemplate, DeckProblem } from '../../cards'
import { createMockBufferHost, createMockTabWindow } from '@numen/plugin-sdk/testing'
import CardModelTab from './CardModelTab.vue'
import { CARD_MODEL, createCardModelTabs, type CardModelTabState } from '../kind'
import { WORDS as words } from '../../cards'

/** Waits for the vault requests made by the tab to complete. */
const settle = () => new Promise((isDone) => setTimeout(isDone, 0))

// Each tab is rendered into the page, so the previous one is unmounted before the
// next is mounted: a mark is teleported to the row a selector finds in the whole
// page.
enableAutoUnmount(afterEach)

/** A window with one card model open, drawn. */
const mountCardModel = async (problems: readonly DeckProblem[] = []) => {
  /** Each field rename the editor requested from the vault. */
  const renameCalls: string[] = []
  let fields: readonly string[] = ['Height', 'Life span']
  let templates: readonly VaultTemplate[] = [
    { name: 'Recognise', preamble: '', front: '{{Life span}}', back: '{{Height}}' },
    { name: 'Name it', preamble: '', front: '{{Life span}}', back: '' },
  ]

  const core: Cards = {
    cardModels: async () => ({ cardModels: [], held: 0 }),
    createDeck: async (title) => ({ ok: true, value: { path: `${title}.md` } }),
    createCardModel: async (title) => ({ ok: true, value: { path: `${title}.md` } }),
    // The vault writes the name in the fields and in the braces of every template.
    renameField: async (path, field, _seen) => {
      const { from, to } = field
      renameCalls.push(`${path} ${from} ${to}`)
      fields = fields.map((one) => (one === from ? to : one))
      const rewriteBraces = (text: string) => text.split(`{{${from}}}`).join(`{{${to}}}`)
      templates = templates.map((template) => ({
        ...template,
        front: rewriteBraces(template.front),
        back: rewriteBraces(template.back),
      }))
      return asValue({ decks: [], cards: 0, notWritten: [], at: 'renamed' })
    },
    readDeck: async () => asFailure({ code: 'missing' as const, bound: 0 }),
    writeDeck: async () => asValue({ at: '' }),
    readCardModel: async (path) =>
      asValue({
        cardModel: { path, title: 'Animal', fields, preamble: '', templates, tail: '', problems },
        at: 'read',
      }),
    writeCardModel: async () => asValue({ at: 'written' }),
  }

  const windowTabs = createMockTabWindow()
  const cardModels = createCardModelTabs(core, {
    buffers: createMockBufferHost(),
    closeTab: windowTabs.finishClose,
  })
  windowTabs.register(cardModels.kind)
  const id = await windowTabs.openTab(CARD_MODEL, 'Animal.md')
  await settle()
  const tab = windowTabs.getTabState<CardModelTabState>(id) as CardModelTabState
  // A mark is teleported into the template or the row it is about, so the editor
  // has to stand in the document for those to be found.
  const window = mount(CardModelTab, { props: { state: tab }, attachTo: document.body })
  await settle()
  return { window, tab, renamed: renameCalls }
}

const missing: DeckProblem = {
  kind: 'templateMissingASide',
  card: null,
  template: 1,
  field: '',
  text: 'this template has no back',
}

const twice: DeckProblem = {
  kind: 'fieldDeclaredTwice',
  card: null,
  template: null,
  field: 'Height',
  text: 'declared twice',
}

describe('a card model drawn', () => {
  it('draws a row for every field and one for every template', async () => {
    const { window } = await mountCardModel()

    expect(window.findAll('[data-field]')).toHaveLength(2)
    expect(window.findAll('[data-template]')).toHaveLength(2)
  })
})

describe('a mark on a template', () => {
  it('is drawn inside the template it was read against', async () => {
    const { window, tab } = await mountCardModel([missing])
    const second = tab.cardModel.value.templates[1]?.id ?? ''

    expect(window.find(`[data-template="${second}"]`).find('[data-wrong]').text()).toBe(
      'this template has no back',
    )
  })

  it('is drawn under the name of that template, where the editor draws it', async () => {
    const { window, tab } = await mountCardModel([missing])
    const second = tab.cardModel.value.templates[1]?.id ?? ''

    expect(window.find(`[data-template="${second}"] header [data-wrong]`).text()).toBe(
      'this template has no back',
    )
  })

  it('is drawn on no other template', async () => {
    const { window, tab } = await mountCardModel([missing])
    const first = tab.cardModel.value.templates[0]?.id ?? ''

    expect(window.find(`[data-template="${first}"]`).find('[data-wrong]').exists()).toBe(false)
  })

  it('follows the template when another is taken out alongside it', async () => {
    const { window, tab } = await mountCardModel([missing])
    const first = tab.cardModel.value.templates[0]?.id ?? ''
    const second = tab.cardModel.value.templates[1]?.id ?? ''

    tab.removeTemplate(first)
    await settle()

    expect(window.find(`[data-template="${second}"]`).find('[data-wrong]').text()).toBe(
      'this template has no back',
    )
  })

  it('is nowhere at all where the vault reported nothing wrong', async () => {
    const { window } = await mountCardModel()

    expect(window.findAll('[data-wrong]')).toHaveLength(0)
  })
})

describe('a mark on a field', () => {
  it('is drawn inside the row of the field it names', async () => {
    const { window } = await mountCardModel([twice])

    expect(window.find('[data-field="Height"]').find('[data-wrong]').text()).toBe('declared twice')
  })

  it('is drawn on no other row', async () => {
    const { window } = await mountCardModel([twice])

    expect(window.find('[data-field="Life span"]').find('[data-wrong]').exists()).toBe(false)
  })

  it('is drawn nowhere above the editor, which is for what remains against neither', async () => {
    const { window } = await mountCardModel([twice])

    expect(window.find(`[aria-label="${words.problems}"]`).exists()).toBe(false)
  })
})

describe('a gesture in the editor', () => {
  it('asks the vault to rename the field the box belongs to', async () => {
    const { window, renamed } = await mountCardModel()
    const box = window.find('[data-field="Height"]').find('input')

    await box.setValue('Shoulder')
    await box.trigger('change')
    await settle()

    expect(renamed).toStrictEqual(['Animal.md Height Shoulder'])
  })

  it('shows the field and the braces as the vault left them', async () => {
    const { window, tab } = await mountCardModel()
    const box = window.find('[data-field="Height"]').find('input')

    await box.setValue('Shoulder')
    await box.trigger('change')
    await settle()
    await settle()

    expect(tab.cardModel.value.fields).toStrictEqual(['Shoulder', 'Life span'])
    expect(tab.cardModel.value.templates[0]?.back).toBe('{{Shoulder}}')
  })

  it('leaves the braces of every other field where they are', async () => {
    const { window, tab } = await mountCardModel()
    const box = window.find('[data-field="Height"]').find('input')

    await box.setValue('Shoulder')
    await box.trigger('change')
    await settle()
    await settle()

    expect(tab.cardModel.value.templates[1]?.front).toBe('{{Life span}}')
  })
})
