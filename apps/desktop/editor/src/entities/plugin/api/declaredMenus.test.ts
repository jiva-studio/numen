/** The menu entries of the plugins as the window reads them from the core. */
import { describe, expect, it, vi } from 'vitest'

vi.mock('@/shared/clients', () => ({ pluginCommands: request }))

const request = {
  listCommands: vi.fn(),
  runPluginCommand: vi.fn(),
}

const { listDeclaredCommands } = await import('./declaredCommands')

describe('the menu entries the plugins declare', () => {
  it('are read field by field in the order the core lists them', async () => {
    request.listCommands.mockResolvedValue({
      commands: [],
      menus: [
        {
          pluginId: 'org.example.atlas',
          location: 'searchResult',
          command: 'atlas.open',
          group: 'navigate',
          order: 20,
        },
        { pluginId: 'org.example.books', location: 'titleBar', command: 'app:note.create' },
      ],
    })

    const { menus } = await listDeclaredCommands()

    expect(menus).toHaveLength(2)
    expect(menus[0]).toMatchObject({
      pluginId: 'org.example.atlas',
      location: 'searchResult',
      command: 'atlas.open',
      group: 'navigate',
      order: 20,
    })
    expect(menus[1]).toMatchObject({
      pluginId: 'org.example.books',
      location: 'titleBar',
      command: 'app:note.create',
    })
  })

  it('keep an order of zero apart from no order', async () => {
    request.listCommands.mockResolvedValue({
      commands: [],
      menus: [
        { pluginId: 'org.example.atlas', location: 'titleBar', command: 'atlas.open', order: 0 },
        { pluginId: 'org.example.atlas', location: 'titleBar', command: 'atlas.close' },
      ],
    })

    const { menus } = await listDeclaredCommands()

    expect(menus[0]?.order).toBe(0)
    expect(menus[1]?.order).toBeUndefined()
  })

  it('read an empty group as none', async () => {
    request.listCommands.mockResolvedValue({
      commands: [],
      menus: [
        { pluginId: 'org.example.atlas', location: 'titleBar', command: 'atlas.open', group: '' },
      ],
    })

    const { menus } = await listDeclaredCommands()

    expect(menus[0]?.group ?? '').toBe('')
  })

  it('stay empty where the core lists commands and no menu entries', async () => {
    request.listCommands.mockResolvedValue({
      commands: [
        {
          pluginId: 'org.example.atlas',
          id: 'atlas.open',
          title: 'Open atlas',
          contentTypes: [],
          extensions: [],
          arguments: [],
        },
      ],
      menus: [],
    })

    const list = await listDeclaredCommands()

    expect(list.menus).toEqual([])
    expect(list.commands.map((one) => one.id)).toEqual(['atlas.open'])
  })
})
