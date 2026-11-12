/**
 * Every way to show a note in the plex or the file tree, through the real pieces: the plugin host
 * with the keybindings and slots of the manifests, the palette model, the runner and the handler
 * table. Only the navigation of a tab is a double, so a path that stops reaching it fails here.
 */
import { beforeEach, describe, expect, it, vi } from 'vitest'
import { SLOT, type CommandTarget as RunTarget, type SlotItem } from '@numen/plugin-loader'
import {
  createRealWindow,
  readManifest,
  windowValues,
  type ManifestKeybinding,
} from '@/testing/realCommands'
import { createWindowCommands } from '@/features/plugin-host'

const REEF: RunTarget = { path: 'Reef.md', title: 'Reef', isFolder: false, others: [], tab: '' }
const REVEAL = 'app:file.reveal'

const MANIFESTS = [readManifest('views-plex-graph'), readManifest('views-file-tree')]

const setup = () => {
  const navigateTab = vi.fn(async (_kind: string, _path: string) => undefined)
  return { navigateTab, ...createRealWindow({ manifests: MANIFESTS, navigateTab }) }
}

const press = (code: string, init: KeyboardEventInit = {}) =>
  new KeyboardEvent('keydown', { code, cancelable: true, ...init })

const getRevealKeys = (): readonly ManifestKeybinding[] =>
  MANIFESTS.flatMap((one) => one.contributes.keybindings ?? []).filter(
    (one) => one.command === REVEAL,
  )

const getRevealItems = (items: readonly SlotItem[]) =>
  items.flatMap((one) => (one.kind === 'command' && one.command === REVEAL ? [one] : []))

beforeEach(() => {
  windowValues['window.isFieldFocused'] = false
  windowValues['window.isPaletteOpen'] = false
})

describe('the keys that show a note', () => {
  it('are declared by the manifests with the kind of the tab', () => {
    expect(
      getRevealKeys()
        .map((one) => one.values?.['kind'])
        .sort(),
    ).toStrictEqual(['files', 'plex'])
  })

  it.each([
    ['plex', 'KeyP', { ctrlKey: true, shiftKey: true }],
    ['files', 'KeyE', { ctrlKey: true, altKey: true }],
  ])('show the note in the tab of kind %s', async (kind, code, init) => {
    const one = setup()
    const event = press(code, init)

    one.used.onKeyDown(event)

    await vi.waitFor(() =>
      expect(one.navigateTab).toHaveBeenCalledExactlyOnceWith(kind, 'Tides.md'),
    )
    expect(event.defaultPrevented).toBe(true)
    expect(one.messages.filter((text) => text !== '')).toStrictEqual([])
  })

  it('leave a key that matches nothing to the page', () => {
    const one = setup()
    const event = press('KeyY', { ctrlKey: true, shiftKey: true })

    one.used.onKeyDown(event)

    expect(event.defaultPrevented).toBe(false)
    expect(one.navigateTab).not.toHaveBeenCalled()
  })
})

describe('the entries of a context menu that show a note', () => {
  it.each([
    [SLOT.plexGraphContext, 'files'],
    [SLOT.plexGraphContext, 'plex'],
    [SLOT.fileTreeContext, 'plex'],
  ])('in %s show the note in the tab of kind %s', async (slot, kind) => {
    const one = setup()
    const commands = createWindowCommands({
      getAppIds: () => [],
      getPluginIds: () => [],
      runCommandToEnd: one.used.pluginRuns.runCommandToEnd,
      listItems: async () => [],
      runWithItem: async () => undefined,
      getTarget: () => ({
        ...REEF,
        kind: null,
        file: 'Reef.md',
        vault: { id: 'sea', name: 'Sea' },
        isReady: true,
      }),
    })
    const item = getRevealItems(one.pluginHost.slots.getEntries(slot, REEF)).find(
      (entry) => entry.values?.['kind'] === kind,
    )

    expect(item).toBeDefined()
    await commands.run(REVEAL, REEF, undefined, item?.values)

    expect(one.navigateTab).toHaveBeenCalledExactlyOnceWith(kind, 'Reef.md')
  })
})
