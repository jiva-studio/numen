/** The keybindings the core lists reach the plugin host as the list changes. All names are invented. */
import { describe, expect, it, vi } from 'vitest'
import { nextTick, ref } from 'vue'
import type { DeclaredKeybinding } from '@/entities/plugin'
import { syncKeybindings } from './syncKeybindings'

const LEDGER_OPEN: DeclaredKeybinding = { pluginId: 'org.example.ledger', key: 'mod+l', command: 'ledger.open' }
const LEDGER_FIND: DeclaredKeybinding = {
  pluginId: 'org.example.ledger',
  key: 'f',
  command: 'ledger.find',
  when: '!window.isFieldFocused',
}
const TIDE_OPEN: DeclaredKeybinding = {
  pluginId: 'org.example.tide',
  key: 'mod+t',
  command: 'app:file.open',
  values: { placement: 'split' },
}

const create = (list?: readonly DeclaredKeybinding[]) => {
  const host = { registerKeybindings: vi.fn(), unregisterKeybindings: vi.fn() }
  const keybindings = ref<readonly DeclaredKeybinding[] | undefined>(list)
  const plugins = ref<readonly unknown[]>([])
  syncKeybindings(host, keybindings, plugins)
  return { host, keybindings, plugins }
}

describe('the keybindings of the plugins the core lists', () => {
  it('are registered for each plugin before any plugin starts', async () => {
    const { host, keybindings } = create()
    expect(host.registerKeybindings).not.toHaveBeenCalled()

    keybindings.value = [LEDGER_OPEN, TIDE_OPEN, LEDGER_FIND]
    await nextTick()

    expect(host.registerKeybindings).toHaveBeenCalledTimes(2)
    expect(host.registerKeybindings).toHaveBeenCalledWith('org.example.ledger', [LEDGER_OPEN, LEDGER_FIND])
    expect(host.registerKeybindings).toHaveBeenCalledWith('org.example.tide', [TIDE_OPEN])
  })

  it('are registered when the list is held already', () => {
    const { host } = create([LEDGER_OPEN])

    expect(host.registerKeybindings).toHaveBeenCalledExactlyOnceWith('org.example.ledger', [LEDGER_OPEN])
  })

  it('are unregistered for a plugin the list no longer holds', async () => {
    const { host, keybindings } = create([LEDGER_OPEN, TIDE_OPEN])

    keybindings.value = [TIDE_OPEN]
    await nextTick()

    expect(host.unregisterKeybindings).toHaveBeenCalledExactlyOnceWith('org.example.ledger')
  })

  it('are registered again as a plugin starts or stops, which drops the keys of the plugin that stops', async () => {
    const { host, plugins } = create([LEDGER_OPEN])
    host.registerKeybindings.mockClear()

    plugins.value = [{ id: 'org.example.ledger', isActive: true }]
    await nextTick()

    expect(host.registerKeybindings).toHaveBeenCalledExactlyOnceWith('org.example.ledger', [LEDGER_OPEN])
  })
})
