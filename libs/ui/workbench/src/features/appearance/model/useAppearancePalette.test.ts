import { mount } from '@vue/test-utils'
import { defineComponent, h, ref, shallowRef } from 'vue'
import { describe, expect, it, vi } from 'vitest'
import type { Mode, Theme } from '../lib/appearance'
import { useAppearancePalette, type AppearancePaletteDeps } from './useAppearancePalette'

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
  const errors: string[] = []
  const deps = {
    appearance: {
      state,
      show: async (theme: string, mode: Mode) => {
        shown.push([theme, mode])
      },
    } as unknown as AppearancePaletteDeps['appearance'],
    words: WORDS,
    getPluginName: (id: string) => id,
    writeChoice: vi.fn(async () => reason),
    onOpen: vi.fn(async () => undefined),
    write: (text: string) => errors.push(text),
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

const pressToggle = (more: KeyboardEventInit = {}) => {
  const event = new KeyboardEvent('keydown', {
    key: 'p',
    ctrlKey: true,
    cancelable: true,
    ...more,
  })
  globalThis.dispatchEvent(event)
  return event
}

describe('the appearance palette', () => {
  it('opens and closes on Ctrl+P, reads again as it opens, and leaves other chords alone', () => {
    const { palette, deps, wrapper } = createPalette()
    palette.query.value = 'stale'

    expect(pressToggle({ shiftKey: true }).defaultPrevented).toBe(false)
    expect(palette.isOpen.value).toBe(false)
    expect(pressToggle().defaultPrevented).toBe(true)
    expect(palette.isOpen.value).toBe(true)
    expect(palette.query.value).toBe('')
    expect(deps.onOpen).toHaveBeenCalledOnce()
    pressToggle({ ctrlKey: false, metaKey: true })
    expect(palette.isOpen.value).toBe(false)

    wrapper.unmount()
    pressToggle()
    expect(palette.isOpen.value).toBe(false)
  })

  it('writes a theme chosen with the mode in force, then shows it', async () => {
    const { palette, deps, shown } = createPalette()

    await palette.onChoose(`theme:${SEA.name}`)
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

    expect(errors).toStrictEqual(['the settings are read-only'])
    expect(shown).toStrictEqual([])
  })

  it('writes nothing for an item that is neither a theme nor a mode', async () => {
    const { palette, deps } = createPalette()

    await palette.onChoose('other:thing')

    expect(deps.writeChoice).not.toHaveBeenCalled()
    expect(palette.groups.value.map((one) => one.id)).toStrictEqual(['shipping', 'owned', 'half'])
  })
})
