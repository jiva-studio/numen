/**
 * Background streaming subscriptions for the window.
 */
import type { Ref, ShallowRef } from 'vue'
import type { createFollower } from '@numen/ui'
import { watchEdits, type NoteEdit } from '@/entities/note'
import { watchFocus, watchTasks, type Task } from '@/entities/window'
import { watchVaultChanges, type VaultChange } from '@/entities/vault'
import type { PathRename } from '@/shared/lib/paths'
import type { TextRange } from '@/shared/lib/textRange'

/** What the window reads of the vault it asks about: whether its scan is done and what failed. */
export interface VaultScan {
  readonly scan: { readonly error?: string | undefined; readonly isReady?: boolean | undefined }
}

/** Takes up a stream the same way each time it ends, or fails to open. */
export type Follows = ReturnType<typeof createFollower>

/** What the stream of the changes to the vault reaches. */
export interface VaultStreamDeps {
  readonly follows: Follows
  readonly listening: AbortController
  readonly opening: Ref<string>
  readonly isVaultSwapped: () => Promise<boolean>
  readonly reload: () => void
  readonly onVaultChanged: (
    paths: readonly string[],
    renamed: readonly PathRename[],
    versionPaths: readonly string[],
  ) => Promise<void> | void
  readonly readInitialNote: () => Promise<string>
  readonly ask: () => Promise<VaultScan>
}

/** What the stream of the changes to the notes reaches. */
export interface EditStreamDeps {
  readonly follows: Follows
  readonly listening: AbortController
  readonly reportChange: (edit: NoteEdit) => void
}

/** What the stream of the requests to show a note reaches. */
export interface FocusStreamDeps {
  readonly follows: Follows
  readonly listening: AbortController
  readonly travelTo: (path: string) => Promise<void> | void
  readonly openFileAt: (path: string, spans: readonly TextRange[]) => void
  readonly activateTab: (id: string) => void
}

/** What the stream of the running tasks reaches. */
export interface TaskStreamDeps {
  readonly follows: Follows
  readonly listening: AbortController
  readonly tasks: ShallowRef<readonly Task[]>
  readonly ask: () => Promise<VaultScan>
}

/** The streams the window follows, each read until the window closes. */
export interface WindowStreams {
  readonly follow: () => Promise<void>
  readonly draw: () => Promise<void>
  readonly watch: () => Promise<void>
  readonly followTasks: () => Promise<void>
}

function isChangeEmpty(change: VaultChange): boolean {
  return (
    change.paths.length === 0 &&
    change.renamed.length === 0 &&
    !change.shouldReload &&
    change.versionPaths.length === 0
  )
}

async function reportVaultChange(change: VaultChange, deps: VaultStreamDeps): Promise<void> {
  if (change.shouldReload) await deps.onVaultChanged([], change.renamed, [])
  else await deps.onVaultChanged(change.paths, change.renamed, change.versionPaths)
}

async function handleVaultChange(change: VaultChange, deps: VaultStreamDeps): Promise<void> {
  if (isChangeEmpty(change)) return
  if (change.shouldReload && (await deps.isVaultSwapped())) {
    deps.reload()
    return
  }
  await reportVaultChange(change, deps)
  if (change.renamed.some((went) => went.from === deps.opening.value)) {
    await attempt(deps.readInitialNote)
  }
  await attempt(deps.ask)
}

async function attempt<T>(step: () => Promise<T>): Promise<boolean> {
  try {
    await step()
    return true
  } catch {
    return false
  }
}

async function handleFocusChange(
  request: { path?: string; ranges: readonly TextRange[]; tabId?: string },
  deps: Pick<FocusStreamDeps, 'travelTo' | 'openFileAt' | 'activateTab'>,
): Promise<void> {
  if (request.tabId) {
    deps.activateTab(request.tabId)
    return
  }
  if (!request.path) return
  const ranges = request.ranges
    .map((one) => ({ start: one.start, end: one.end }))
    .filter((one) => one.end > one.start)
  if (!ranges.length) {
    await deps.travelTo(request.path)
    return
  }
  deps.openFileAt(request.path, ranges)
}

/** Follows the changes to the vault. */
export function createVaultStream(deps: VaultStreamDeps): () => Promise<void> {
  return () =>
    deps.follows(
      () => watchVaultChanges(deps.listening.signal),
      (change) => handleVaultChange(change, deps),
    )
}

/** Follows the changes being made to notes. */
export function createEditStream(deps: EditStreamDeps): () => Promise<void> {
  return () =>
    deps.follows(
      () => watchEdits(deps.listening.signal),
      (response) => {
        // The stream opens with an empty message, which distinguishes an open
        // stream from one that never opened.
        if (response.path) deps.reportChange(response)
      },
    )
}

/** Follows the requests to show a note or a tab. */
export function createFocusStream(deps: FocusStreamDeps): () => Promise<void> {
  return () =>
    deps.follows(
      () => watchFocus(deps.listening.signal),
      (request) => handleFocusChange(request, deps),
    )
}

/** Follows the running tasks, and asks for the vault again once they have all ended. */
export function createTaskStream(deps: TaskStreamDeps): () => Promise<void> {
  const { tasks } = deps
  return () =>
    deps.follows(
      () => watchTasks(deps.listening.signal),
      async (list) => {
        const ran = tasks.value.length > 0
        tasks.value = list
        if (!ran || list.length > 0) return
        await attempt(deps.ask)
      },
    )
}
