import { nextTick, ref } from 'vue'
import { afterEach, expect, it, vi } from 'vitest'
import { createTestTabs } from '@/testing/tabs'
import { parseSavedTabs, useOpenTabs } from './useOpenTabs'
import type { OpenTabs } from '../lib/tab'

afterEach(() => vi.useRealTimers())

function createDraftWindow() {
  const windowTabs = createTestTabs()
  windowTabs.kinds.registerKind({
    kind: 'chat',
    pane: {},
    open: (_path, _id, saved) => ({
      draft: ref(typeof saved === 'string' ? saved : ''),
    }),
    address: { getState: (state: { draft: { value: string } }) => state.draft.value },
  })
  return windowTabs
}

async function openDraft() {
  const windowTabs = createDraftWindow()
  const id = await windowTabs.tabs.openTabOfKind('chat')
  const draft = windowTabs.handle.getTabState<{ draft: { value: string } }>('chat', id)!.draft
  const write = vi.fn<(_open: OpenTabs) => Promise<void>>().mockResolvedValue()
  const tabs = useOpenTabs({ windowTabs, writeOpenTabs: write, writeMessage: vi.fn() })
  await nextTick()
  write.mockClear()
  return { windowTabs, draft, tabs, write, id }
}

it('coalesces typing into one saved draft when leaving the vault', async () => {
  vi.useFakeTimers()
  const { draft, tabs, write, id } = await openDraft()
  for (const text of ['h', 'he', 'hel', 'hell', 'hello']) {
    draft.value = text
    await nextTick()
    expect(tabs.open.value.tabs[0]?.state).toBe(JSON.stringify(text))
  }
  expect(write).not.toHaveBeenCalled()

  await tabs.pause()
  await vi.runAllTimersAsync()
  expect(write).toHaveBeenCalledTimes(1)
  const saved = write.mock.calls[0]![0]
  const restarted = createDraftWindow()
  restarted.restored.restoreTabs(parseSavedTabs(saved), saved.activeId, saved.layout)
  expect(restarted.handle.getTabState<{ draft: { value: string } }>('chat', id)?.draft.value).toBe(
    'hello',
  )
})

it('saves the latest draft after typing stops and updates the live report immediately', async () => {
  vi.useFakeTimers()
  const { draft, tabs, write } = await openDraft()
  draft.value = 'first'
  await nextTick()
  await vi.advanceTimersByTimeAsync(400)
  draft.value = 'second'
  await nextTick()
  await vi.advanceTimersByTimeAsync(400)
  expect(write).not.toHaveBeenCalled()

  await vi.runAllTimersAsync()
  expect(write).toHaveBeenCalledExactlyOnceWith(tabs.open.value)
  draft.value = 'third'
  await nextTick()
  expect(write).toHaveBeenCalledTimes(1)
  await tabs.flush()
  expect(write.mock.lastCall?.[0].tabs[0]?.state).toBe('"third"')

  draft.value = 'fourth'
  await nextTick()
  await vi.runAllTimersAsync()
  expect(write.mock.lastCall?.[0].tabs[0]?.state).toBe('"fourth"')
})

it('reports a tab switch with its pending draft immediately', async () => {
  vi.useFakeTimers()
  const { windowTabs, draft, id, tabs, write } = await openDraft()
  draft.value = 'unsent'
  await nextTick()
  windowTabs.kinds.registerKind({ kind: 'other', open: () => ({}), pane: {} })
  const other = await windowTabs.tabs.openTabOfKind('other')
  await nextTick()

  expect(write).toHaveBeenCalledExactlyOnceWith(tabs.open.value)
  expect(write.mock.lastCall?.[0].activeId).toBe(other)
  expect(write.mock.lastCall?.[0].tabs.find((tab) => tab.id === id)?.state).toBe('"unsent"')
  await vi.runAllTimersAsync()
  expect(write).toHaveBeenCalledTimes(1)
})

it('flushes draft edits made while the previous write is in flight before pausing', async () => {
  vi.useFakeTimers()
  const { draft, tabs, write } = await openDraft()
  let finish!: () => void
  write.mockImplementationOnce(
    () =>
      new Promise<void>((resolve) => {
        finish = resolve
      }),
  )
  draft.value = 'before'
  await nextTick()
  const paused = tabs.pause()
  draft.value = 'after'
  await nextTick()
  finish()
  await paused
  expect(write.mock.calls.map(([saved]) => saved.tabs[0]?.state)).toEqual(['"before"', '"after"'])
  await vi.runAllTimersAsync()
  expect(write).toHaveBeenCalledTimes(2)
})

it('rejects a failed flush and retries the same pending draft', async () => {
  vi.useFakeTimers()
  const { draft, tabs, write } = await openDraft()
  draft.value = 'retry'
  await nextTick()
  write.mockRejectedValueOnce(new Error('offline'))
  await expect(tabs.pause()).rejects.toThrow('offline')
  await tabs.flush()
  await vi.runAllTimersAsync()
  expect(write).toHaveBeenCalledTimes(2)
  expect(write.mock.lastCall?.[0].tabs[0]?.state).toBe('"retry"')
})
