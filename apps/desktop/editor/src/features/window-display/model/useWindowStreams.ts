/**
 * Background streaming subscriptions for the window.
 */
import type { Ref, ShallowRef } from 'vue'
import { createFollower } from '@numen/ui'
import { watchEdits, type NoteEdit } from '@/entities/note'
import { watchFocus, watchTasks, type Task } from '@/entities/window'
import { watchVaultChanges, type VaultChange } from '@/entities/vault'
import type { PathRename } from '@/shared/paths'
import type { TextRange } from '@/shared/textRange'

export interface WindowStreamsDeps {
  readonly listening: AbortController
  readonly isOpen: () => boolean
  readonly setLost: (response: string) => void
  readonly wait: (ms: number) => Promise<void>
  readonly opening: Ref<string>
  readonly tasks: ShallowRef<readonly Task[]>
  readonly isVaultSwapped: () => Promise<boolean>
  readonly reload: () => void
  readonly onVaultChanged: (
    paths: readonly string[],
    renamed: readonly PathRename[],
    versionPaths: readonly string[],
  ) => Promise<void> | void
  readonly readInitialNote: () => Promise<string>
  readonly ask: () => Promise<void>
  readonly reportChange: (edit: NoteEdit) => void
  readonly travelTo: (path: string) => Promise<void> | void
  readonly openFileAt: (path: string, spans: readonly TextRange[]) => void
  readonly activateTab: (id: string) => void
}

async function handleVaultChange(change: VaultChange, deps: WindowStreamsDeps): Promise<void> {
  const { versionPaths } = change
  const isEmpty = change.paths.length === 0 && change.renamed.length === 0
  if (isEmpty && !change.shouldReload && versionPaths.length === 0) return
  if (change.shouldReload && (await deps.isVaultSwapped())) {
    deps.reload()
    return
  }
  if (change.shouldReload) await deps.onVaultChanged([], change.renamed, [])
  else await deps.onVaultChanged(change.paths, change.renamed, versionPaths)
  if (change.renamed.some((went) => went.from === deps.opening.value)) {
    await attempt(deps.readInitialNote)
  }
  await attempt(deps.ask)
}

async function attempt(step: () => Promise<void>): Promise<boolean> {
  try {
    await step()
    return true
  } catch {
    return false
  }
}

async function handleFocusChange(
  request: { path?: string; ranges: readonly TextRange[]; tabId?: string },
  deps: Pick<WindowStreamsDeps, 'travelTo' | 'openFileAt' | 'activateTab'>,
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

export function createWindowStreams(deps: WindowStreamsDeps) {
  const { listening, isOpen, setLost, wait, tasks, reportChange } = deps

  /** Every stream is read the same way, and taken up again the same way. */
  const follows = createFollower({
    isOpen,
    setLost,
    wait,
  })

  const follow = () =>
    follows(
      () => watchVaultChanges(listening.signal),
      (change) => handleVaultChange(change, deps),
    )

  /** Follows the changes being made to notes. */
  const draw = () =>
    follows(
      () => watchEdits(listening.signal),
      (response) => {
        // The stream opens with an empty message, which distinguishes an open
        // stream from one that never opened.
        if (response.path) reportChange(response)
      },
    )

  const watch = () =>
    follows(
      () => watchFocus(listening.signal),
      (request) => handleFocusChange(request, deps),
    )

  const followTasks = () =>
    follows(
      () => watchTasks(listening.signal),
      async (list) => {
        const ran = tasks.value.length > 0
        tasks.value = list
        if (!ran || list.length > 0) return
        await attempt(deps.ask)
      },
    )

  return {
    follow,
    draw,
    watch,
    followTasks,
  }
}
