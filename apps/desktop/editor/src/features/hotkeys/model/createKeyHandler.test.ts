/** The keys of the window: the tabs take a key first, then the keybindings the plugins declare. */
import { describe, expect, it, vi } from 'vitest'
import { ref } from 'vue'
import { createPluginHost, getWhenValues } from '@numen/plugin-loader'
import { COMMAND, type CommandTarget } from '@/shared/model/command'
import type { DeclaredKeybinding } from '@/entities/plugin'
import { createAppKeyRunner, createKeyHandler } from './createKeyHandler'
import { syncKeybindings } from './syncKeybindings'

const TARGET = {
  tab: 'note:1',
  kind: 'note',
  path: 'Tides.md',
  title: 'Tides',
  file: 'Tides.md',
  vault: { id: 'sea', name: 'Sea' },
  isReady: true,
} as CommandTarget

const bind = (
  key: string,
  command: string,
  values?: Record<string, string>,
): DeclaredKeybinding => ({
  pluginId: 'org.example.tide',
  key,
  command,
  values: values ?? {},
})

const setup = (keybindings: readonly DeclaredKeybinding[], takesKey = false) => {
  const host = createPluginHost({
    target: 'desktop.editor',
    registerKind: () => () => undefined,
    registerExtensionReader: () => () => undefined,
    openTab: () => Promise.resolve('tab-1'),
    getSlotWhenValues: (target) => getWhenValues({ window: {}, target, contentTypes: {} }),
  })
  const calls = {
    toggleSearch: vi.fn(),
    toggleCommands: vi.fn(),
    runCommand: vi.fn(),
    runWithValues: vi.fn(),
  }
  syncKeybindings(host.keybindings, ref(keybindings), ref([]))
  const onKeyDown = createKeyHandler({
    windowTabs: { panes: { onKeyPress: () => takesKey } },
    runApp: createAppKeyRunner({ ...calls, getTarget: () => TARGET }),
    getTarget: () => TARGET,
    handleShortcut: (event, _getTarget, runApp) =>
      host.commands.handleShortcut(event, async () => ({}) as never, {
        target: {
          path: TARGET.path,
          title: TARGET.title,
          isFolder: false,
          others: [],
          tab: TARGET.tab,
        },
        runApp,
      }),
  })
  const press = (init: KeyboardEventInit = { code: 'KeyL', ctrlKey: true }) => {
    const event = new KeyboardEvent('keydown', { cancelable: true, ...init })
    onKeyDown(event)
    return event
  }
  return { press, onKeyDown, ...calls }
}

describe('a key press', () => {
  it('does nothing where the event was already handled', () => {
    const one = setup([bind('mod+l', COMMAND.searchVault)])
    const event = new KeyboardEvent('keydown', { code: 'KeyL', ctrlKey: true, cancelable: true })
    event.preventDefault()

    one.onKeyDown(event)

    expect(one.toggleSearch).not.toHaveBeenCalled()
  })

  it('stops at the tabs where they take the key', () => {
    const one = setup([bind('mod+l', COMMAND.searchVault)], true)

    const event = one.press()

    expect(event.defaultPrevented).toBe(true)
    expect(one.toggleSearch).not.toHaveBeenCalled()
  })

  it('is left alone where no keybinding matches', () => {
    const one = setup([bind('mod+l', COMMAND.searchVault)])

    const event = one.press({ code: 'KeyZ', ctrlKey: true })

    expect(event.defaultPrevented).toBe(false)
  })
})

describe('the command a key is bound to', () => {
  it('toggles the search for the search command', () => {
    const one = setup([bind('mod+l', COMMAND.searchVault)])

    const event = one.press()

    expect(event.defaultPrevented).toBe(true)
    expect(one.toggleSearch).toHaveBeenCalledOnce()
    expect(one.runCommand).not.toHaveBeenCalled()
  })

  it('toggles the commands for the command that opens them', () => {
    const one = setup([bind('mod+l', COMMAND.openCommands)])

    one.press()

    expect(one.toggleCommands).toHaveBeenCalledOnce()
  })

  it('runs any other command of the window over the target in front', () => {
    const one = setup([bind('mod+l', 'app:note.save')])

    one.press()

    expect(one.runCommand).toHaveBeenCalledExactlyOnceWith('app:note.save', TARGET)
  })

  it('runs a command with the values its keybinding carries', () => {
    const one = setup([bind('mod+l', 'app:file.reveal', { kind: 'plex' })])

    one.press()

    expect(one.runWithValues).toHaveBeenCalledExactlyOnceWith('app:file.reveal', TARGET, {
      kind: 'plex',
    })
    expect(one.runCommand).not.toHaveBeenCalled()
  })
})
