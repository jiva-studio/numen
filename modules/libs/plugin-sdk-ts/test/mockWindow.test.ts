import { describe, expect, it } from 'vitest'
import { createMockTabWindow } from '../src/testing/mockWindow'

const register = (window: ReturnType<typeof createMockTabWindow>, canClose = true) => {
  window.register({
    kind: 'sample',
    component: {},
    open: (at) => ({ at }),
    identity: (at) => at,
    onClose: () => canClose,
  })
}

describe('the mock tab window', () => {
  it('opens a tab on what it is given, holding what the kind made', async () => {
    const window = createMockTabWindow()
    register(window)

    const id = await window.openTab('sample', 'a.md')

    expect(id).toBe('sample:a.md')
    expect(window.getTabState(id)).toStrictEqual({ at: 'a.md' })
  })

  it('opens the same thing twice as the tab it has', async () => {
    const window = createMockTabWindow()
    register(window)

    const first = await window.openTab('sample', 'a.md')

    expect(await window.openTab('sample', 'a.md')).toBe(first)
    expect(window.each('sample')).toHaveLength(1)
  })

  it('opens a tab of a kind that declares no identity under one of its own each time', async () => {
    const window = createMockTabWindow()
    window.register({ kind: 'free', component: {}, open: () => ({}) })

    expect(await window.openTab('free')).not.toBe(await window.openTab('free'))
  })

  it('refuses to open a kind nothing registered', async () => {
    await expect(createMockTabWindow().openTab('nothing')).rejects.toThrow('nothing')
  })

  it('lets go of a tab the kind lets go of', async () => {
    const window = createMockTabWindow()
    register(window)
    const id = await window.openTab('sample', 'a.md')

    expect(window.requestClose(id)).toBe(true)
    expect(window.each('sample')).toHaveLength(0)
  })

  it('keeps a tab the kind will close itself until it does', async () => {
    const window = createMockTabWindow()
    register(window, false)
    const id = await window.openTab('sample', 'a.md')

    expect(window.requestClose(id)).toBe(false)
    expect(window.each('sample')).toHaveLength(1)

    window.finishClose(id)
    expect(window.each('sample')).toHaveLength(0)
  })

  it('has nothing to refuse for a tab it does not hold', () => {
    expect(createMockTabWindow().requestClose('sample:none')).toBe(true)
  })
})
