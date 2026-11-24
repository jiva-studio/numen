import { shallowRef } from 'vue'
import { useOpeningNote, type NoteApi, type NoteEdit } from '@/entities/note'
import type { FocusApi, Task, TaskApi } from '@/entities/window'
import type { HealthApi, PluginHealthItem } from '@/entities/plugin'
import { isVaultSwapped, readShownVault, useShownVault, type VaultStateApi } from '@/entities/vault'
import type { PathRename } from '@/shared/lib/paths'
import type { TextRange } from '@/shared/lib/textRange'
import { formatError } from '@/shared/api/errors'
import { runInBackground } from '@numen/ui'
import { createStreamReader } from '@/shared/api/stream'
import {
  createEditStream,
  createFocusStream,
  createHealthStream,
  createTaskStream,
  createVaultStream,
  type StreamRunners,
  type VaultScan,
} from './streams'
import { useCoreHealth, type CoreHealth } from './useCoreHealth'

/**
 * The streams the whole window follows, and what they report.
 *
 * One window reads the vault once: one stream of changes, one stream of edits,
 * one stream of requested notes, one stream of running tasks, and one set of
 * counts. What is rendered from any of it, and by how many tabs, is the
 * window's concern; nothing here depends on the contents of a tab.
 */
/** The calls the window reads the vault and follows its streams through. */
export type WindowStreamsApi = VaultStateApi &
  Pick<NoteApi, 'getInitialOpenPath' | 'watchEdits'> &
  FocusApi &
  Pick<TaskApi, 'watchTasks'> &
  Pick<HealthApi, 'watchPluginHealth'>

/** What the window does on the events of the streams. */
interface StreamHandlers {
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

interface StartDeps {
  health: CoreHealth
  ask: () => Promise<VaultScan>
  readInitialNote: () => Promise<string>
  onVaultChanged: (paths: readonly string[], renames: readonly PathRename[]) => Promise<void> | void
  streams: StreamRunners
  wait: (ms: number) => Promise<void>
  isOpen: () => boolean
}

function startStreams(streams: StreamRunners, health: CoreHealth): void {
  const onError = (error: unknown) => {
    health.failure.value = formatError(error)
  }
  runInBackground(streams.follow(), onError)
  runInBackground(streams.watch(), onError)
  runInBackground(streams.draw(), onError)
  runInBackground(streams.followTasks(), onError)
  runInBackground(streams.followHealth(), onError)
}

async function startWhenReady(deps: StartDeps): Promise<void> {
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

function createStreamFollower(
  health: CoreHealth,
  wait: (ms: number) => Promise<void>,
  isOpen: () => boolean,
) {
  return createStreamReader({
    isOpen,
    setLost: (response) => {
      health.lost.value = response
    },
    wait,
  })
}

export function useWindowStreams(api: WindowStreamsApi, how: StreamHandlers = {}) {
  const wait = how.wait ?? sleep
  // No listener is attached by default; each callback does nothing.
  const onVaultChanged = how.onVaultChanged ?? (() => undefined)
  const reload = how.reload ?? (() => undefined)
  const vault = useShownVault()
  const health = useCoreHealth()
  const opening = useOpeningNote(api.getInitialOpenPath)
  const tasks = shallowRef<readonly Task[]>([])
  const pluginHealth = shallowRef<readonly PluginHealthItem[]>([])
  let open = true
  const abortController = new AbortController()

  const ask = () => readShownVault(api.getVaultState, vault)

  const follower = createStreamFollower(health, wait, () => open)
  const streams: StreamRunners = {
    follow: createVaultStream({
      watchVaultChanges: api.watchVaultChanges,
      follower: follower,
      listening: abortController,
      opening,
      isVaultSwapped: () => isVaultSwapped(api.getVaultState, vault),
      reload,
      onVaultChanged,
      ask,
    }),
    draw: createEditStream({
      watchEdits: api.watchEdits,
      follower: follower,
      listening: abortController,
      reportChange: how.reportChange ?? (() => undefined),
    }),
    watch: createFocusStream({
      watchFocus: api.watchFocus,
      follower: follower,
      listening: abortController,
      travelTo: how.travelTo ?? (() => undefined),
      openFileAt: how.openFileAt ?? (() => undefined),
      activateTab: how.activateTab ?? (() => undefined),
    }),
    followTasks: createTaskStream({
      watchTasks: api.watchTasks,
      follower: follower,
      listening: abortController,
      tasks,
      ask,
    }),
    followHealth: createHealthStream({
      watchPluginHealth: api.watchPluginHealth,
      follower: follower,
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
      startWhenReady({
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

const sleep = (ms: number) => new Promise<void>((resolve) => setTimeout(resolve, ms))
