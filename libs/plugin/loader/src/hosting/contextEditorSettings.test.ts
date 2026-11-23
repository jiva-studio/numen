/**
 * The editor settings of the application as a plugin context offers them: read-only, with no
 * permission, and followed until the plugin is disposed. The host is the stand-in of the testing
 * folder; every plugin here is invented.
 */
import { describe, expect, it, vi } from 'vitest'
import type { EditorSettings } from '@numen/plugin-sdk'
import { createPluginParts } from './context'
import { createTestContextHost } from '../testing/contextHost'

const MANIFEST = (permissions: readonly string[]) => ({
  id: 'md.numen.sample',
  name: 'Sample',
  version: '1.0.0',
  permissions,
})

const WRAPPED: EditorSettings = { lineNumbers: false, softWrap: true }

describe('the editor settings of a plugin context', () => {
  it('return the editor settings of the host to a plugin that has no permission', () => {
    const host = createTestContextHost({ settings: { getEditorSettings: () => WRAPPED } })

    const context = createPluginParts('md.numen.sample', host, MANIFEST([]))

    expect(context.plugin.settings.getEditor()).toStrictEqual(WRAPPED)
  })

  it('are undefined while the host holds none', () => {
    const host = createTestContextHost({ settings: { getEditorSettings: () => undefined } })

    const context = createPluginParts('md.numen.sample', host, MANIFEST([]))

    expect(context.plugin.settings.getEditor()).toBeUndefined()
  })

  it('follow the host: a read after the settings arrived returns them', () => {
    const held: { value: EditorSettings | undefined } = { value: undefined }
    const host = createTestContextHost({ settings: { getEditorSettings: () => held.value } })
    const context = createPluginParts('md.numen.sample', host, MANIFEST([]))
    expect(context.plugin.settings.getEditor()).toBeUndefined()

    held.value = { lineNumbers: true, softWrap: false }

    expect(context.plugin.settings.getEditor()).toStrictEqual({
      lineNumbers: true,
      softWrap: false,
    })
  })

  it('reach the handler of a plugin, named as the plugin, when the host reports a change', async () => {
    const subscribed: {
      id: string
      handler: (settings: EditorSettings) => void | Promise<void>
    }[] = []
    const host = createTestContextHost({
      settings: {
        watchEditorSettings: (id, handler) => {
          subscribed.push({ id, handler })
          return { dispose: () => undefined }
        },
      },
    })
    const context = createPluginParts('md.numen.sample', host, MANIFEST([]))
    const handler = vi.fn()

    context.plugin.settings.onEditorChange(handler)
    await subscribed[0]?.handler({ lineNumbers: true, softWrap: true })

    expect(subscribed.map((one) => one.id)).toStrictEqual(['md.numen.sample'])
    expect(handler).toHaveBeenCalledExactlyOnceWith({ lineNumbers: true, softWrap: true })
  })

  it('stop being followed when the plugin disposes the subscription', () => {
    const stopped = vi.fn()
    const host = createTestContextHost({
      settings: { watchEditorSettings: () => ({ dispose: stopped }) },
    })
    const context = createPluginParts('md.numen.sample', host, MANIFEST([]))

    const subscription = context.plugin.settings.onEditorChange(vi.fn())
    expect(stopped).not.toHaveBeenCalled()
    subscription.dispose()

    expect(stopped).toHaveBeenCalledOnce()
  })

  it('are followed through a subscription the plugin lists with its others', () => {
    const host = createTestContextHost({})
    const context = createPluginParts('md.numen.sample', host, MANIFEST([]))

    context.plugin.settings.onEditorChange(vi.fn())

    expect(context.plugin.identity.subscriptions).toHaveLength(1)
  })
})
