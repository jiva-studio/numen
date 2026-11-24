import { describe, expect, it, vi } from 'vitest'
import {
  createMockDialogHost,
  createMockHostServices,
  createMockPluginContext,
} from '@numen/plugin-sdk/testing'
import { ACCEPT_COMMAND, activate, SHARE_COMMAND } from './index'
import AcceptInvitationDialog from './ui/AcceptInvitationDialog.vue'
import ShareInvitationDialog from './ui/ShareInvitationDialog.vue'

const setup = () => {
  vi.stubGlobal('fetch', () =>
    Promise.resolve(Response.json({ hasIdentity: false, isReachable: true, items: [] })),
  )
  const dialogs = createMockDialogHost()
  const context = createMockPluginContext({ pluginId: 'md.numen.tools.sync' })
  activate(context, createMockHostServices({ dialogs }))
  return { context, dialogs }
}

const getCommand = (context: ReturnType<typeof setup>['context'], id: string) =>
  context.getRegisteredCommands().find((one) => one.id === id)

describe('the command "Share..."', () => {
  it('opens the dialog of the folder it was run on', async () => {
    const { context, dialogs } = setup()

    await getCommand(context, SHARE_COMMAND)?.handler({ path: 'Food' } as never)

    const opened = dialogs.getOpened()
    expect(opened).toHaveLength(1)
    expect(opened.at(0)?.component).toBe(ShareInvitationDialog)
    expect(opened.at(0)?.props).toMatchObject({ state: { target: { folder: 'Food' } } })
    context.dispose()
    vi.unstubAllGlobals()
  })

  it('has one dialog for each run and closes it with the button Done or Cancel', async () => {
    const { context, dialogs } = setup()
    const command = getCommand(context, SHARE_COMMAND)
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

describe('the command "Accept a shared folder..."', () => {
  it('opens the dialog that asks for the link, and closes it with Cancel or Done', async () => {
    const { context, dialogs } = setup()

    await getCommand(context, ACCEPT_COMMAND)?.handler({} as never)

    const opened = dialogs.getOpened()
    expect(opened).toHaveLength(1)
    expect(opened.at(0)?.component).toBe(AcceptInvitationDialog)
    const view = opened.at(0)?.props?.['view'] as { phase: string; close: () => void }
    expect(view.phase).toBe('link')
    view.close()
    expect(dialogs.getOpened()).toHaveLength(0)
    context.dispose()
    vi.unstubAllGlobals()
  })
})
