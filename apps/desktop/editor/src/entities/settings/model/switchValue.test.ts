import { describe, expect, it } from 'vitest'
import { watch } from 'vue'
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
    const store = createSettingsStore(core, { unturned: 'u', unreadSettings: 'r' }, () => {})
    await store.start()
    const values: unknown[] = []
    watch(
      () => store.getSetting(AT),
      (v) => values.push(v),
      { flush: 'sync' },
    )
    const poll = setInterval(() => void store.start(), 20)
    await store.writeSetting(AT, false)
    await sleep(300)
    clearInterval(poll)
    expect(values).toEqual([false])
  })
})
