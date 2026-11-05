/** The commands the plugins declare, read again as the palette opens. */
import { describe, expect, it, vi } from 'vitest'
import { nextTick, ref, shallowRef, type Ref } from 'vue'
import { useCommands, type CommandsDepsOptions } from './useCommands'
import type { DeclaredCommands } from './declaredCommands'

const isOpen = ref(false)

vi.mock('@/features/command-palette', () => ({
  ANSWER_WORDS: {},
  isChord: vi.fn(),
  findChordCommand: vi.fn(),
  runInvocation: vi.fn(),
  useSearch: () => ({ setOpen: vi.fn() }),
  useCommandPalette: () => ({ open: isOpen }),
}))
vi.mock('./commandDeps', () => ({ createCommandDeps: () => ({}) }))

const setup = (
  load: () => Promise<void>,
  loadMenu?: (file: string) => Promise<void>,
  plugins?: Ref<{ id: string; isActive: boolean }[]>,
) => {
  const messages: [string, string][] = []
  const declaredCommands = { paletteCommands: shallowRef([]), load } as unknown as DeclaredCommands
  const fileMenuCommands = loadMenu
    ? ({
        paletteCommands: shallowRef([]),
        load: loadMenu,
      } as unknown as CommandsDepsOptions['fileMenuCommands'])
    : undefined
  useCommands({
    windowTabs: {},
    rest: { plugins },
    getTarget: () => ({ file: 'books/Scan.pdf' }),
    writeMessage: (text: string, kind: string) => messages.push([text, kind]),
    declaredCommands,
    fileMenuCommands,
  } as unknown as CommandsDepsOptions)
  return messages
}

describe('the declared commands', () => {
  it('are read again when a plugin starts or stops', async () => {
    isOpen.value = false
    const load = vi.fn(() => Promise.resolve())
    const plugins = ref([{ id: 'md.tree', isActive: false }])
    setup(load, undefined, plugins)
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
    const loadMenu = vi.fn(() => Promise.resolve())
    setup(load, loadMenu)

    expect(loadMenu).not.toHaveBeenCalled()
    isOpen.value = true
    await nextTick()
    expect(loadMenu).toHaveBeenCalledTimes(1)
    expect(loadMenu).toHaveBeenCalledWith('books/Scan.pdf')
    isOpen.value = false
    await nextTick()
    expect(loadMenu).toHaveBeenCalledTimes(1)
    isOpen.value = true
    await nextTick()

    expect(load).toHaveBeenCalledTimes(3)
    expect(loadMenu).toHaveBeenCalledTimes(2)
  })

  it('that cannot be read say why', async () => {
    isOpen.value = false
    const messages = setup(
      async () => {
        throw new Error('the application did not answer')
      },
      async () => {
        throw new Error('the menu did not answer')
      },
    )

    isOpen.value = true
    await nextTick()
    await nextTick()

    expect(messages.map(([, kind]) => kind)).toStrictEqual(['error', 'error', 'error'])
  })
})
