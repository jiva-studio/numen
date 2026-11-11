/**
 * What every part of the window reads: the log of messages, the store of the settings, the frame
 * and the registry of the commands, made before the parts that read the vault.
 */
import { useWindowFrame } from '@numen/workbench'
import { createSettingsStore, SETTINGS_WORDS } from '@/entities/settings'
import { createWindowPluginStatus } from '@/features/plugin-commands'
import { createEmitter } from '@/shared/lib/emitter'
import { createCommandRegistry } from '@/shared/model/command'
import { createMessageLog } from '@/shared/model/notices/messages'

export function createWindowFoundation() {
  const log = createMessageLog()
  const rest = createSettingsStore(SETTINGS_WORDS, log.getWriter('configured'))
  return {
    log,
    rest,
    writeMessage: log.getWriter('command'),
    frame: useWindowFrame(),
    status: createWindowPluginStatus(rest),
    registry: createCommandRegistry(),
    /** The plugin whose settings page a part asks for, or none for the page of the settings. */
    settingsRequests: createEmitter<string | undefined>(),
  }
}

export type WindowFoundation = ReturnType<typeof createWindowFoundation>
