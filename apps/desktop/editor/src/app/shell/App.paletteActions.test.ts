/**
 * The actions of a row of the search, chosen on the mounted window.
 *
 * The row that creates a note is drawn when a search found nothing, and each of its actions is
 * the id of a command that makes the note under the words typed.
 */
import { describe, expect, it } from 'vitest'
import { Palette, type PaletteGroup } from '@numen/shell'
import {
  DEBOUNCE,
  mountWindowWithPalette,
  requests,
  settle,
  sourceAnswer,
  vaultState,
} from '@/testing/window'

/** The id of the group and of the item of the row that creates a note. */
const CREATE_NOTE_ROW = 'creating'

type Window = Awaited<ReturnType<typeof mountWindowWithPalette>>

/** The search opened, with the words typed and the answers back. */
const search = async (window: Window, text: string) => {
  globalThis.dispatchEvent(new KeyboardEvent('keydown', { key: 'k', ctrlKey: true }))
  await settle()
  window.findComponent(Palette).vm.$emit('update:modelValue', text)
  await new Promise((isDone) => setTimeout(isDone, DEBOUNCE))
  await settle()
}

const getGroups = (window: Window) =>
  window.findComponent(Palette).props('groups') as readonly PaletteGroup[]

const choose = async (window: Window, item: string, action: string) => {
  window.findComponent(Palette).vm.$emit('choose', item, action)
  await settle()
  await settle()
}

describe('the row that creates a note, chosen on the window', () => {
  it('draws the command that makes the note alone where no note is in front', async () => {
    vaultState.names = []
    const window = await mountWindowWithPalette()

    await search(window, 'Entropy')

    const row = getGroups(window).find((one) => one.id === CREATE_NOTE_ROW)?.items[0]
    expect(row?.actions?.map((one) => one.id)).toStrictEqual(['app:note.create'])
  })

  it('makes one note under the words typed on the choice of app:note.create, and closes the search', async () => {
    vaultState.names = []
    const window = await mountWindowWithPalette()
    await search(window, '  Entropy  ')

    await choose(window, CREATE_NOTE_ROW, 'app:note.create')

    expect(requests.made).toStrictEqual(['Entropy'])
    expect(window.findComponent(Palette).props('isOpen')).toBe(false)
    expect(window.findComponent(Palette).props('crumb')).toBe('')
  })

  it('makes no note on the choice of an action that is no command of the row', async () => {
    vaultState.names = []
    const window = await mountWindowWithPalette()
    await search(window, 'Entropy')

    await choose(window, CREATE_NOTE_ROW, 'creating')

    expect(requests.made).toStrictEqual([])
  })
})

describe('the row of a passage of a document, chosen on the window', () => {
  it('draws the command that opens the document as its one action', async () => {
    vaultState.names = []
    vaultState.passages = [sourceAnswer('Ants.bin', { page: 3 })]
    const window = await mountWindowWithPalette()

    await search(window, 'ants')

    const actions = getGroups(window)
      .flatMap((one) => one.items)
      .flatMap((one) => one.actions ?? [])
    expect(actions.length).toBeGreaterThan(0)
    expect(actions.map((one) => one.id)).toStrictEqual(actions.map(() => 'app:file.open'))
  })

  it('closes the search once and writes no message on the choice of that command', async () => {
    vaultState.names = []
    vaultState.passages = [sourceAnswer('Ants.bin', { page: 3 })]
    const window = await mountWindowWithPalette()
    await search(window, 'ants')
    const item = getGroups(window).find((one) => one.items.length > 0)?.items[0]?.id ?? ''

    await choose(window, item, 'app:file.open')

    expect(window.findComponent(Palette).props('isOpen')).toBe(false)
    expect(document.body.querySelector('[data-palette="silence"]')).toBeNull()
    expect(requests.made).toStrictEqual([])
  })
})
