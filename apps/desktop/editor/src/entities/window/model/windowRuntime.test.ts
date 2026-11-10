import { describe, expect, it, vi } from 'vitest'
import { nextTick, ref } from 'vue'
import { closeWindow, closeWindowWhenEmpty, focusWindow } from './windowRuntime'

describe('focusWindow and closeWindow', () => {
  it('call Focus and Close of the Wails runtime window', async () => {
    const Focus = vi.fn(async () => undefined)
    const Close = vi.fn(async () => undefined)
    const load = async () => ({ Window: { Focus, Close } })

    await focusWindow(load)
    await closeWindow(load)

    expect(Focus).toHaveBeenCalledOnce()
    expect(Close).toHaveBeenCalledOnce()
  })

  it('do nothing when the Wails runtime is not loaded', async () => {
    await expect(focusWindow(async () => undefined)).resolves.toBeUndefined()
    await expect(closeWindow(async () => ({}))).resolves.toBeUndefined()
  })
})

describe('closeWindowWhenEmpty', () => {
  const setup = (count: number) => {
    const isLayoutReady = ref(false)
    const tabCount = ref(count)
    const close = vi.fn(async () => undefined)
    const onError = vi.fn()
    closeWindowWhenEmpty({
      isLayoutReady,
      tabCount: () => tabCount.value,
      closeWindow: close,
      onError,
    })
    return { isLayoutReady, tabCount, close, onError }
  }

  it('closes the window after the layout is ready and the last tab closes', async () => {
    const { isLayoutReady, tabCount, close } = setup(1)

    isLayoutReady.value = true
    await nextTick()
    expect(close).not.toHaveBeenCalled()

    tabCount.value = 0
    await nextTick()
    expect(close).toHaveBeenCalledOnce()
  })

  it('does not close the window before the layout is ready', async () => {
    const { tabCount, close } = setup(1)

    tabCount.value = 0
    await nextTick()

    expect(close).not.toHaveBeenCalled()
  })

  it('reports an error when the close fails', async () => {
    const { isLayoutReady, close, onError } = setup(0)
    const failure = new Error('no runtime')
    close.mockRejectedValueOnce(failure)

    isLayoutReady.value = true
    await vi.waitFor(() => expect(onError).toHaveBeenCalledWith(failure))
  })
})
