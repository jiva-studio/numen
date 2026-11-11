import { mount } from '@vue/test-utils'
import { describe, expect, it } from 'vitest'
import { defineComponent, h } from 'vue'
import type { ReaderHandle } from '@numen/plugin-sdk'
import { useReaderAttachment } from './useReaderAttachment'

const mountAttached = (hasAttach = true) => {
  const log: string[] = []
  const attached: (ReaderHandle | null)[] = []
  const reader: ReaderHandle = {
    focus: () => log.push('focus'),
    measure: () => log.push('measure'),
  }
  const state = hasAttach ? { attachReader: (handle: ReaderHandle | null) => attached.push(handle) } : {}
  const view = mount(
    defineComponent({
      setup() {
        useReaderAttachment(() => state, reader, () => log.push('detach'))
        return () => h('div')
      },
    }),
  )
  return { view, log, attached, reader }
}

describe('attaching a reader to the window', () => {
  it('focuses the reader and attaches its handle to the tab state at mount', () => {
    const { log, attached, reader } = mountAttached()

    expect(log).toEqual(['focus'])
    expect(attached).toEqual([reader])
  })

  it('takes the handle back and detaches at unmount', () => {
    const { view, log, attached, reader } = mountAttached()

    view.unmount()

    expect(attached).toEqual([reader, null])
    expect(log).toEqual(['focus', 'detach'])
  })

  it('still focuses and detaches where the window takes no handle', () => {
    const { view, log } = mountAttached(false)

    view.unmount()

    expect(log).toEqual(['focus', 'detach'])
  })
})
