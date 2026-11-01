import { describe, expect, it, vi } from 'vitest'
import { watch } from 'vue'
import { runInBackground } from '@numen/ui'
import { createSettingsStore } from './store'

const sleep = (ms: number) => new Promise((resolve) => setTimeout(resolve, ms))
const AT = ['plugins', 'p', 'enabled'] as const

describe('settings store', () => {
  it('shows a written value without reverting while the vault applies it', async () => {
    const file: Record<string, unknown> = { plugins: { p: { enabled: true } } }
    const core = {
      getSettings: async () => ({
        written: JSON.stringify(file),
        path: '/f',
        models: [],
        plugins: [],
      }),
      updateSettings: async (edits: readonly { at: readonly string[]; value: string }[]) => {
        for (const e of edits)
          (file.plugins as { p: { enabled: boolean } }).p.enabled = JSON.parse(e.value)
        await sleep(100) // the plugin host applies the change
      },
    }
    const store = createSettingsStore(core, { unturned: 'u', unreadSettings: 'r' }, vi.fn())
    await store.start()
    const values: unknown[] = []
    watch(
      () => store.getSetting(AT),
      (v) => values.push(v),
      { flush: 'sync' },
    )
    const failures: unknown[] = []
    const poll = setInterval(
      () => runInBackground(store.start(), (error) => failures.push(error)),
      20,
    )
    await store.writeSetting(AT, false)
    await sleep(300)
    clearInterval(poll)
    expect(values).toEqual([false])
    expect(failures).toEqual([])
  })
})

/** An answer the test lets through, out of order where it chooses. */
interface HeldCall {
  readonly resolve: () => void
  readonly reject: (error: unknown) => void
}

/**
 * A vault holding one plugin. A read takes what the file holds when the vault receives it, and
 * every read and write waits until the test answers it.
 */
const createHeldVault = (isEnabled: boolean) => {
  const file = { plugins: { p: { enabled: isEnabled } } }
  let isActive = isEnabled
  const reads: HeldCall[] = []
  const writes: HeldCall[] = []
  const core = {
    getSettings: () =>
      new Promise<{ written: string; path: string; models: []; plugins: [] }>((resolve, reject) => {
        const settingsText = JSON.stringify(file)
        reads.push({
          resolve: () => resolve({ written: settingsText, path: '/f', models: [], plugins: [] }),
          reject,
        })
      }),
    updateSettings: (edits: readonly { at: readonly string[]; value: string }[]) =>
      new Promise<void>((resolve, reject) => {
        for (const e of edits) file.plugins.p.enabled = JSON.parse(e.value) as boolean
        writes.push({
          resolve: () => {
            isActive = file.plugins.p.enabled
            resolve()
          },
          reject,
        })
      }),
  }
  return { core, reads, writes, isActive: () => isActive }
}

describe('the enabled switch of a plugin', () => {
  it('shows each requested value from the click on, with reads out of order and a failed read-back', async () => {
    const vault = createHeldVault(false)
    const store = createSettingsStore(vault.core, { unturned: 'u', unreadSettings: 'r' }, vi.fn())
    const firstRead = store.start()
    vault.reads[0]?.resolve()
    await firstRead
    const history: unknown[] = [store.getSetting(AT)]
    watch(
      () => store.getSetting(AT),
      (v) => history.push(v),
      { flush: 'sync' },
    )

    for (const [index, isEnabled] of [true, false].entries()) {
      // A refresh that began before the click answers after the write's own read-back.
      const refresh = store.start()
      const before = vault.reads.at(-1)
      const write = store.writeSetting(AT, isEnabled)
      await vi.waitFor(() => expect(vault.writes).toHaveLength(index + 1))
      // The vault answers the write once the plugin has started or stopped.
      vault.writes.at(-1)?.resolve()
      await vi.waitFor(() => expect(vault.reads.at(-1)).not.toBe(before))
      // The read-back fails while the host still lists the plugin as starting or stopping.
      vault.reads.at(-1)?.reject(new Error('busy'))
      await write
      before?.resolve()
      await refresh
      const later = store.start()
      vault.reads.at(-1)?.resolve()
      await later
      expect(vault.isActive()).toBe(isEnabled)
    }

    expect(history).toEqual([false, true, false])
  })
})
