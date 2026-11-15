/**
 * The commands the plugins declare as the window reads them, and a run as the
 * window dispatches it and reads its result.
 */
import { describe, expect, it, vi } from 'vitest'
import { COMMAND_ERROR_CODE } from '@numen/core-client'
import { createAnsweredClients } from '@/testing/coreAnswers'
import { createDeclaredCommandsApi } from './declaredCommands'

const request = {
  listCommands: vi.fn(),
  runPluginCommand: vi.fn(),
  listCommandItems: vi.fn(),
}

const { listDeclaredCommands, listDeclaredItemLists, runDeclaredCommand, runDeclaredHandler } =
  createDeclaredCommandsApi(
    createAnsweredClients({
      pluginCommands: {
        ...request,
        listCommands: async (...asked: unknown[]) => {
          const answer = (await request.listCommands(...asked)) as {
            slots?: { entry?: object }[]
          }
          // A message from the core always carries its maps and lists, empty where it has none.
          return {
            contentTypeByExtension: {},
            handlers: [],
            keybindings: [],
            ...answer,
            slots: (answer.slots ?? []).map((one) =>
              one.entry ? { ...one, entry: { values: {}, ...one.entry } } : one,
            ),
          }
        },
      },
    }).pluginCommands,
  )

const proofread = {
  pluginId: 'org.example.ink',
  id: 'ink.proofread',
  title: 'Proofread text',
  when: 'version.isLive',
  hasItems: true as const,
  arguments: [],
}

const convert = {
  pluginId: 'md.numen.pages',
  id: 'convert',
  title: 'Convert',
  when: "target.contentType == 'application/pdf'",
  arguments: [],
}

describe('the slot entries the plugins declare', () => {
  it('are read with what each places, its look, its group, its order and its condition', async () => {
    request.listCommands.mockResolvedValue({
      commands: [],
      slots: [
        {
          pluginId: 'org.example.ledger',
          entry: {
            slot: 'fileTree.context',
            placed: { case: 'submenu', value: 'ledger.new' },
            title: 'New ledger',
            icon: 'plus',
            group: '0-new',
            order: 0,
            when: 'isFolder && !isMultiple',
          },
        },
        {
          pluginId: 'org.example.ledger',
          entry: {
            slot: 'titleBar.status',
            placed: { case: 'view', value: 'balance' },
            title: '',
            icon: '',
            group: '',
            when: '',
          },
        },
      ],
    })

    const { slots } = await listDeclaredCommands()

    expect(slots).toStrictEqual([
      {
        pluginId: 'org.example.ledger',
        slot: 'fileTree.context',
        submenu: 'ledger.new',
        title: 'New ledger',
        iconName: 'plus',
        group: '0-new',
        order: 0,
        when: 'isFolder && !isMultiple',
      },
      { pluginId: 'org.example.ledger', slot: 'titleBar.status', view: 'balance' },
    ])
  })
})

describe('the values of a slot entry', () => {
  it('are read with the entry, and absent where the entry holds none', async () => {
    request.listCommands.mockResolvedValue({
      commands: [],
      slots: [
        {
          pluginId: 'org.example.ledger',
          entry: {
            slot: 'titleBar.commands',
            placed: { case: 'command', value: 'app:file.open' },
            title: '',
            icon: '',
            group: '',
            when: '',
            values: { placement: 'split' },
          },
        },
        {
          pluginId: 'org.example.ledger',
          entry: {
            slot: 'titleBar.commands',
            placed: { case: 'command', value: 'ledger.sum' },
            title: '',
            icon: '',
            group: '',
            when: '',
            values: {},
          },
        },
      ],
    })

    const { slots } = await listDeclaredCommands()

    expect(slots).toStrictEqual([
      {
        pluginId: 'org.example.ledger',
        slot: 'titleBar.commands',
        command: 'app:file.open',
        values: { placement: 'split' },
      },
      { pluginId: 'org.example.ledger', slot: 'titleBar.commands', command: 'ledger.sum' },
    ])
  })
})

describe('the commands the plugins declare', () => {
  it('are read with their arguments in order', async () => {
    request.listCommands.mockResolvedValue({
      commands: [
        {
          pluginId: 'org.example.sample',
          id: 'create-sample',
          title: 'Create a sample',
          arguments: [
            { name: 'title', title: 'Title', isRequired: true },
            { name: 'folder', title: 'Folder', isRequired: false },
          ],
        },
      ],
      slots: [],
    })

    const {
      commands: [one],
    } = await listDeclaredCommands()

    expect(one?.pluginId).toBe('org.example.sample')
    expect(one?.arguments).toStrictEqual([
      { name: 'title', title: 'Title', isRequired: true },
      { name: 'folder', title: 'Folder', isRequired: false },
    ])
  })

  it('are read with their condition, destination flag and the parts of their plugin', async () => {
    request.listCommands.mockResolvedValue({
      commands: [
        {
          pluginId: 'md.numen.tree',
          id: 'tree.revealNote',
          title: 'Show note in file tree',
          icon: 'folder-open',
          when: 'target.isNote',
          arguments: [],
        },
      ],
      slots: [],
    })

    const {
      commands: [one],
    } = await listDeclaredCommands()

    expect(one).toStrictEqual({
      pluginId: 'md.numen.tree',
      id: 'tree.revealNote',
      title: 'Show note in file tree',
      icon: 'folder-open',
      when: 'target.isNote',
      arguments: [],
    })
  })

  it('are read as offering items where the core marks them', async () => {
    request.listCommands.mockResolvedValue({
      commands: [{ ...proofread, hasItems: true }],
      slots: [],
    })

    const {
      commands: [one],
    } = await listDeclaredCommands()

    expect(one).toStrictEqual(proofread)
  })
})

describe('a run', () => {
  it('sends the file and values and returns the plugin message', async () => {
    request.runPluginCommand.mockResolvedValue({
      error: COMMAND_ERROR_CODE.UNSPECIFIED,
      message: 'Converted 3 pages',
      openedPath: 'a.md',
    })

    const outcome = await runDeclaredCommand(convert, { path: 'a.pdf', values: { pages: '3' } })

    expect(outcome).toStrictEqual({
      ok: true,
      value: { message: 'Converted 3 pages', openedPath: 'a.md' },
    })
    expect(request.runPluginCommand.mock.lastCall?.[0]).toMatchObject({
      pluginId: 'md.numen.pages',
      command: 'convert',
      path: 'a.pdf',
      arguments: { pages: '3' },
    })
  })

  it('of a command over no file is handed none', async () => {
    request.runPluginCommand.mockResolvedValue({
      error: COMMAND_ERROR_CODE.UNSPECIFIED,
      message: '',
      openedPath: '',
    })

    await runDeclaredCommand(
      { ...convert, when: 'window.isVaultReady' },
      { path: 'a.md', values: {} },
    )

    expect(request.runPluginCommand.mock.lastCall?.[0]).toMatchObject({ path: '' })
  })

  it('of a command over no file is handed the file a place named', async () => {
    request.runPluginCommand.mockResolvedValue({
      error: COMMAND_ERROR_CODE.UNSPECIFIED,
      message: '',
      openedPath: '',
    })

    await runDeclaredCommand(
      { ...convert, when: 'window.isVaultReady' },
      { path: 'Glaciers', values: {}, isTargetNamed: true },
    )

    expect(request.runPluginCommand.mock.lastCall?.[0]).toMatchObject({ path: 'Glaciers' })
  })

  it('that did not succeed answers why', async () => {
    request.runPluginCommand.mockResolvedValue({
      error: COMMAND_ERROR_CODE.FAILED,
      message: 'the disk is full',
    })

    const outcome = await runDeclaredCommand(convert, { path: 'a.pdf', values: {} })

    expect(outcome).toStrictEqual({ ok: false, error: 'the disk is full' })
  })
})

describe('the items of a command', () => {
  it('are the items the core lists over the file for each command, in one request', async () => {
    request.listCommandItems.mockResolvedValue({
      lists: [
        {
          pluginId: 'org.example.ink',
          command: 'ink.proofread',
          items: [{ id: 'org.example.ears/ears', title: 'transcript (live)' }],
          error: COMMAND_ERROR_CODE.UNSPECIFIED,
          message: '',
        },
        {
          pluginId: 'md.numen.pages',
          command: 'convert',
          items: [],
          error: COMMAND_ERROR_CODE.FAILED,
          message: 'talks/a.mp3 is not in the vault',
        },
      ],
    })

    expect(
      await listDeclaredItemLists('talks/a.mp3', [
        { pluginId: 'org.example.ink', command: 'ink.proofread' },
        { pluginId: 'md.numen.pages', command: 'convert' },
      ]),
    ).toStrictEqual([
      {
        pluginId: 'org.example.ink',
        command: 'ink.proofread',
        result: { ok: true, value: [{ id: 'org.example.ears/ears', title: 'transcript (live)' }] },
      },
      {
        pluginId: 'md.numen.pages',
        command: 'convert',
        result: { ok: false, error: 'talks/a.mp3 is not in the vault' },
      },
    ])
    expect(request.listCommandItems.mock.lastCall?.[0]).toMatchObject({
      path: 'talks/a.mp3',
      commands: [
        { pluginId: 'org.example.ink', command: 'ink.proofread' },
        { pluginId: 'md.numen.pages', command: 'convert' },
      ],
    })
  })

  it('run the command over the file with the item picked as its itemId', async () => {
    request.runPluginCommand.mockResolvedValue({
      error: COMMAND_ERROR_CODE.UNSPECIFIED,
      message: '',
      openedPath: '',
    })

    await runDeclaredCommand(proofread, {
      path: 'talks/a.mp3',
      values: {},
      itemId: 'org.example.ears/ears',
    })

    expect(request.runPluginCommand.mock.lastCall?.[0]).toMatchObject({
      pluginId: 'org.example.ink',
      command: 'ink.proofread',
      path: 'talks/a.mp3',
      arguments: {},
      itemId: 'org.example.ears/ears',
    })
  })
})

describe('the handlers of shared commands', () => {
  it('are read with their plugin, command, title, condition and whether the core lists their items', async () => {
    request.listCommands.mockResolvedValue({
      commands: [],
      slots: [],
      handlers: [
        {
          pluginId: 'org.example.ink',
          handler: {
            command: 'app:version.delete',
            title: 'Delete proofread',
            when: "version.kind == 'ink'",
          },
          hasItems: true,
        },
        {
          pluginId: 'org.example.pages',
          handler: { command: 'app:file.createText', title: 'Extract text layer', when: '' },
          hasItems: false,
        },
      ],
    })

    const { handlers } = await listDeclaredCommands()

    expect(handlers).toStrictEqual([
      {
        pluginId: 'org.example.ink',
        command: 'app:version.delete',
        title: 'Delete proofread',
        when: "version.kind == 'ink'",
        hasItems: true,
      },
      {
        pluginId: 'org.example.pages',
        command: 'app:file.createText',
        title: 'Extract text layer',
        hasItems: false,
      },
    ])
  })

  it('run in the backend under the full id of the command, with the item picked and the values', async () => {
    request.runPluginCommand.mockResolvedValue({
      error: COMMAND_ERROR_CODE.UNSPECIFIED,
      message: 'Deleted',
      openedPath: '',
    })

    const outcome = await runDeclaredHandler(
      { pluginId: 'org.example.ink', command: 'app:version.delete' },
      { path: 'talks/a.mp3', values: { reason: 'stale' }, itemId: 'v1' },
    )

    expect(outcome).toStrictEqual({ ok: true, value: { message: 'Deleted', openedPath: '' } })
    expect(request.runPluginCommand.mock.lastCall?.[0]).toMatchObject({
      pluginId: 'org.example.ink',
      command: 'app:version.delete',
      path: 'talks/a.mp3',
      arguments: { reason: 'stale' },
      itemId: 'v1',
    })
  })

  it('that did not succeed answer why', async () => {
    request.runPluginCommand.mockResolvedValue({
      error: COMMAND_ERROR_CODE.FAILED,
      message: 'no such version',
    })

    expect(
      await runDeclaredHandler(
        { pluginId: 'org.example.ink', command: 'app:version.delete' },
        { path: 'a.mp3', values: {} },
      ),
    ).toStrictEqual({ ok: false, error: 'no such version' })
  })
})

describe('the keybindings of the plugins', () => {
  it('are read with their plugin, key, command, values and condition, and none where the core lists none', async () => {
    request.listCommands.mockResolvedValue({
      commands: [],
      slots: [],
      keybindings: [
        {
          pluginId: 'org.example.ink',
          keybinding: { key: 'mod+shift+i', command: 'ink.open', values: {}, when: '' },
        },
        {
          pluginId: 'org.example.ink',
          keybinding: {
            key: 'g',
            command: 'app:file.open',
            values: { placement: 'split' },
            when: '!window.isFieldFocused',
          },
        },
      ],
    })

    const { keybindings } = await listDeclaredCommands()

    expect(keybindings).toStrictEqual([
      { pluginId: 'org.example.ink', key: 'mod+shift+i', command: 'ink.open' },
      {
        pluginId: 'org.example.ink',
        key: 'g',
        command: 'app:file.open',
        values: { placement: 'split' },
        when: '!window.isFieldFocused',
      },
    ])

    request.listCommands.mockResolvedValue({ commands: [], slots: [] })
    expect((await listDeclaredCommands()).keybindings).toStrictEqual([])
  })
})
