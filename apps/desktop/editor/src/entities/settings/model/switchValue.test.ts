import { describe, expect, it, vi } from 'vitest'
import { watch } from 'vue'
import type { CoreAnswers } from '@numen/core-client/testing'
import { createAnsweredApi } from '@/testing/coreAnswers'
import { createSettingsStore } from './store'

const createAnsweredSettings = (answers: CoreAnswers) =>
  createAnsweredApi(answers).configuration.settings

/** The settings calls of the core the test answered last. */
let settings = createAnsweredSettings({})

const AT = ['p', 'enabled'] as const

const answerWith = (written: string, version = 0n) => ({
  settingsJson: written,
  version,
  path: '/f',
  plugins: [],
  app: [],
  appGroups: [],
  secretSections: [],
  secretMenus: [],
  credentialsPath: '',
})

describe('settings store', () => {
  it('shows a written value without reverting while the vault applies it', async () => {
    const file: Record<string, unknown> = { p: { enabled: true } }
    let version = 1n
    let applied = false
    let release: () => void = () => undefined
    const released = new Promise<void>((resolve) => {
      release = resolve
    })
    settings = createAnsweredSettings({
      settingsService: {
        getSettings: () => answerWith(JSON.stringify(file), version),
        writeSettings: async (request) => {
          for (const e of request.settings) {
            ;(file as { p: { enabled: boolean } }).p.enabled = JSON.parse(e.value)
          }
          version += 1n
          const written = { settingsJson: JSON.stringify(file), version }
          applied = true
          await released // the plugin host applies the change
          return written
        },
      },
    })
    const store = createSettingsStore(settings, { unturned: 'u', unreadSettings: 'r' }, vi.fn())
    await store.start()
    const values: unknown[] = []
    watch(
      () => store.read.getSetting(AT),
      (v) => values.push(v),
      { flush: 'sync' },
    )
    const write = store.write.writeSetting(AT, false)
    await vi.waitFor(() => expect(applied).toBe(true))
    await store.start()
    await store.start()
    release()
    await write
    await store.start()
    expect(values).toEqual([false])
  })
})

/** An answer the test lets through, out of order where it chooses. */
interface HeldCall {
  readonly resolve: () => void
  readonly reject: (error: unknown) => void
}

const createHeldVault = (isEnabled: boolean) => {
  // A read answers with the state and the version it had when it began; a write changes both when
  // it is let through.
  const file = { p: { enabled: isEnabled } }
  let version = 1n
  const reads: HeldCall[] = []
  const writes: HeldCall[] = []
  settings = createAnsweredSettings({
    settingsService: {
      getSettings: () =>
        new Promise<ReturnType<typeof answerWith>>((resolve, reject) => {
          const answer = answerWith(JSON.stringify(file), version)
          reads.push({ resolve: () => resolve(answer), reject })
        }),
      writeSettings: (request) =>
        new Promise<{ settingsJson: string; version: bigint }>((resolve, reject) => {
          writes.push({
            resolve: () => {
              for (const e of request.settings) file.p.enabled = JSON.parse(e.value) as boolean
              version += 1n
              resolve({ settingsJson: JSON.stringify(file), version })
            },
            reject,
          })
        }),
    },
  })
  return { reads, writes }
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
  it('shows each requested value from the click on, whatever order the reads answer in', async () => {
    const vault = createHeldVault(false)
    const store = createSettingsStore(settings, { unturned: 'u', unreadSettings: 'r' }, vi.fn())
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
      // A refresh that began before the click answers after the write.
      const refresh = await startRead(store, vault)
      const write = store.write.writeSetting(AT, isEnabled)
      await vi.waitFor(() => expect(vault.writes).toHaveLength(index + 1))
      vault.writes.at(-1)?.resolve()
      await write
      refresh.call?.resolve()
      await refresh.read
      expect(store.read.getSetting(AT)).toBe(isEnabled)
    }

    expect(history).toEqual([false, true, false])
    expect(vault.reads).toHaveLength(3)
  })
})

describe('the enabled switch waits for the answer of the vault', () => {
  const words = { unturned: 'u', unreadSettings: 'r' }

  it('keeps its position while the write is in flight, and moves once when the answer arrives', async () => {
    const vault = createHeldVault(true)
    const store = createSettingsStore(settings, words, vi.fn())
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
    await write
    expect(store.read.isPending(AT)).toBe(false)
    expect(store.read.getCommittedSetting(AT)).toBe(false)
    during.call?.resolve()
    early.call?.resolve()
    await Promise.all([during.read, early.read])
    expect(history).toEqual([true, false])
    expect(vault.reads).toHaveLength(3)
  })

  it('keeps the old value, stops waiting and says why where the write fails', async () => {
    const vault = createHeldVault(true)
    const say = vi.fn()
    const store = createSettingsStore(settings, words, say)
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
    await write
    expect(store.read.isPending(AT)).toBe(false)
    expect(say).toHaveBeenCalledWith(expect.stringMatching(/^u /), 'error')
    expect(history).toEqual([true])
  })
})
