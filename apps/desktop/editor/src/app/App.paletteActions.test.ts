/**
 * The actions of a row of the search, chosen on the mounted window.
 *
 * The actions that create a note are offered on every row, and each of them is the id of a command
 * that makes the note under the words typed.
 */
import { describe, expect, it } from 'vitest'
import { Palette, type PaletteGroup } from '@numen/workbench'
import {
  DEBOUNCE,
  mountWindowWithPalette,
  nameAnswer,
  requests,
  settle,
  sourceAnswer,
  vaultState,
} from '@/testing/window'

type Window = Awaited<ReturnType<typeof mountWindowWithPalette>>

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

const ROW = 'Ants.md'

describe('the actions that create a note, chosen on the window', () => {
  it('follows the action of a row with the command that makes the note alone where no note is in front', async () => {
    vaultState.names = [nameAnswer(ROW, 'Ants')]
    const window = await mountWindowWithPalette()

    await search(window, 'Entropy')

    const row = getGroups(window).flatMap((one) => one.items)[0]
    expect(
      row?.actions?.map((one) => one.id).filter((id) => id.startsWith('app:note.create')),
    ).toStrictEqual(['app:note.create'])
  })

  it('makes one note under the words typed on the choice of app:note.create, and closes the search', async () => {
    vaultState.names = [nameAnswer(ROW, 'Ants')]
    const window = await mountWindowWithPalette()
    await search(window, '  Entropy  ')

    await choose(window, ROW, 'app:note.create')

    expect(requests.made).toStrictEqual(['Entropy'])
    expect(window.findComponent(Palette).props('isOpen')).toBe(false)
    expect(window.findComponent(Palette).props('crumb')).toBe('')
  })

  it('makes no note on the choice of an action that is no command of the row', async () => {
    vaultState.names = [nameAnswer(ROW, 'Ants')]
    const window = await mountWindowWithPalette()
    await search(window, 'Entropy')

    await choose(window, ROW, 'creating')

    expect(requests.made).toStrictEqual([])
  })
})

describe('the row of a passage of a document, chosen on the window', () => {
  it('draws the command that opens the document as its first action', async () => {
    vaultState.names = []
    vaultState.searchResults = [sourceAnswer('Ants.bin', { page: 3 })]
    const window = await mountWindowWithPalette()

    await search(window, 'ants')

    const actions = getGroups(window)
      .flatMap((one) => one.items)
      .map((one) => one.actions?.[0]?.id)
    expect(actions.length).toBeGreaterThan(0)
    expect(actions).toStrictEqual(actions.map(() => 'app:file.open#document'))
  })

  it('closes the search once and writes no message on the choice of that command', async () => {
    vaultState.names = []
    vaultState.searchResults = [sourceAnswer('Ants.bin', { page: 3 })]
    const window = await mountWindowWithPalette()
    await search(window, 'ants')
    const item = getGroups(window).find((one) => one.items.length > 0)?.items[0]?.id ?? ''

    await choose(window, item, 'app:file.open')

    expect(window.findComponent(Palette).props('isOpen')).toBe(false)
    expect(document.body.querySelector('[data-palette="silence"]')).toBeNull()
    expect(requests.made).toStrictEqual([])
  })
})
