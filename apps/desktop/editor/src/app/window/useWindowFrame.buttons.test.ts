/**
 * The setting "Window buttons" decides where the window draws its buttons on Linux. Windows always
 * draws them at the right and macOS always uses the system buttons, whatever the setting says.
 * Every value here is invented.
 */
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { nextTick, ref } from 'vue'
import { flushPromises } from '@vue/test-utils'
import { useWindowFrame, type FrameRuntime } from './useWindowFrame'
import { useSettings } from './useSettings'
import { useWindowTabs } from '@/entities/tab'
import { WORDS } from '@/shared/words'
import type { SettingEdit } from '@/entities/settings'
import type { SettingsPort } from '@/app/ports/settings'
import type { MessageLog } from '@/shared/notices/messages'

type Buttons = 'windows' | 'mac'

const createRuntime = (platform: 'windows' | 'mac' | 'linux'): FrameRuntime => ({
  Window: { IsMaximised: async () => false },
  System: {
    IsWindows: () => platform === 'windows',
    IsMac: () => platform === 'mac',
    IsLinux: () => platform === 'linux',
  },
})

describe('the window buttons on Linux', () => {
  it('stand at the right when the setting says windows', async () => {
    const frame = useWindowFrame(async () => createRuntime('linux'), ref<Buttons>('windows'))
    await flushPromises()

    expect(frame.controls.value).toBe('windows')
  })

  it('stand at the left when the setting says mac', async () => {
    const frame = useWindowFrame(async () => createRuntime('linux'), ref<Buttons>('mac'))
    await flushPromises()

    expect(frame.controls.value).toBe('mac')
  })

  it('move to the left when the setting changes to mac and back to the right when it changes to windows', async () => {
    const buttons = ref<Buttons>('windows')
    const frame = useWindowFrame(async () => createRuntime('linux'), buttons)
    await flushPromises()

    buttons.value = 'mac'
    await nextTick()
    expect(frame.controls.value).toBe('mac')

    buttons.value = 'windows'
    await nextTick()
    expect(frame.controls.value).toBe('windows')
  })
})

describe('the window buttons on the other platforms', () => {
  it('stand at the right on Windows when the setting says mac', async () => {
    const frame = useWindowFrame(async () => createRuntime('windows'), ref<Buttons>('mac'))
    await flushPromises()

    expect(frame.controls.value).toBe('windows')
  })

  it('stay with the system on macOS when the setting says windows', async () => {
    const frame = useWindowFrame(async () => createRuntime('mac'), ref<Buttons>('windows'))
    await flushPromises()

    expect(frame.controls.value).toBe('native-mac')
  })

  it('stay with the system on macOS when the setting says mac', async () => {
    const frame = useWindowFrame(async () => createRuntime('mac'), ref<Buttons>('mac'))
    await flushPromises()

    expect(frame.controls.value).toBe('native-mac')
  })

  it('do not move on Windows when the setting changes', async () => {
    const buttons = ref<Buttons>('windows')
    const frame = useWindowFrame(async () => createRuntime('windows'), buttons)
    await flushPromises()

    buttons.value = 'mac'
    await nextTick()

    expect(frame.controls.value).toBe('windows')
  })
})

const createFeed = () => {
  const pending: ((batch: readonly SettingEdit[]) => void)[] = []
  return {
    open: (): AsyncIterable<readonly SettingEdit[]> => ({
      [Symbol.asyncIterator]: () => ({
        next: async () => ({
          done: false,
          value: await new Promise<readonly SettingEdit[]>((resolve) => pending.push(resolve)),
        }),
      }),
    }),
    push: (batch: readonly SettingEdit[]) => pending.shift()?.(batch),
  }
}

const settle = async () => {
  for (let step = 0; step < 10; step++) await Promise.resolve()
  await nextTick()
}

describe('the window buttons of a window that follows the settings of the shell', () => {
  beforeEach(() => {
    vi.useFakeTimers()
  })

  afterEach(() => {
    vi.useRealTimers()
  })

  const create = (stored: Record<string, unknown>) => {
    let file: Record<string, unknown> = { app: stored }
    const feed = createFeed()
    const core = {
      getSettings: vi.fn(async () => ({
        written: JSON.stringify(file),
        path: '/config/numen.json',
        models: [],
        plugins: [],
        app: [],
      })),
      getSyncEnabled: vi.fn(async () => true),
      updateSettings: vi.fn(async () => undefined),
      watchSettings: vi.fn(() => feed.open()),
    } as unknown as SettingsPort
    const settings = useSettings({
      core,
      words: WORDS,
      log: { getWriter: () => vi.fn() } as unknown as MessageLog,
      windowTabs: useWindowTabs(),
      onSizeChanged: vi.fn(),
    })
    return {
      settings,
      feed,
      store: (next: Record<string, unknown>) => {
        file = { app: next }
      },
    }
  }

  it('stand at the right on Linux while the file says nothing about them', async () => {
    const { settings } = create({})
    const frame = useWindowFrame(async () => createRuntime('linux'), settings.windowButtons)
    await settings.start()
    await flushPromises()

    expect(frame.controls.value).toBe('windows')
    settings.close()
  })

  it('stand at the left on Linux when the file says mac', async () => {
    const { settings } = create({ window: { buttons: 'mac' } })
    const frame = useWindowFrame(async () => createRuntime('linux'), settings.windowButtons)
    await settings.start()
    await flushPromises()

    expect(frame.controls.value).toBe('mac')
    settings.close()
  })

  it('move to the left on Linux when the event of the setting arrives, without a restart', async () => {
    const { settings, feed, store } = create({})
    const frame = useWindowFrame(async () => createRuntime('linux'), settings.windowButtons)
    await settings.start()
    await flushPromises()
    expect(frame.controls.value).toBe('windows')

    store({ window: { buttons: 'mac' } })
    feed.push([{ at: ['app', 'window', 'buttons'], value: '"mac"' }])
    await vi.advanceTimersByTimeAsync(0)
    await settle()

    expect(frame.controls.value).toBe('mac')
    settings.close()
  })

  it('stand at the right on Linux when the file holds an unknown style', async () => {
    const { settings } = create({ window: { buttons: 'floating' } })
    const frame = useWindowFrame(async () => createRuntime('linux'), settings.windowButtons)
    await settings.start()
    await flushPromises()

    expect(frame.controls.value).toBe('windows')
    settings.close()
  })
})
