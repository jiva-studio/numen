import { flushPromises, mount } from '@vue/test-utils'
import { defineComponent, h, ref, shallowRef } from 'vue'
import { describe, expect, it, vi } from 'vitest'
import { type Mode, type Theme } from '../lib/appearance'
import { useAppearancePalette } from './useAppearancePalette'

type AppearancePaletteDeps = Parameters<typeof useAppearancePalette>[0]

const WORDS = {
  apply: 'Apply',
  shipping: 'Built-in',
  owned: 'Custom',
  half: 'Mode',
  current: 'Current',
  pinned: 'The theme sets light or dark itself',
  modes: { system: 'System', light: 'Light', dark: 'Dark' },
}

const SEA: Theme = { name: 'mine:sea', title: 'sea', origin: 'mine', pluginId: '', isPinned: false }

const createPalette = (reason = '') => {
  const state = {
    themes: shallowRef<readonly Theme[]>([SEA]),
    applied: ref('preset:numen'),
    mode: ref<Mode>('system'),
  }
  const shown: [string, Mode][] = []
  const errors: [string, string][] = []
  const deps: AppearancePaletteDeps & {
    writeChoice: ReturnType<typeof vi.fn<() => Promise<string>>>
    onOpen: ReturnType<typeof vi.fn<() => Promise<void>>>
  } = {
    appearance: {
      state,
      show: async (theme: string, mode: Mode) => {
        shown.push([theme, mode])
      },
    },
    words: WORDS,
    getPluginName: (id: string) => id,
    writeChoice: vi.fn(async () => reason),
    onOpen: vi.fn(async () => undefined),
    write: (text: string, kind: string) => errors.push([text, kind]),
  }
  let palette: ReturnType<typeof useAppearancePalette> | undefined
  const wrapper = mount(
    defineComponent({
      setup: () => {
        palette = useAppearancePalette(deps)
        return () => h('div')
      },
    }),
  )
  if (!palette) throw new Error('the palette was not made')
  return { palette, deps, shown, errors, wrapper }
}

const pressToggle = (more: KeyboardEventInit = {}, preventDefault = false) => {
  const event = new KeyboardEvent('keydown', {
    key: 'p',
    ctrlKey: true,
    cancelable: true,
    ...more,
  })
  if (preventDefault) event.preventDefault()
  globalThis.dispatchEvent(event)
  return event
}

describe('the appearance palette', () => {
  it('opens and closes on Ctrl+P, reads again as it opens, and leaves other chords alone', async () => {
    const { palette, deps, errors, wrapper } = createPalette()
    expect(palette.query.value).toBe('')
    palette.query.value = 'stale'

    expect(pressToggle({ shiftKey: true }).defaultPrevented).toBe(false)
    expect(pressToggle({ altKey: true }).defaultPrevented).toBe(false)
    expect(pressToggle({ ctrlKey: false, metaKey: false }).defaultPrevented).toBe(false)
    expect(pressToggle({ key: 'x' }).defaultPrevented).toBe(false)
    pressToggle({}, true)
    expect(palette.isOpen.value).toBe(false)

    expect(pressToggle().defaultPrevented).toBe(true)
    expect(palette.isOpen.value).toBe(true)
    expect(palette.query.value).toBe('')
    expect(deps.onOpen).toHaveBeenCalledOnce()
    pressToggle({ ctrlKey: false, metaKey: true })
    expect(palette.isOpen.value).toBe(false)

    deps.onOpen.mockRejectedValueOnce(new Error('open failed'))
    pressToggle()
    await flushPromises()
    expect(errors).toStrictEqual([['open failed', 'error']])

    wrapper.unmount()
    pressToggle()
    expect(palette.isOpen.value).toBe(true)
  })

  it('writes a theme chosen with the mode in force, then shows it', async () => {
    const { palette, deps, shown } = createPalette()
    pressToggle()
    expect(palette.isOpen.value).toBe(true)

    await palette.onChoose(`theme:${SEA.name}`)
    expect(palette.isOpen.value).toBe(false)
    await palette.onChoose('mode:dark')

    expect(deps.writeChoice.mock.calls).toStrictEqual([
      [SEA.name, 'system'],
      ['preset:numen', 'dark'],
    ])
    expect(shown).toStrictEqual([
      [SEA.name, 'system'],
      ['preset:numen', 'dark'],
    ])
  })

  it('says why a choice was not written and shows nothing', async () => {
    const { palette, shown, errors } = createPalette('the settings are read-only')

    await palette.onChoose(`theme:${SEA.name}`)

    expect(errors).toStrictEqual([['the settings are read-only', 'error']])
    expect(shown).toStrictEqual([])
  })

  it('writes nothing for an item that is neither a theme nor a mode', async () => {
    const { palette, deps } = createPalette()

    await palette.onChoose('other:thing')

    expect(deps.writeChoice).not.toHaveBeenCalled()
    expect(palette.groups.value.map((one) => one.id)).toStrictEqual(['shipping', 'owned', 'half'])
  })
})
