import type { VaultChangeEvent } from '@numen/plugin-sdk'
import type { fileOpeners, useWindowTabs } from '@/entities/tab'
import type { CommandTarget } from '@/features/command-palette'
import type { PathRename } from '@/shared/paths'
import type { Span } from '@/shared/span'
import type { useCommands } from '../commands/useCommands'
import type { createNoteEditors } from '../tabs/useNoteEditors'
import type { useVaults } from './useVaults'
import type { useWindowDisplay } from './useWindowDisplay'
import type { useWindowKinds } from './useWindowKinds'

export interface DisplayWiring {
  editing: ReturnType<typeof createNoteEditors>
  getCommands: () => ReturnType<typeof useCommands>
  getKinds: () => ReturnType<typeof useWindowKinds>
  tabOpeners: ReturnType<typeof fileOpeners>
  getVaults: () => ReturnType<typeof useVaults>
  notifyPlugins: (event: VaultChangeEvent) => void
}

export function createDisplayOptions(deps: DisplayWiring) {
  const { editing, getCommands, getKinds, tabOpeners, getVaults, notifyPlugins } = deps
  return {
    onVaultChanged: (paths: readonly string[], renamed: readonly PathRename[]) => {
      editing.notes.applyPathChanges(paths, renamed)
      getCommands().commands.applyRenames(renamed)
      notifyPlugins({ paths, renamed })
    },
    reportChange: editing.changes.reportChange,
    travelTo: (path: string) => getKinds().travel(path),
    openFileAt: (path: string, spans: readonly Span[]) => void tabOpeners.openFileAt(path, spans),
    reload: () => getVaults().reload(),
  }
}

export function createTargetResolver(
  windowTabs: ReturnType<typeof useWindowTabs>,
  getVaults: () => ReturnType<typeof useVaults>,
  window: ReturnType<typeof useWindowDisplay>,
) {
  const getFront = () => {
    const front = windowTabs.handle.front()
    const tab = front?.id ?? ''
    const on = front && windowTabs.getTab(tab)?.kind.getTarget?.(front.state)
    return { tab, kind: front?.kind ?? null, on }
  }
  const getMade = (file: string) => (file && getVaults().makes.value.get(file)) || {}

  return (): CommandTarget => {
    const { tab, kind, on } = getFront()
    const file = on?.file ?? ''
    return {
      tab,
      kind,
      path: on?.path ?? '',
      title: on?.title ?? '',
      file,
      source: on?.source ?? null,
      made: getMade(file),
      vault: getVaults().activeVault.value,
      isReady: !window.failure.value,
    }
  }
}
