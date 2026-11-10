/** The keybindings the core lists reach the keys of the plugin host as the list changes. All names are invented. */
import { describe, expect, it, vi } from 'vitest'
import { nextTick, ref } from 'vue'
import { createPluginHost, getWhenValues } from '@numen/plugin-loader'
import type { DeclaredKeybinding } from '@/entities/plugin'
import { syncKeybindings } from './syncKeybindings'

const LEDGER = 'org.example.ledger'
const REVEAL_L: DeclaredKeybinding = {
  pluginId: LEDGER,
  key: 'mod+l',
  command: 'app:file.reveal',
  values: { kind: 'plex' },
}
const FIND_F: DeclaredKeybinding = {
  pluginId: LEDGER,
  key: 'f',
  command: 'app:file.reveal',
  values: { kind: 'files' },
  when: '!window.isFieldFocused',
}
const OPEN_T: DeclaredKeybinding = {
  pluginId: 'org.example.tide',
  key: 'mod+t',
  command: 'app:file.open',
  values: { placement: 'split' },
}

const TARGET = { path: 'Tides.md', title: 'Tides', isFolder: false, others: [], tab: '' }

const create = (list?: readonly DeclaredKeybinding[]) => {
  const window = { 'window.isFieldFocused': false }
  const host = createPluginHost({
    registerKind: () => () => undefined,
    registerExtensionReader: () => () => undefined,
    openTab: async () => 'tab-1',
    getSlotWhenValues: (target) => getWhenValues({ window, target, contentTypes: {} }),
  })
  const keybindings = ref<readonly DeclaredKeybinding[] | undefined>(list)
  const plugins = ref<readonly object[]>([])
  syncKeybindings(host.keybindings, keybindings, plugins)
  const runApp = vi.fn()
  const press = (code: string, init: KeyboardEventInit = {}) =>
    host.commands.handleShortcut(
      new KeyboardEvent('keydown', { code, ...init }),
      async () => ({}) as never,
      {
        target: TARGET,
        runApp,
      },
    )
  return { window, keybindings, plugins, runApp, press }
}

describe('the keybindings of the plugins the core lists', () => {
  it('fire the key of each plugin with its values once listed, and no key before', async () => {
    const one = create()
    expect(one.press('KeyL', { ctrlKey: true })).toBe(false)

    one.keybindings.value = [REVEAL_L, OPEN_T, FIND_F]
    await nextTick()

    expect(one.press('KeyL', { ctrlKey: true })).toBe(true)
    expect(one.press('KeyT', { ctrlKey: true })).toBe(true)
    expect(one.runApp.mock.calls).toStrictEqual([
      ['app:file.reveal', { kind: 'plex' }],
      ['app:file.open', { placement: 'split' }],
    ])
  })

  it('fire a key listed at the start', () => {
    const one = create([REVEAL_L])

    expect(one.press('KeyL', { ctrlKey: true })).toBe(true)
    expect(one.runApp).toHaveBeenCalledExactlyOnceWith('app:file.reveal', { kind: 'plex' })
  })

  it('keep the condition of a key', () => {
    const one = create([FIND_F])
    one.window['window.isFieldFocused'] = true

    expect(one.press('KeyF')).toBe(false)

    one.window['window.isFieldFocused'] = false
    expect(one.press('KeyF')).toBe(true)
  })

  it('drop the keys of a plugin the list drops, and keep those of the others', async () => {
    const one = create([REVEAL_L, OPEN_T])

    one.keybindings.value = [OPEN_T]
    await nextTick()

    expect(one.press('KeyL', { ctrlKey: true })).toBe(false)
    expect(one.press('KeyT', { ctrlKey: true })).toBe(true)
  })

  it('keep the keys as a plugin starts or stops', async () => {
    const one = create([REVEAL_L])

    one.plugins.value = [{ id: LEDGER, isActive: true }]
    await nextTick()

    expect(one.press('KeyL', { ctrlKey: true })).toBe(true)
  })
})
