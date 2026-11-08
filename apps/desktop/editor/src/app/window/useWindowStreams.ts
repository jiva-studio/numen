/**
 * Background streaming subscriptions for the window.
 */
import type { Ref, ShallowRef } from 'vue'
import { createFollower } from '@numen/ui'
import type { NotePort } from '@/app/ports/notes'
import type { VaultPort } from '@/app/ports/vault'
import type { NoteEdit } from '@/entities/note'
import type { Task } from '@/shared/notices/task'
import type { PathRename } from '@/shared/paths'
import type { TextRange } from '@/shared/textRange'

export interface WindowStreamsDeps {
  readonly core: Pick<VaultPort, 'watchVaultChanges' | 'watchFocus' | 'watchTasks'> &
    Pick<NotePort, 'watchEdits'>
  readonly listening: AbortController
  readonly isOpen: () => boolean
  readonly setLost: (response: string) => void
  readonly wait: (ms: number) => Promise<unknown>
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
  readonly ask: () => Promise<unknown>
  readonly reportChange: (edit: NoteEdit) => void
  readonly travelTo: (path: string) => Promise<void> | void
  readonly openFileAt: (path: string, spans: readonly TextRange[]) => void
  readonly activateTab: (id: string) => void
}

interface VaultChangeData {
  paths: readonly string[]
  shouldReload?: boolean
  renamed: readonly PathRename[]
  versionPaths?: readonly string[]
}

async function handleVaultChange(change: VaultChangeData, deps: WindowStreamsDeps): Promise<void> {
  const versionPaths = change.versionPaths ?? []
  const isEmpty = change.paths.length === 0 && change.renamed.length === 0
  if (isEmpty && !change.shouldReload && versionPaths.length === 0) return
  if (change.shouldReload && (await deps.isVaultSwapped())) {
    deps.reload()
    return
  }
  if (change.shouldReload) await deps.onVaultChanged([], change.renamed, [])
  else await deps.onVaultChanged(change.paths, change.renamed, versionPaths)
  if (change.renamed.some((went) => went.from === deps.opening.value)) {
    try {
      await deps.readInitialNote()
    } catch {
      // The next change retries.
    }
  }
  try {
    await deps.ask()
  } catch {
    // What a change means is already drawn; the counts come round with the next one.
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
  const { core, listening, isOpen, setLost, wait, tasks, reportChange } = deps

  /** Every stream is read the same way, and taken up again the same way. */
  const follows = createFollower({
    isOpen,
    setLost,
    wait,
  })

  const follow = () =>
    follows(
      () => core.watchVaultChanges(listening.signal),
      (change) => handleVaultChange(change, deps),
    )

  /** Follows the changes being made to notes. */
  const draw = () =>
    follows(
      () => core.watchEdits(listening.signal),
      (response) => {
        // The stream opens with an empty message, which distinguishes an open
        // stream from one that never opened.
        if (response.path) reportChange(response)
      },
    )

  const watch = () =>
    follows(
      () => core.watchFocus(listening.signal),
      (request) => handleFocusChange(request, deps),
    )

  const followTasks = () =>
    follows(
      () => core.watchTasks(listening.signal),
      async (list) => {
        const ran = tasks.value.length > 0
        tasks.value = list
        if (!ran || list.length > 0) return
        try {
          await deps.ask()
        } catch {
          // The next pass retries.
        }
      },
    )

  return {
    follow,
    draw,
    watch,
    followTasks,
  }
}
