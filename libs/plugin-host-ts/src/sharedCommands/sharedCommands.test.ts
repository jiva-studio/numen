/** The handlers of the shared commands over a target, and the run of one. Every plugin here is invented. */
import { describe, expect, it, vi } from 'vitest'
import type { CommandTarget } from '@numen/plugin-sdk'
import { getTargetValues } from '../when/targetValues'
import { createSharedCommands, type DeclaredHandler } from './sharedCommands'

const LEDGER = 'org.example.ledger'
const HARBOR = 'org.example.harbor'
const TYPES = { pdf: 'application/pdf', png: 'image/png' }

const HANDLERS: DeclaredHandler[] = [
  {
    pluginId: LEDGER,
    command: 'app:file.createText',
    title: 'Read the ledger',
    when: "target.contentType == 'application/pdf'",
  },
  {
    pluginId: HARBOR,
    command: 'app:file.createText',
    title: 'Recognize text',
    when: "target.contentType in ['application/pdf', 'image/png']",
  },
  {
    pluginId: HARBOR,
    command: 'app:link.getMedia',
    title: 'Play',
    when: "target.noteType == 'link'",
  },
]

const file = (path: string): CommandTarget => ({ path, title: path, isFolder: false, others: [] })

function setup(handlers: readonly DeclaredHandler[] = HANDLERS) {
  const runHandler = vi.fn(async () => 'http://127.0.0.1:4010/copies/talk.mp4' as unknown)
  const shared = createSharedCommands({
    getHandlers: () => handlers,
    getWhenValues: (target) =>
      getTargetValues(target, TYPES, { type: target?.path.endsWith('.md') ? 'link' : '' }),
    runHandler,
  })
  return { shared, runHandler }
}

describe('a shared command', () => {
  it('is one of commands.json, and no command of a plugin is', () => {
    const { shared } = setup()

    expect(shared.isShared('app:file.createText')).toBe(true)
    expect(shared.isShared(`${LEDGER}:ledger.open`)).toBe(false)
  })

  it('has the handlers whose condition holds over the target, in the order of the plugins, as items by plugin id', () => {
    const { shared } = setup()

    expect(shared.listHandlers('app:file.createText', file('papers/Hive.pdf'))).toStrictEqual([
      { id: LEDGER, title: 'Read the ledger' },
      { id: HARBOR, title: 'Recognize text' },
    ])
    expect(shared.listHandlers('app:file.createText', file('papers/Hive.png'))).toStrictEqual([
      { id: HARBOR, title: 'Recognize text' },
    ])
    expect(shared.listHandlers('app:file.createText', file('notes/Bees.md'))).toStrictEqual([])
  })

  it('has no handlers until the first list has loaded', () => {
    const shared = createSharedCommands({
      getHandlers: () => undefined,
      getWhenValues: (target) => getTargetValues(target, TYPES),
      runHandler: vi.fn(),
    })

    expect(shared.listHandlers('app:file.createText', file('a.pdf'))).toStrictEqual([])
  })

  it('runs its one handler directly, and returns what the handler returns', async () => {
    const { shared, runHandler } = setup()

    await expect(shared.run('app:link.getMedia', file('links/talk.md'))).resolves.toBe(
      'http://127.0.0.1:4010/copies/talk.mp4',
    )
    expect(runHandler).toHaveBeenCalledWith({
      pluginId: HARBOR,
      command: 'app:link.getMedia',
      target: file('links/talk.md'),
      argument: undefined,
    })
  })

  it('runs the handler its argument names, of several', async () => {
    const { shared, runHandler } = setup()

    await shared.run('app:file.createText', file('papers/Hive.pdf'), { pluginId: HARBOR })

    expect(runHandler).toHaveBeenCalledWith({
      pluginId: HARBOR,
      command: 'app:file.createText',
      target: file('papers/Hive.pdf'),
      argument: undefined,
    })
  })

  it('refuses a run that names no handler of several, or a handler that does not hold over the target', async () => {
    const { shared, runHandler } = setup()

    await expect(shared.run('app:file.createText', file('papers/Hive.pdf'))).rejects.toThrow(
      'app:file.createText has 2 handlers over papers/Hive.pdf; the run names none of them',
    )
    await expect(
      shared.run('app:file.createText', file('papers/Hive.png'), { pluginId: LEDGER }),
    ).rejects.toThrow(`app:file.createText has no handler of ${LEDGER} over papers/Hive.png`)
    expect(runHandler).not.toHaveBeenCalled()
  })
})
