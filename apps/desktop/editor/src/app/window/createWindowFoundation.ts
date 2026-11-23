/**
 * What every part of the window reads: the calls to the core, the log of messages, the store of the
 * settings, the frame and the registry of the commands, made before the parts that read the vault.
 */
import { createWindowFrameControls } from '@numen/plugin-loader'
import { useWindowFrame } from '@numen/workbench'
import { createConnectionStore } from '@/entities/connection'
import { createSettingsStore, SETTINGS_WORDS } from '@/entities/settings'
import { createWindowPluginStatus } from '@/features/plugin-host'
import { createEmitter } from '@/shared/lib/emitter'
import { createCommandRegistry, type SettingsRequest } from '@/shared/model/command'
import { createMessageLog, createRecordedMessageLog } from '@/shared/model/notices/messages'
import type { WindowApi } from './createWindowApi'

export function createWindowFoundation(api: WindowApi) {
  const { configuration } = api
  const log = createRecordedMessageLog(createMessageLog(), api.window.logs)
  const rest = createSettingsStore(
    configuration.settings,
    SETTINGS_WORDS,
    log.getWriter('configured'),
  )
  return {
    api,
    log,
    rest,
    connections: createConnectionStore(configuration.connection, log.getWriter('connections')),
    frame: useWindowFrame(createWindowFrameControls(api.window.frameRuntime)),
    status: createWindowPluginStatus(rest),
    registry: createCommandRegistry(),
    /** The settings a part asks for: the page of a plugin and a setting of it, or the settings tab. */
    settingsRequests: createEmitter<SettingsRequest>(),
  }
}
