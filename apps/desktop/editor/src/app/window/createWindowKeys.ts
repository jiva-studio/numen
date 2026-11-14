/**
 * The keys of the window: the shortcuts that open the palettes and run the commands.
 */
import type { WindowTabs } from '@/entities/tab'
import type { createWindowPalette } from '@/features/command-palette'
import { createAppKeyRunner, createKeyHandler, useAppHotkeys } from '@/features/hotkeys'
import type { createPluginCommandPort } from '@/features/plugin-commands'
import type { CommandTarget } from '@/shared/model/command'

/** What the keys of the window reach. */
export interface WindowKeysDeps {
  readonly windowTabs: Pick<WindowTabs, 'panes'>
  readonly getTarget: () => CommandTarget
  readonly palette: Pick<ReturnType<typeof createWindowPalette>, 'paletteSwitch' | 'runs'>
  readonly pluginCommands: ReturnType<typeof createPluginCommandPort>
}

export function createWindowKeys(deps: WindowKeysDeps): void {
  const { windowTabs, getTarget, palette, pluginCommands } = deps
  useAppHotkeys(
    createKeyHandler({
      windowTabs,
      getTarget,
      handleShortcut: pluginCommands.handleShortcut,
      runApp: createAppKeyRunner({
        toggleSearch: palette.paletteSwitch.toggleSearch,
        toggleCommands: palette.paletteSwitch.toggleCommands,
        runCommand: palette.runs.runCommand,
        runWithValues: palette.runs.runWithValues,
        getTarget,
      }),
    }),
  )
}
