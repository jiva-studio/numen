import { describe, expect, it, vi } from 'vitest'
import { createMockHostServices, createMockPluginContext } from '@numen/plugin-sdk/testing'
import { activate, SHARE_COMMAND, SHARE_KIND } from './index'

const setup = () => {
  vi.stubGlobal('fetch', () =>
    Promise.resolve(Response.json({ hasIdentity: false, isReachable: true, items: [] })),
  )
  const opened: [string, string | undefined][] = []
  const closed: string[] = []
  const host = createMockHostServices({
    openTab: (kind, path) => {
      opened.push([kind, path])
      return Promise.resolve(`${kind}-tab`)
    },
    closeTab: (id) => closed.push(id),
  })
  const context = createMockPluginContext({ pluginId: 'md.numen.tools.sync' })
  activate(context, host)
  return { context, opened, closed }
}

describe('the command "Share..."', () => {
  it('opens the dialog of the folder it was run on', async () => {
    const { context, opened } = setup()
    const command = context.getRegisteredCommands().find((one) => one.id === SHARE_COMMAND)

    await command?.handler({ path: 'Food' } as never)

    expect(opened).toStrictEqual([[SHARE_KIND, 'Food']])
    context.dispose()
    vi.unstubAllGlobals()
  })

  it('has one dialog for each folder and closes it on Escape', () => {
    const { context, closed } = setup()
    const kind = context.getRegisteredTabKinds().find((one) => one.kind === SHARE_KIND)
    const state = kind?.open?.('Food', 'tab-1')

    expect(kind?.identity?.('Food')).not.toBe(kind?.identity?.('Drinks'))
    expect(kind?.onKeyPress?.(state as never, new KeyboardEvent('keydown', { key: 'a' }))).toBe(
      false,
    )
    expect(
      kind?.onKeyPress?.(state as never, new KeyboardEvent('keydown', { key: 'Escape' })),
    ).toBe(true)
    expect(closed).toStrictEqual(['tab-1'])
    context.dispose()
    vi.unstubAllGlobals()
  })
})
