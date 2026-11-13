/**
 * What every part of the window reads: the log of messages, the store of the settings, the frame
 * and the registry of the commands, made before the parts that read the vault.
 */
import { loadFrameRuntime, useWindowFrame } from '@numen/workbench'
import { createConnectionStore } from '@/entities/connection'
import { createSettingsStore, SETTINGS_WORDS } from '@/entities/settings'
import { createWindowPluginStatus } from '@/features/plugin-host'
import { createEmitter } from '@/shared/lib/emitter'
import { createCommandRegistry, type SettingsRequest } from '@/shared/model/command'
import { createMessageLog } from '@/shared/model/notices/messages'

export function createWindowFoundation() {
  const log = createMessageLog()
  const rest = createSettingsStore(SETTINGS_WORDS, log.getWriter('configured'))
  return {
    log,
    rest,
    connections: createConnectionStore(log.getWriter('connections')),
    writeMessage: log.getWriter('command'),
    frame: useWindowFrame(loadFrameRuntime),
    status: createWindowPluginStatus(rest),
    registry: createCommandRegistry(),
    /** The settings a part asks for: the page of a plugin and a setting of it, or the settings tab. */
    settingsRequests: createEmitter<SettingsRequest>(),
  }
}
