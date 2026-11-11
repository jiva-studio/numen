/**
 * The kinds of tab and the commands of the window, and the few requests one makes of the other.
 */
import { getFileKinds } from '@/entities/file'
import { listDeclaredItemLists } from '@/entities/plugin'
import {
  createCommandNotes,
  createNoteLookup,
  runSupport,
  searchCore,
} from '@/features/command-palette'
import {
  createCommandItemLists,
  createCommandNoteReader,
  createFileMakers,
} from '@/features/plugin-commands'
import { createTargetResolver } from '@/features/window-display'
import type { TabsHold } from '@/features/vault-commands'
import { WORDS as words } from '@/shared/ui/words'
import { createCommands } from './createCommands'
import { collectFeatureCommands, createFeatureDeps } from './createFeatureCommands'
import type { WindowCore, WindowFoundation } from './windowCore'
import type { VaultServices, WindowServices } from './createWindowServices'
import type { createWindowTabMover } from '@/features/tab-move'

export function createTargetAndFeatureCommands(
  base: WindowFoundation,
  services: VaultServices,
  moves: ReturnType<typeof createWindowTabMover>,
  tabsHold: TabsHold,
) {
  const { editing, window, vaults } = services
  const { windowTabs, tabLinks } = base.tabs
  const itemLists = createCommandItemLists({
    listLists: listDeclaredItemLists,
    fileKinds: getFileKinds,
  })
  const getTarget = createTargetResolver(windowTabs, vaults, window)
  const featureCommands = collectFeatureCommands(
    createFeatureDeps({
      base,
      itemLists,
      services,
      moves,
      tabsHold,
      notes: createCommandNotes(editing.reached, windowTabs),
      tabLinks,
    }),
    words,
  )
  base.registry.registerAll(featureCommands.handlers)
  return { getTarget, featureCommands, itemLists }
}

export function createEditorCommands(
  base: WindowCore,
  services: WindowServices,
  initial: ReturnType<typeof createTargetAndFeatureCommands>,
) {
  const { editing, settings, vaults } = services
  const { windowTabs } = base.tabs
  const { getTarget, featureCommands } = initial
  const commands = createCommands({
    declaredCommands: base.plugins.declaredCommands,
    getAppCommands: () => featureCommands.commands,
    readCommandNote: createCommandNoteReader(base.plugins.itemLists),
    core: searchCore,
    words,
    windowTabs,
    getTarget,
    knows: createNoteLookup(editing, windowTabs),
    paletteLists: settings.paletteLists,
    runs: runSupport(),
    registry: base.registry,
    coverage: vaults.coverage,
    reached: editing.reached,
    settingPalette: settings.settingPalette,
    settings: settings.rest,
    writeMessage: base.writeMessage,
    pluginHost: base.plugins.host.pluginHost,
  })
  services.renames.subscribe((renamed) => commands.commands.navigation.applyRenames(renamed))
  return { getTarget, featureCommands, commands, facts: base.plugins.facts }
}

/** The tab kinds and file makers of the window. */
export function createWindowKinds(
  base: WindowCore,
  services: WindowServices,
) {
  const { settings } = services
  const { windowTabs, tabOpeners } = base.tabs
  windowTabs.groups.registerKinds(settings.kinds)
  return {
    findFileIcon: (path: string) => tabOpeners.findFileIcon(path),
    made: createFileMakers({ tabOpeners, windowTabs, writeMessage: base.log.getWriter('command') }),
    services: base.plugins.services,
  }
}
