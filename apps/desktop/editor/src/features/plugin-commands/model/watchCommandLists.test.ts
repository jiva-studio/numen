/** When the lists of commands the plugins declare are read again. */
import { describe, expect, it, vi } from 'vitest'
import { nextTick, ref } from 'vue'
import { watchCommandLists } from './watchCommandLists'
import type { DeclaredCommands } from './declaredCommands'
import type { JsonValue } from '@/shared/api/json'

const PLEX = 'org.example.plex'

function setup(load = vi.fn(async (): Promise<void> => undefined)) {
  const isOpen = ref(false)
  const writeMessage = vi.fn()
  const plugins = ref([{ id: PLEX, isActive: false }])
  const settings = ref<Record<string, JsonValue>>({})
  const getCommittedSetting = (path: readonly string[]) => settings.value[path.join('.')]
  watchCommandLists(
    {
      declaredCommands: { load } as unknown as DeclaredCommands,
      writeMessage,
      plugins,
      getCommittedSetting,
    },
    isOpen,
  )
  const switchOff = async () => {
    settings.value = { ...settings.value, [`${PLEX}.enabled`]: false }
    await nextTick()
  }
  return { load, switchOff, plugins, isOpen, writeMessage }
}

describe('the lists of declared commands', () => {
  it('are read as the window starts', () => {
    expect(setup().load).toHaveBeenCalledTimes(1)
  })

  it('are read again when a plugin without a backend is switched off', async () => {
    const { load, switchOff } = setup()

    await switchOff()

    expect(load).toHaveBeenCalledTimes(2)
  })

  it('are read again when a plugin starts', async () => {
    const { load, plugins } = setup()

    plugins.value = [{ id: PLEX, isActive: true }]
    await nextTick()

    expect(load).toHaveBeenCalledTimes(2)
  })

  it('are read each time the palette opens, and not as it closes', async () => {
    const { load, isOpen } = setup()

    isOpen.value = true
    await nextTick()
    expect(load).toHaveBeenCalledTimes(2)

    isOpen.value = false
    await nextTick()
    expect(load).toHaveBeenCalledTimes(2)

    isOpen.value = true
    await nextTick()
    expect(load).toHaveBeenCalledTimes(3)
  })

  it('that cannot be read are written as errors', async () => {
    const { writeMessage, isOpen } = setup(
      vi.fn(async () => {
        throw new Error('the application did not answer')
      }),
    )

    isOpen.value = true
    await vi.waitFor(() => expect(writeMessage).toHaveBeenCalledTimes(2))

    expect(writeMessage.mock.calls.map(([, kind]) => kind)).toStrictEqual(['error', 'error'])
  })
})
