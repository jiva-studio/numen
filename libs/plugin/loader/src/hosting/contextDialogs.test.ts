/** The dialogs a plugin opens are closed when the plugin is disposed, and only those. */
import { describe, expect, it } from 'vitest'
import type { DialogHandle, DialogHost } from '@numen/plugin-sdk'
import { createPluginParts } from './context'
import { createTestContextHost } from '../testing/contextHost'

const createDialogs = () => {
  const open = new Set<number>()
  let last = 0
  const host: DialogHost = {
    open: () => {
      const id = ++last
      open.add(id)
      return {
        close: () => {
          open.delete(id)
        },
      } satisfies DialogHandle
    },
    saveText: () => Promise.resolve(undefined),
  }
  return { host, open }
}

const createParts = (pluginId: string, dialogs: DialogHost) =>
  createPluginParts(
    pluginId,
    createTestContextHost({
      ui: { prompts: { getPicker: () => undefined, getDialogs: () => dialogs } },
    }),
    { id: pluginId, name: 'Sample', version: '1.0.0', permissions: [] },
  )

describe('the dialogs of a plugin', () => {
  it('close when the plugin is disposed', () => {
    const { host, open } = createDialogs()
    const parts = createParts('md.numen.tools.sample', host)
    parts.host.ui.dialogs!.open({ component: {} })
    parts.host.ui.dialogs!.open({ component: {} })

    parts.subscriptions.disposeAll()

    expect(open.size).toBe(0)
  })

  it('leave the dialogs of another plugin open', () => {
    const { host, open } = createDialogs()
    const first = createParts('md.numen.tools.first', host)
    const second = createParts('md.numen.tools.second', host)
    first.host.ui.dialogs!.open({ component: {} })
    second.host.ui.dialogs!.open({ component: {} })

    first.subscriptions.disposeAll()

    expect(open.size).toBe(1)
  })

  it('do not close a second time when the plugin closed one itself', () => {
    const { host } = createDialogs()
    let closed = 0
    const counting: DialogHost = {
      ...host,
      open: (options) => {
        const handle = host.open(options)
        return {
          close: () => {
            closed += 1
            handle.close()
          },
        }
      },
    }
    const parts = createParts('md.numen.tools.sample', counting)
    const handle = parts.host.ui.dialogs!.open({ component: {} })
    handle.close()

    parts.subscriptions.disposeAll()

    expect(closed).toBe(1)
  })
})
