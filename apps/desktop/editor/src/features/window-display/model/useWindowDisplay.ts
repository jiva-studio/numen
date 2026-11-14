import { shallowRef } from 'vue'
import type { NoteApi, NoteEdit } from '@/entities/note'
import type { FocusApi, Task, TaskApi } from '@/entities/window'
import type { HealthApi, PluginHealthItem } from '@/entities/plugin'
import type { VaultStateApi } from '@/entities/vault'
import type { PathRename } from '@/shared/lib/paths'
import type { TextRange } from '@/shared/lib/textRange'
import { formatError, type Caught } from '@/shared/api/errors'
import { createFollower, runInBackground } from '@numen/ui'
import {
  createEditStream,
  createFocusStream,
  createHealthStream,
  createTaskStream,
  createVaultStream,
  type VaultScan,
  type WindowStreams,
} from './useWindowStreams'
import { useDisplayHealth, type DisplayHealth } from './useDisplayHealth'
import { useDisplayOpening } from './useDisplayOpening'
import { askVaultState, isVaultSwapped, useDisplayVault } from './useDisplayVault'

/**
 * The vault as the whole window reads it, and the notifications when the vault
 * changes.
 *
 * One window reads the vault once: one stream of changes, one stream of edits,
 * one stream of requested notes, one stream of running tasks, and one set of
 * counts. What is rendered from any of it, and by how many tabs, is the
 * window's concern; nothing here depends on the contents of a tab.
 */
/** The calls the window reads the vault and follows its streams through. */
export type DisplayApi = VaultStateApi &
  Pick<NoteApi, 'getInitialOpenPath' | 'watchEdits'> &
  FocusApi &
  Pick<TaskApi, 'watchTasks'> &
  Pick<HealthApi, 'watchPluginHealth'>

/** What the window passes for reading a vault, alongside the vault itself. */
interface DisplayOptions {
  wait?(this: void, ms: number): Promise<void>
  /**
   * Receives vault change notifications, and is awaited. A change carrying no
   * paths is empty: everything showing the vault reads again.
   */
  onVaultChanged?(
    this: void,
    paths: readonly string[],
    renamed?: readonly PathRename[],
    versionPaths?: readonly string[],
  ): void | Promise<void>
  /** Receives notifications of a change to a note while it is being created. */
  reportChange?(this: void, response: NoteEdit): void
  /** Shows a note to the user, requested from outside the window. */
  travelTo?(this: void, path: string): void | Promise<void>
  /**
   * Opens a document at spans of its own text, in the tab it is read in. The
   * first span is brought into view.
   */
  openFileAt?(this: void, path: string, spans: readonly TextRange[]): void
  /** Activates the open tab with the id, on a request from outside the window. */
  activateTab?(this: void, id: string): void
  /** What draws the page again, once another vault is under this window. */
  reload?(this: void): void
}

interface StartDisplayDeps {
  health: DisplayHealth
  ask: () => Promise<VaultScan>
  readInitialNote: () => Promise<string>
  onVaultChanged: (paths: readonly string[], renames: readonly PathRename[]) => Promise<void> | void
  streams: WindowStreams
  wait: (ms: number) => Promise<void>
  isOpen: () => boolean
}

function startStreams(streams: WindowStreams, health: DisplayHealth): void {
  const onError = (error: Caught) => {
    health.failure.value = formatError(error)
  }
  runInBackground(streams.follow(), onError)
  runInBackground(streams.watch(), onError)
  runInBackground(streams.draw(), onError)
  runInBackground(streams.followTasks(), onError)
  runInBackground(streams.followHealth(), onError)
}

async function startDisplay(deps: StartDisplayDeps): Promise<void> {
  const { health, ask, readInitialNote, onVaultChanged, streams, wait, isOpen } = deps
  try {
    while (isOpen()) {
      const vault = await ask()
      const note = await readInitialNote()
      if (!isOpen()) return
      if (note) {
        health.isIndexing.value = false
        await onVaultChanged([], [])
        startStreams(streams, health)
        return
      }
      if (vault.scan.error || vault.scan.isReady) {
        health.isIndexing.value = false
        startStreams(streams, health)
        return
      }
      await wait(100)
    }
  } catch (error) {
    health.failure.value = formatError(error)
    health.isIndexing.value = false
  }
}

export function useWindowDisplay(api: DisplayApi, how: DisplayOptions = {}) {
  const wait = how.wait ?? sleep
  // No listener is attached by default; each callback does nothing.
  const onVaultChanged = how.onVaultChanged ?? (() => undefined)
  const reload = how.reload ?? (() => undefined)
  const vault = useDisplayVault()
  const health = useDisplayHealth()
  const opening = useDisplayOpening(api.getInitialOpenPath)
  const tasks = shallowRef<readonly Task[]>([])
  const pluginHealth = shallowRef<readonly PluginHealthItem[]>([])
  let open = true
  const abortController = new AbortController()

  const ask = () => askVaultState(api.getVaultState, vault)

  const follows = createFollower({
    isOpen: () => open,
    setLost: (response) => {
      health.lost.value = response
    },
    wait,
  })
  const streams: WindowStreams = {
    follow: createVaultStream({
      watchVaultChanges: api.watchVaultChanges,
      follows,
      listening: abortController,
      opening: opening.path,
      isVaultSwapped: () => isVaultSwapped(api.getVaultState, vault),
      reload,
      onVaultChanged,
      readInitialNote: opening.read,
      ask,
    }),
    draw: createEditStream({
      watchEdits: api.watchEdits,
      follows,
      listening: abortController,
      reportChange: how.reportChange ?? (() => undefined),
    }),
    watch: createFocusStream({
      watchFocus: api.watchFocus,
      follows,
      listening: abortController,
      travelTo: how.travelTo ?? (() => undefined),
      openFileAt: how.openFileAt ?? (() => undefined),
      activateTab: how.activateTab ?? (() => undefined),
    }),
    followTasks: createTaskStream({
      watchTasks: api.watchTasks,
      follows,
      listening: abortController,
      tasks,
      ask,
    }),
    followHealth: createHealthStream({
      watchPluginHealth: api.watchPluginHealth,
      follows,
      listening: abortController,
      items: pluginHealth,
    }),
  }

  return {
    vault,
    health,
    opening,
    tasks,
    pluginHealth,
    start: () =>
      startDisplay({
        health,
        ask,
        readInitialNote: opening.read,
        onVaultChanged,
        streams,
        wait,
        isOpen: () => open,
      }),
    streams,
    close: () => {
      open = false
      abortController.abort()
    },
  }
}

export type WindowDisplay = ReturnType<typeof useWindowDisplay>

const sleep = (ms: number) => new Promise<void>((resolve) => setTimeout(resolve, ms))
