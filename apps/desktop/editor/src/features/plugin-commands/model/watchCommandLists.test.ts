/** When the lists of commands the plugins declare are read again. */
import { describe, expect, it, vi } from 'vitest'
import { nextTick, ref } from 'vue'
import { watchCommandLists } from './watchCommandLists'
import type { DeclaredCommands } from './declaredCommands'

const PLEX = 'org.example.plex'

function setup() {
  const load = vi.fn(async () => undefined)
  const plugins = ref([{ id: PLEX, isActive: false }])
  const settings = ref<Record<string, unknown>>({})
  const getCommittedSetting = (path: readonly string[]) => settings.value[path.join('.')]
  watchCommandLists(
    {
      declaredCommands: { load } as unknown as DeclaredCommands,
      writeMessage: vi.fn(),
      plugins,
      getCommittedSetting,
    },
    ref(false),
  )
  const switchOff = async () => {
    settings.value = { ...settings.value, [`${PLEX}.enabled`]: false }
    await nextTick()
  }
  return { load, switchOff, plugins }
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
})
