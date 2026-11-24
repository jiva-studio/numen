import { describe, expect, it, vi } from 'vitest'
import {
  createMockDialogHost,
  createMockHostServices,
  createMockPluginContext,
} from '@numen/plugin-sdk/testing'
import { activate, SHARE_COMMAND } from './index'
import ShareDialog from './ui/ShareDialog.vue'

const setup = () => {
  vi.stubGlobal('fetch', () =>
    Promise.resolve(Response.json({ hasIdentity: false, isReachable: true, items: [] })),
  )
  const dialogs = createMockDialogHost()
  const context = createMockPluginContext({ pluginId: 'md.numen.tools.sync' })
  activate(context, createMockHostServices({ dialogs }))
  return { context, dialogs }
}

describe('the command "Share..."', () => {
  it('opens the dialog of the folder it was run on', async () => {
    const { context, dialogs } = setup()
    const command = context.getRegisteredCommands().find((one) => one.id === SHARE_COMMAND)

    await command?.handler({ path: 'Food' } as never)

    const opened = dialogs.getOpened()
    expect(opened).toHaveLength(1)
    expect(opened.at(0)?.component).toBe(ShareDialog)
    expect(opened.at(0)?.props).toMatchObject({ state: { target: { folder: 'Food' } } })
    context.dispose()
    vi.unstubAllGlobals()
  })

  it('has one dialog for each run and closes it with the button Done or Cancel', async () => {
    const { context, dialogs } = setup()
    const command = context.getRegisteredCommands().find((one) => one.id === SHARE_COMMAND)
    await command?.handler({ path: 'Food' } as never)
    await command?.handler({ path: 'Drinks' } as never)
    expect(dialogs.getOpened()).toHaveLength(2)

    const state = dialogs.getOpened().at(0)?.props?.['state'] as { close: () => void }
    state.close()

    expect(dialogs.getOpened()).toHaveLength(1)
    context.dispose()
    vi.unstubAllGlobals()
  })
})
