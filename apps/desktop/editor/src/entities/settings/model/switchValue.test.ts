import { afterEach, describe, expect, it, vi } from 'vitest'
import { watch } from 'vue'
import { runInBackground } from '@numen/ui'
import { installCoreAnswers } from '@numen/core-client/testing'
import { createSettingsStore } from './store'

const sleep = (ms: number) => new Promise((resolve) => setTimeout(resolve, ms))
const AT = ['p', 'enabled'] as const

const answerWith = (written: string) => ({
  written,
  path: '/f',
  models: [],
  plugins: [],
  app: [],
  secretSections: [],
  secretMenus: [],
  credentialsPath: '',
})

let restore: (() => void) | undefined

afterEach(() => {
  restore?.()
  restore = undefined
})

describe('settings store', () => {
  it('shows a written value without reverting while the vault applies it', async () => {
    const file: Record<string, unknown> = { p: { enabled: true } }
    restore = installCoreAnswers({
      settingsService: {
        getSettings: () => answerWith(JSON.stringify(file)),
        writeSettings: async (request) => {
          for (const e of request.settings) {
            ;(file as { p: { enabled: boolean } }).p.enabled = JSON.parse(e.value)
          }
          await sleep(100) // the plugin host applies the change
          return {}
        },
      },
    })
    const store = createSettingsStore({ unturned: 'u', unreadSettings: 'r' }, vi.fn())
    await store.start()
    const values: unknown[] = []
    watch(
      () => store.read.getSetting(AT),
      (v) => values.push(v),
      { flush: 'sync' },
    )
    const failures: unknown[] = []
    const poll = setInterval(
      () => runInBackground(store.start(), (error) => failures.push(error)),
      20,
    )
    await store.write.writeSetting(AT, false)
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
  restore = installCoreAnswers({
    settingsService: {
      getSettings: () =>
        new Promise<ReturnType<typeof answerWith>>((resolve, reject) => {
          const settingsText = JSON.stringify(file)
          reads.push({ resolve: () => resolve(answerWith(settingsText)), reject })
        }),
      writeSettings: (request) =>
        new Promise<Record<string, never>>((resolve, reject) => {
          const previous = file.p.enabled
          for (const e of request.settings) file.p.enabled = JSON.parse(e.value) as boolean
          writes.push({
            resolve: () => {
              isActive = file.p.enabled
              resolve({})
            },
            reject: (error) => {
              file.p.enabled = previous
              reject(error)
            },
          })
        }),
    },
  })
  return { reads, writes, isActive: () => isActive }
}

const startRead = async (
  store: ReturnType<typeof createSettingsStore>,
  vault: ReturnType<typeof createHeldVault>,
) => {
  const count = vault.reads.length
  const read = store.start()
  await vi.waitFor(() => expect(vault.reads).toHaveLength(count + 1))
  return { read, call: vault.reads.at(-1) }
}

describe('the enabled switch of a plugin', () => {
  it('shows each requested value from the click on, with reads out of order and a failed read-back', async () => {
    const vault = createHeldVault(false)
    const store = createSettingsStore({ unturned: 'u', unreadSettings: 'r' }, vi.fn())
    const first = await startRead(store, vault)
    first.call?.resolve()
    await first.read
    const history: unknown[] = [store.read.getSetting(AT)]
    watch(
      () => store.read.getSetting(AT),
      (v) => history.push(v),
      { flush: 'sync' },
    )

    for (const [index, isEnabled] of [true, false].entries()) {
      // A refresh that began before the click answers after the write's own read-back.
      const refresh = await startRead(store, vault)
      const before = refresh.call
      const write = store.write.writeSetting(AT, isEnabled)
      await vi.waitFor(() => expect(vault.writes).toHaveLength(index + 1))
      // The vault answers the write once the plugin has started or stopped.
      vault.writes.at(-1)?.resolve()
      await vi.waitFor(() => expect(vault.reads.at(-1)).not.toBe(before))
      // The read-back fails while the host still lists the plugin as starting or stopping.
      vault.reads.at(-1)?.reject(new Error('busy'))
      await write
      before?.resolve()
      await refresh.read
      const later = await startRead(store, vault)
      later.call?.resolve()
      await later.read
      expect(vault.isActive()).toBe(isEnabled)
    }

    expect(history).toEqual([false, true, false])
  })
})

describe('the enabled switch waits for the answer of the vault', () => {
  const words = { unturned: 'u', unreadSettings: 'r' }

  it('keeps its position while the write is in flight, and moves once when the answer arrives', async () => {
    const vault = createHeldVault(true)
    const store = createSettingsStore(words, vi.fn())
    const first = await startRead(store, vault)
    first.call?.resolve()
    await first.read
    const history: unknown[] = [store.read.getCommittedSetting(AT)]
    watch(
      () => store.read.getCommittedSetting(AT),
      (v) => history.push(v),
      { flush: 'sync' },
    )

    // A read begins before the click and one while the write is in flight; both answer last.
    const early = await startRead(store, vault)
    const write = store.write.writeSetting(AT, false)
    await vi.waitFor(() => expect(vault.writes).toHaveLength(1))
    const during = await startRead(store, vault)
    expect(store.read.isPending(AT)).toBe(true)
    expect(store.read.getCommittedSetting(AT)).toBe(true)

    vault.writes[0]?.resolve()
    await vi.waitFor(() => expect(store.read.isPending(AT)).toBe(false))
    expect(store.read.getCommittedSetting(AT)).toBe(false)
    await vi.waitFor(() => expect(vault.reads.length).toBeGreaterThan(3))
    vault.reads.at(-1)?.resolve()
    during.call?.resolve()
    early.call?.resolve()
    await Promise.all([write, during.read, early.read])
    expect(history).toEqual([true, false])
  })

  it('keeps the old value, stops waiting and says why where the write fails', async () => {
    const vault = createHeldVault(true)
    const say = vi.fn()
    const store = createSettingsStore(words, say)
    const first = await startRead(store, vault)
    first.call?.resolve()
    await first.read
    const history: unknown[] = [store.read.getCommittedSetting(AT)]
    watch(
      () => store.read.getCommittedSetting(AT),
      (v) => history.push(v),
      { flush: 'sync' },
    )

    const write = store.write.writeSetting(AT, false)
    await vi.waitFor(() => expect(vault.writes).toHaveLength(1))
    vault.writes[0]?.reject(new Error('refused'))
    await vi.waitFor(() => expect(vault.reads.length).toBeGreaterThan(1))
    // The file holds the old value again, as the vault failed before it wrote.
    vault.reads.at(-1)?.resolve()
    await write
    expect(store.read.isPending(AT)).toBe(false)
    expect(say).toHaveBeenCalledWith(expect.stringMatching(/^u /), 'error')
    expect(history).toEqual([true])
  })
})
