/** The commands the plugins declare, read again as the palette opens. */
import { describe, expect, it, vi } from 'vitest'
import { nextTick, ref, shallowRef, type Ref } from 'vue'
import { createCommands, type CommandsDepsOptions } from './createCommands'
import type { DeclaredCommands } from '@/features/plugin-commands'

const isOpen = ref(false)

vi.mock('@/features/command-palette', () => ({
  PALETTE_WORDS: {},
  runInvocation: vi.fn(),
  useSearch: () => ({ setOpen: vi.fn() }),
  createCommandPalette: () => ({ palette: { open: isOpen } }),
}))
vi.mock('../tabs/commandDeps', () => ({
  createCommandDeps: () => ({}),
  createCommandNotes: () => ({}),
}))

const setup = (load: () => Promise<void>, plugins?: Ref<{ id: string; isActive: boolean }[]>) => {
  const messages: [string, string][] = []
  const declaredCommands = { paletteCommands: shallowRef([]), load } as unknown as DeclaredCommands
  createCommands({
    windowTabs: {},
    rest: { write: { writeSettings: () => undefined }, state: { plugins } },
    getTarget: () => ({ file: 'books/Scan.pdf' }),
    writeMessage: (text: string, kind: string) => messages.push([text, kind]),
    declaredCommands,
  } as unknown as CommandsDepsOptions)
  return messages
}

describe('the declared commands', () => {
  it('are read again when a plugin starts or stops', async () => {
    isOpen.value = false
    const load = vi.fn(() => Promise.resolve())
    const plugins = ref([{ id: 'md.tree', isActive: false }])
    setup(load, plugins)
    expect(load).toHaveBeenCalledTimes(1)

    plugins.value = [{ id: 'md.tree', isActive: true }]
    await nextTick()
    expect(load).toHaveBeenCalledTimes(2)

    plugins.value = [{ id: 'md.tree', isActive: true }]
    await nextTick()
    expect(load).toHaveBeenCalledTimes(2)

    plugins.value = [{ id: 'md.tree', isActive: false }]
    await nextTick()
    expect(load).toHaveBeenCalledTimes(3)
  })

  it('are read as the window starts and each time the palette opens, and not as it closes', async () => {
    isOpen.value = false
    const load = vi.fn(() => Promise.resolve())
    setup(load)

    expect(load).toHaveBeenCalledTimes(1)
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

  it('that cannot be read say why', async () => {
    isOpen.value = false
    const messages = setup(async () => {
      throw new Error('the application did not answer')
    })

    isOpen.value = true
    await nextTick()
    await nextTick()

    expect(messages.map(([, kind]) => kind)).toStrictEqual(['error', 'error'])
  })
})
