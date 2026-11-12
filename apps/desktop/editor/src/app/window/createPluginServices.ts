/**
 * The services a plugin is given by the window: the workspace, the notes and files of the vault,
 * and what the window owns: the buffers, the tasks, the commands, the vaults and the
 * window itself.
 */
import type { Ref } from 'vue'
import type { BackgroundTaskHost, WindowControls } from '@numen/plugin-loader'
import type { NoticeQueue } from '@numen/workbench'
import type { FileOpeners, WindowTabs } from '@/entities/tab'
import { createWindowBackgroundTasks } from '@/features/background-tasks'
import { searchCore } from '@/features/command-palette'
import { createBufferHost } from '@/features/file-conflict'
import {
  createVaultAccess,
  createWindowService,
  createWindowVaults,
  createWorkspaceService,
  type HostPort,
  type PluginNotes,
} from '@/features/plugin-host'
import type { CommandTarget } from '@/shared/model/command'
import type { MessageLog } from '@/shared/model/notices/messages'
import type { createVaultServices } from './createVaultServices'

type VaultServices = ReturnType<typeof createVaultServices>

/** What the workspace a plugin is given is made from. */
export interface PluginWorkspaceDeps {
  readonly log: MessageLog
  readonly windowTabs: WindowTabs
  readonly tabOpeners: FileOpeners
  readonly tabSync: Parameters<typeof createWorkspaceService>[0]['tabSync']
  readonly window: VaultServices['window']
  readonly hostPort: Pick<HostPort, 'getCommandIcon'>
}

/** The workspace of the window: its tabs, files, icons and messages. */
export function createPluginWorkspace(deps: PluginWorkspaceDeps) {
  return createWorkspaceService({
    log: deps.log,
    windowTabs: deps.windowTabs,
    tabOpeners: deps.tabOpeners,
    tabSync: deps.tabSync,
    window: deps.window,
    getPluginCommandIcon: deps.hostPort.getCommandIcon,
  })
}

/** What the notes, the links and the files a plugin is given are made from. */
export interface PluginAccessDeps {
  readonly making: VaultServices['editing']['making']
  readonly pluginNotes: PluginNotes
  readonly getTarget: () => CommandTarget
  readonly hostPort: Pick<HostPort, 'runInvocation'>
}

/** The notes, the links and the files a plugin is given. */
export function createPluginAccess(deps: PluginAccessDeps) {
  return createVaultAccess({
    search: searchCore,
    making: deps.making,
    pluginNotes: deps.pluginNotes,
    getTarget: deps.getTarget,
    runInvocation: deps.hostPort.runInvocation,
  })
}

/** What the background tasks of the plugins read of the window. */
export interface PluginBackgroundTasksDeps {
  readonly log: MessageLog
  readonly notices: NoticeQueue
  readonly window: Pick<VaultServices['window'], 'health' | 'vault' | 'tasks'>
  readonly appearance: { readonly state: { readonly lost: Readonly<Ref<string>> } }
  readonly openSettings: (pluginId?: string) => void
}

/** The tasks of the window and the failures of the plugins, as the background tasks a plugin lists. */
export function createPluginBackgroundTasks(deps: PluginBackgroundTasksDeps): BackgroundTaskHost {
  const { window } = deps
  return createWindowBackgroundTasks({
    log: deps.log,
    plugins: deps.notices,
    window: {
      lost: window.health.lost,
      unwatched: window.vault.unwatched,
      error: window.vault.error,
      tasks: window.tasks,
    },
    settings: { appearance: deps.appearance },
    writeMessage: deps.log.getWriter('command'),
    openSettings: deps.openSettings,
  })
}

/** What the services a plugin is given that the window owns are made from. */
export interface WindowOwnedServicesDeps {
  readonly editing: Pick<VaultServices['editing'], 'fileFlush' | 'notes'>
  readonly window: Pick<VaultServices['window'], 'tasks'>
  readonly vaults: Pick<VaultServices['vaults'], 'vaultList'>
  readonly hostPort: Pick<HostPort, 'commands' | 'openVault'>
  readonly backgroundTasks: BackgroundTaskHost
  readonly frame: WindowControls
  readonly log: MessageLog
}

/** The buffers, the tasks, the commands, the vaults and the window a plugin is given. */
export function createWindowOwnedServices(deps: WindowOwnedServicesDeps) {
  const { editing, window, vaults, hostPort } = deps
  return {
    buffers: createBufferHost({ fileFlush: editing.fileFlush }),
    noteBuffer: editing.notes,
    taskList: window.tasks,
    backgroundTasks: deps.backgroundTasks,
    commands: hostPort.commands,
    vaults: createWindowVaults({ vaultList: vaults.vaultList, openVault: hostPort.openVault }),
    window: createWindowService({ frame: deps.frame, writeMessage: deps.log.getWriter('command') }),
  }
}
