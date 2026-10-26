/**
 * The panel of actions over the palette: closed until asked for, closed again
 * when its item offers nothing, and choosing in it never reaches an item that
 * is not lit.
 */
import { describe, expect, it, vi } from 'vitest'
import { nextTick, ref } from 'vue'
import { useActionPanel } from './panel'
import type { PaletteAction } from '../lib/item'

const OPEN: PaletteAction = { id: 'open', text: 'Open the note' }

const setUp = (lit: string | undefined) => {
  const actions = ref<readonly PaletteAction[]>([OPEN])
  const choose = vi.fn<(item: string, action: string) => void>()
  const dismiss = vi.fn<() => void>()
  const panel = useActionPanel({
    getActions: () => actions.value,
    getLit: () => lit,
    choose,
    dismiss,
  })
  return { actions, choose, dismiss, panel }
}

describe('useActionPanel', () => {
  it('remains closed until it is opened', () => {
    expect(setUp('note').panel.open.value).toBe(false)
  })

  it('closes when item has no remaining actions', async () => {
    const { actions, panel } = setUp('note')
    panel.open.value = true
    actions.value = []
    await nextTick()
    expect(panel.open.value).toBe(false)
  })

  it('passes the lit item and the chosen action back', () => {
    const { choose, panel } = setUp('note')
    panel.chooseAction('open')
    expect(choose).toHaveBeenCalledExactlyOnceWith('note', 'open')
  })

  it('chooses nothing while no item is lit', () => {
    const { choose, panel } = setUp(undefined)
    panel.chooseAction('open')
    expect(choose).not.toHaveBeenCalled()
  })

  it('puts the panel away on a press, and leaves the palette standing', () => {
    const { dismiss, panel } = setUp('note')
    panel.open.value = true
    panel.onPress()
    expect(panel.open.value).toBe(false)
    expect(dismiss).not.toHaveBeenCalled()
  })

  it('takes the panel away first on the ground, and the palette on the next press', () => {
    const { dismiss, panel } = setUp('note')
    panel.open.value = true
    panel.onGround()
    expect(panel.open.value).toBe(false)
    expect(dismiss).not.toHaveBeenCalled()
    panel.onGround()
    expect(dismiss).toHaveBeenCalledOnce()
  })
})
