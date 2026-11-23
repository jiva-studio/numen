import { flushPromises, mount } from '@vue/test-utils'
import { defineComponent, h } from 'vue'
import { describe, expect, it, vi } from 'vitest'
import { createCoreHandlerTransport } from '@numen/core-client/testing'
import { createWindowClients } from '@/shared/api/clients'
import { useSlotLayouts } from './useSlotLayouts'

const BAR = 'org.example.bar'
const BUTTONS = [{ id: 'org.example.a/command/org.example.a:one' }, { id: '---' }]

const plugin = (id: string, settings: object[]) => ({ id, settings })
const slotsSetting = (value: unknown) => ({ type: 'slots', valueJson: JSON.stringify(value) })

function mountLayouts(
  getSettings: () => object,
  watchSettings: (signal: AbortSignal) => AsyncGenerator<object>,
) {
  const clients = createWindowClients(
    createCoreHandlerTransport({
      getSettings,
      watchSettings: (_request: unknown, context: { signal: AbortSignal }) =>
        watchSettings(context.signal),
    }),
  )
  let layouts!: ReturnType<typeof useSlotLayouts>
  const wrapper = mount(
    defineComponent({
      setup() {
        layouts = useSlotLayouts(clients.settingsService, () => new Promise<void>(() => undefined))
        return () => h('div')
      },
    }),
  )
  return { wrapper, layouts }
}

describe('the layout of the slots a plugin draws', () => {
  it('is read from the setting of the type slots of the plugin when the window mounts', async () => {
    const { layouts } = mountLayouts(
      () => ({
        plugins: [
          plugin(BAR, [
            { type: 'bool', valueJson: 'true' },
            slotsSetting({ 'bar.buttons': BUTTONS }),
          ]),
          plugin('org.example.plain', [{ type: 'bool', valueJson: 'true' }]),
          plugin('org.example.broken', [{ type: 'slots', valueJson: '[1]' }]),
        ],
      }),
      async function* () {
        yield { changed: [] }
      },
    )

    await flushPromises()

    expect(layouts.getSlotLayout(BAR)).toStrictEqual({ 'bar.buttons': BUTTONS })
    expect(layouts.getSlotLayout('org.example.plain')).toBeUndefined()
    expect(layouts.getSlotLayout('org.example.broken')).toBeUndefined()
  })

  it('is read again when the settings change', async () => {
    let stored: unknown = {}
    let change: () => void = () => undefined
    const { layouts } = mountLayouts(
      () => ({ plugins: [plugin(BAR, [slotsSetting(stored)])] }),
      async function* (signal) {
        yield { changed: [] }
        await new Promise<void>((resolve) => {
          change = resolve
          signal.addEventListener('abort', () => resolve())
        })
        yield { changed: [{ keyPath: [BAR, 'slots'], value: '{}' }] }
      },
    )
    await flushPromises()
    stored = { 'bar.buttons': BUTTONS }

    change()
    await flushPromises()

    expect(layouts.getSlotLayout(BAR)).toStrictEqual({ 'bar.buttons': BUTTONS })
  })

  it('stops following the settings when the window unmounts', async () => {
    const aborted = vi.fn()
    const { wrapper } = mountLayouts(
      () => ({ plugins: [] }),
      async function* (signal) {
        signal.addEventListener('abort', aborted)
        yield { changed: [] }
        await new Promise<void>((resolve) => signal.addEventListener('abort', () => resolve()))
      },
    )
    await flushPromises()

    wrapper.unmount()

    expect(aborted).toHaveBeenCalledTimes(1)
  })
})
