/** What the window gives the tab kinds it opens. */
import type { CommandDeps, CommandTarget, runSupport } from '@/features/command-palette'
import type { FileOpeners, useWindowTabs } from '@/entities/tab'
import type { createMediaTypeProbe } from '@/entities/media'
import type { MessageLog } from '@/shared/notices/messages'
import type { NotePort } from '@/app/ports/notes'
import type { FilePort } from '@/app/ports/files'
import type { useNoteEditors } from '../useNoteEditors'
import type { useSettings } from '../useSettings'
import type { useVaults } from '../useVaults'
import type { useWindowDisplay } from '../useWindowDisplay'

export interface WindowKindsDeps {
  core: NotePort & FilePort
  log: MessageLog
  tabOpeners: FileOpeners
  held: ReturnType<typeof useWindowTabs>
  runs: ReturnType<typeof runSupport>
  plays: ReturnType<typeof createMediaTypeProbe>
  editing: ReturnType<typeof useNoteEditors>
  settings: ReturnType<typeof useSettings>
  vaults: ReturnType<typeof useVaults>
  window: ReturnType<typeof useWindowDisplay>
  getTarget: () => CommandTarget
  runCommand: (id: string, target: CommandTarget) => void
  commandDeps: () => CommandDeps
}
