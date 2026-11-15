/**
 * The activation of the plugin in a window: the saved search tab kind, over the service of the
 * backend and the files of the workspace, in a window that has a workspace and in no other.
 */
import { afterEach, describe, expect, it, vi } from 'vitest'
import type { TabKindRegistration, WorkspaceHost } from '@numen/plugin-sdk'
import { createMockHostServices, createMockPluginContext } from '@numen/plugin-sdk/testing'
import { HARBOR, NOTE_PATH } from './results/fixtures/responses'
import type { SavedSearchTabState } from './results'
import { activate } from './index'

const settle = (): Promise<void> => new Promise((resolve) => setTimeout(resolve, 0))

const encoder = new TextEncoder()

const serveBackend = (url: string, init?: RequestInit): Response => {
  if (url.endsWith('/md.numen.tools.saved.search/GetResults')) return Response.json(HARBOR)
  if (url.endsWith('/md.numen.tools.saved.search/WatchResults')) {
    return new Response(
      new ReadableStream({
        start: (controller) => {
          const change = { computedAt: HARBOR.computedAt, isRunning: false }
          controller.enqueue(encoder.encode(`data: ${JSON.stringify(change)}\n\n`))
          init?.signal?.addEventListener('abort', () =>
            controller.error(new DOMException('The call was aborted.', 'AbortError')),
          )
        },
      }),
      { headers: { 'Content-Type': 'text/event-stream' } },
    )
  }
  return Response.json({ code: 'not_found', message: url }, { status: 404 })
}

const createWorkspace = () => {
  const openAt = vi.fn()
  return { openAt, workspace: { files: { openAt } } as unknown as WorkspaceHost }
}

afterEach(() => {
  vi.unstubAllGlobals()
})

describe('the saved search frontend', () => {
  it('registers the saved-search kind', () => {
    const { workspace } = createWorkspace()
    const context = createMockPluginContext({ pluginId: 'md.numen.tools.saved.search' })

    activate(context, createMockHostServices({ workspace }))

    expect(context.getRegisteredTabKinds().map((kind) => kind.kind)).toStrictEqual(['saved-search'])
  })

  it('registers nothing in a window without a workspace', () => {
    const context = createMockPluginContext({ pluginId: 'md.numen.tools.saved.search' })

    activate(context, createMockHostServices())

    expect(context.getRegisteredTabKinds()).toStrictEqual([])
  })

  it('opens a result through the files of the workspace', async () => {
    vi.stubGlobal('fetch', async (url: string, init?: RequestInit) =>
      serveBackend(String(url), init),
    )
    const { workspace, openAt } = createWorkspace()
    const context = createMockPluginContext({ pluginId: 'md.numen.tools.saved.search' })
    activate(context, createMockHostServices({ workspace }))
    const [kind] = context.getRegisteredTabKinds() as TabKindRegistration<SavedSearchTabState>[]
    if (!kind?.open) throw new Error('the saved search kind opens no tab')

    const state = kind.open(NOTE_PATH, 'tab-1')
    await vi.waitFor(() => expect(state.view.value?.items).toHaveLength(3))
    state.actions.openResult(1)
    kind.onDestroy?.(state, 'tab-1')
    await settle()

    expect(openAt).toHaveBeenCalledWith('Harbor/Night of the gale.md', [{ start: 0, end: 712 }])
  })
})
