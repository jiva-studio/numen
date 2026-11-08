import { describe, expect, it, vi } from 'vitest'
import { watch } from 'vue'
import { runInBackground } from '@numen/ui'
import { createSettingsStore } from './store'

const sleep = (ms: number) => new Promise((resolve) => setTimeout(resolve, ms))
const AT = ['p', 'enabled'] as const

describe('settings store', () => {
  it('shows a written value without reverting while the vault applies it', async () => {
    const file: Record<string, unknown> = { p: { enabled: true } }
    const core = {
      getSettings: async () => ({
        written: JSON.stringify(file),
        path: '/f',
        models: [],
        plugins: [],
      }),
      updateSettings: async (edits: readonly { at: readonly string[]; value: string }[]) => {
        for (const e of edits) (file as { p: { enabled: boolean } }).p.enabled = JSON.parse(e.value)
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

const createHeldVault = (isEnabled: boolean) => {
  const file = { p: { enabled: isEnabled } }
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
        const previous = file.p.enabled
        for (const e of edits) file.p.enabled = JSON.parse(e.value) as boolean
        writes.push({
          resolve: () => {
            isActive = file.p.enabled
            resolve()
          },
          reject: (error) => {
            file.p.enabled = previous
            reject(error)
          },
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

describe('the enabled switch waits for the answer of the vault', () => {
  const words = { unturned: 'u', unreadSettings: 'r' }

  it('keeps its position while the write is in flight, and moves once when the answer arrives', async () => {
    const vault = createHeldVault(true)
    const store = createSettingsStore(vault.core, words, vi.fn())
    const first = store.start()
    vault.reads[0]?.resolve()
    await first
    const history: unknown[] = [store.getCommittedSetting(AT)]
    watch(
      () => store.getCommittedSetting(AT),
      (v) => history.push(v),
      { flush: 'sync' },
    )

    // A read begins before the click and one while the write is in flight; both answer last.
    const early = store.start()
    const earlyRead = vault.reads.at(-1)
    const write = store.writeSetting(AT, false)
    await vi.waitFor(() => expect(vault.writes).toHaveLength(1))
    const during = store.start()
    const duringRead = vault.reads.at(-1)
    expect(store.isPending(AT)).toBe(true)
    expect(store.getCommittedSetting(AT)).toBe(true)

    vault.writes[0]?.resolve()
    await vi.waitFor(() => expect(store.isPending(AT)).toBe(false))
    expect(store.getCommittedSetting(AT)).toBe(false)
    await vi.waitFor(() => expect(vault.reads.length).toBeGreaterThan(3))
    vault.reads.at(-1)?.resolve()
    duringRead?.resolve()
    earlyRead?.resolve()
    await Promise.all([write, during, early])
    expect(history).toEqual([true, false])
  })

  it('keeps the old value, stops waiting and says why where the write fails', async () => {
    const vault = createHeldVault(true)
    const say = vi.fn()
    const store = createSettingsStore(vault.core, words, say)
    const first = store.start()
    vault.reads[0]?.resolve()
    await first
    const history: unknown[] = [store.getCommittedSetting(AT)]
    watch(
      () => store.getCommittedSetting(AT),
      (v) => history.push(v),
      { flush: 'sync' },
    )

    const write = store.writeSetting(AT, false)
    await vi.waitFor(() => expect(vault.writes).toHaveLength(1))
    vault.writes[0]?.reject(new Error('refused'))
    await vi.waitFor(() => expect(vault.reads.length).toBeGreaterThan(1))
    // The file holds the old value again, as the vault failed before it wrote.
    vault.reads.at(-1)?.resolve()
    await write
    expect(store.isPending(AT)).toBe(false)
    expect(say).toHaveBeenCalledWith(expect.stringMatching(/^u /), 'error')
    expect(history).toEqual([true])
  })
})
