/**
 * The handlers of shared commands that run in their plugins' backends, as the plugin host lists
 * their items and runs them.
 */
import { describe, expect, it, vi } from 'vitest'
import { createBackendHandlers } from './backendHandlers'

const TARGET = { path: 'talks/a.mp3', title: 'a', isFolder: false, others: [], tab: '' }

const createWith = (
  outcome: Awaited<ReturnType<Parameters<typeof createBackendHandlers>[0]['run']>>,
) => {
  const deps = {
    getHandlers: () => [],
    list: vi.fn(async () => [{ id: 'v1', title: 'Transcript' }]),
    run: vi.fn(async () => outcome),
    writeMessage: vi.fn(),
    openFile: vi.fn(async () => undefined),
  }
  return { deps, handlers: createBackendHandlers(deps) }
}

describe('createBackendHandlers', () => {
  it('lists the items of a handler over the path of the target', async () => {
    const { deps, handlers } = createWith({ ok: true, value: { message: '', openedPath: '' } })

    expect(
      await handlers.listBackendHandlerItems('md.ears', 'app:link.downloadMedia', TARGET),
    ).toStrictEqual([{ id: 'v1', title: 'Transcript' }])
    expect(deps.list).toHaveBeenCalledWith(
      { pluginId: 'md.ears', id: 'app:link.downloadMedia' },
      'talks/a.mp3',
    )
  })

  it('runs a handler over the path with the item picked, writes its message and opens its file', async () => {
    const { deps, handlers } = createWith({
      ok: true,
      value: { message: 'Transcribed', openedPath: 'talks/a.md' },
    })

    await handlers.runBackendHandler({
      pluginId: 'md.ears',
      command: 'app:file.createText',
      target: TARGET,
      itemId: 'fast',
      values: { language: 'en' },
    })

    expect(deps.run).toHaveBeenCalledWith(
      { pluginId: 'md.ears', command: 'app:file.createText' },
      { path: 'talks/a.mp3', values: { language: 'en' }, itemId: 'fast' },
    )
    expect(deps.writeMessage).toHaveBeenCalledWith('Transcribed', 'info')
    expect(deps.openFile).toHaveBeenCalledWith('talks/a.md')
  })

  it('rejects a run that did not succeed with the error as the plugin words it, and writes nothing', async () => {
    const { deps, handlers } = createWith({ ok: false, error: 'no audio' })

    await expect(
      handlers.runBackendHandler({
        pluginId: 'md.ears',
        command: 'app:file.createText',
        target: TARGET,
      }),
    ).rejects.toThrow('no audio')

    expect(deps.writeMessage).not.toHaveBeenCalled()
    expect(deps.openFile).not.toHaveBeenCalled()
  })

  it('gives the handlers the core lists', () => {
    const handler = {
      pluginId: 'md.ears',
      command: 'app:file.createText',
      title: 'Transcribe',
      hasItems: false,
    }
    const handlers = createBackendHandlers({
      ...createWith({ ok: false, error: '' }).deps,
      getHandlers: () => [handler],
    })

    expect(handlers.getHandlerDeclarations()).toStrictEqual([handler])
  })
})
