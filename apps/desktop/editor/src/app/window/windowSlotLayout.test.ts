/** The layout of the bar slots as the window reads and writes it. Every plugin here is invented. */
import { describe, expect, it, vi } from 'vitest'
import { shallowRef } from 'vue'
import { createWindowSlotLayout, getLayoutEdits } from './windowSlotLayout'

const TASKS = 'org.example.tide/status'
const UPDATER = 'org.example.harbor/update'

describe('the layout of the window', () => {
  it('is empty before the settings are set, then follows them', () => {
    const stored = shallowRef<unknown>({ 'titleBar.status': [TASKS] })
    const layout = createWindowSlotLayout()

    expect(layout.getLayout()).toStrictEqual({})

    layout.setSettings({ getSetting: () => stored.value, writeSettings: vi.fn() })
    expect(layout.getLayout()).toStrictEqual({ 'titleBar.status': [TASKS] })

    stored.value = { 'statusBar.right': [UPDATER] }
    expect(layout.getLayout()).toStrictEqual({ 'statusBar.right': [UPDATER] })
  })

  it('writes the lists of a drop in one write, one edit per slot', async () => {
    const writeSettings = vi.fn(() => Promise.resolve())
    const layout = createWindowSlotLayout()
    layout.setSettings({ getSetting: () => undefined, writeSettings })

    await layout.writeLayout({ 'titleBar.status': [], 'statusBar.right': [UPDATER] })

    expect(writeSettings).toHaveBeenCalledTimes(1)
    expect(writeSettings).toHaveBeenCalledWith(
      getLayoutEdits({ 'titleBar.status': [], 'statusBar.right': [UPDATER] }),
    )
  })
})

describe('the edits of a drop', () => {
  it('put each list at app.layout.slots of its slot', () => {
    expect(
      getLayoutEdits({ 'statusBar.right': [UPDATER] }).map((one) => ({
        at: one.at,
        value: JSON.parse(one.value) as unknown,
      })),
    ).toStrictEqual([{ at: ['app', 'layout', 'slots', 'statusBar.right'], value: [UPDATER] }])
  })
})
