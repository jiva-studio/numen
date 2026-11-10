import { shallowRef } from 'vue'
import type { NoteEdit } from '@/entities/note'
import type { Task } from '@/entities/window'
import type { PathRename } from '@/shared/paths'
import type { TextRange } from '@/shared/textRange'
import { formatError, type Caught } from '@/shared/api/errors'
import { runInBackground } from '@numen/ui'
import { createWindowStreams, type VaultScan } from './useWindowStreams'
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
/** What the window passes for reading a vault, alongside the vault itself. */
export interface DisplayOptions {
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
  streams: ReturnType<typeof createWindowStreams>
  wait: (ms: number) => Promise<void>
  isOpen: () => boolean
}

function startStreams(streams: ReturnType<typeof createWindowStreams>, health: DisplayHealth): void {
  const onError = (error: Caught) => {
    health.failure.value = formatError(error)
  }
  runInBackground(streams.follow(), onError)
  runInBackground(streams.watch(), onError)
  runInBackground(streams.draw(), onError)
  runInBackground(streams.followTasks(), onError)
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

export function useWindowDisplay(how: DisplayOptions = {}) {
  const wait = how.wait ?? sleep
  // No listener is attached by default; each callback does nothing.
  const onVaultChanged = how.onVaultChanged ?? (() => undefined)
  const reload = how.reload ?? (() => undefined)
  const vault = useDisplayVault()
  const health = useDisplayHealth()
  const opening = useDisplayOpening()
  const tasks = shallowRef<readonly Task[]>([])
  let open = true
  const abortController = new AbortController()

  const ask = () => askVaultState(vault)

  const streams = createWindowStreams({
    listening: abortController,
    isOpen: () => open,
    setLost: (response) => {
      health.lost.value = response
    },
    wait,
    opening: opening.path,
    tasks,
    isVaultSwapped: () => isVaultSwapped(vault),
    reload,
    onVaultChanged,
    readInitialNote: opening.read,
    ask,
    reportChange: how.reportChange ?? (() => undefined),
    travelTo: how.travelTo ?? (() => undefined),
    openFileAt: how.openFileAt ?? (() => undefined),
    activateTab: how.activateTab ?? (() => undefined),
  })

  return {
    vault,
    health,
    opening,
    tasks,
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
    follow: streams.follow,
    watch: streams.watch,
    close: () => {
      open = false
      abortController.abort()
    },
  }
}

export type WindowDisplay = ReturnType<typeof useWindowDisplay>

const sleep = (ms: number) => new Promise<void>((resolve) => setTimeout(resolve, ms))
