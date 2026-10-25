/** The commands the plugins declare, read again as the palette opens. */
import { describe, expect, it, vi } from 'vitest'
import { nextTick, ref, shallowRef } from 'vue'
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

const setup = (load: () => Promise<void>) => {
  const messages: [string, string][] = []
  const declaredCommands = { paletteCommands: shallowRef([]), load } as unknown as DeclaredCommands
  useCommands({
    windowTabs: {},
    rest: {},
    writeMessage: (text: string, kind: string) => messages.push([text, kind]),
    declaredCommands,
  } as unknown as CommandsDepsOptions)
  return messages
}

describe('the declared commands', () => {
  it('are read as the window starts and each time the palette opens, and not as it closes', async () => {
    isOpen.value = false
    const load = vi.fn(async () => {})
    setup(load)

    isOpen.value = true
    await nextTick()
    isOpen.value = false
    await nextTick()
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
