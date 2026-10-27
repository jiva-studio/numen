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
  ) => Promise<void> | void
  readonly readInitialNote: () => Promise<string>
  readonly ask: () => Promise<unknown>
  readonly reportChange: (edit: NoteEdit) => void
  readonly travelTo: (path: string) => Promise<void> | void
  readonly openFileAt: (path: string, spans: readonly TextRange[]) => void
}

interface VaultChangeData {
  paths: readonly string[]
  shouldReload?: boolean
  renamed: readonly PathRename[]
}

async function handleVaultChange(change: VaultChangeData, deps: WindowStreamsDeps): Promise<void> {
  if (change.paths.length === 0 && !change.shouldReload && change.renamed.length === 0) return
  if (change.shouldReload && (await deps.isVaultSwapped())) return void deps.reload()
  await deps.onVaultChanged(change.shouldReload ? [] : change.paths, change.renamed)
  if (change.renamed.some((went) => went.from === deps.opening.value)) {
    try {
      await deps.readInitialNote()
    } catch {
      // The next change asks again.
    }
  }
  try {
    await deps.ask()
  } catch {
    // What a change means is already drawn; the counts come round with the next one.
  }
}

async function handleFocusChange(
  request: { path?: string; ranges: readonly TextRange[] },
  travelTo: (path: string) => Promise<void> | void,
  openFileAt: (path: string, spans: readonly TextRange[]) => void,
): Promise<void> {
  if (!request.path) return
  const ranges = request.ranges
    .map((one) => ({ start: one.start, end: one.end }))
    .filter((one) => one.end > one.start)
  if (!ranges.length) return void (await travelTo(request.path))
  openFileAt(request.path, ranges)
}

export function createWindowStreams(deps: WindowStreamsDeps) {
  const { core, listening, isOpen, setLost, wait, tasks, reportChange, travelTo, openFileAt } = deps

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
        // The stream opens by saying nothing, which is how an open one is told
        // from one that never opened.
        if (response.path) reportChange(response)
      },
    )

  const watch = () =>
    follows(
      () => core.watchFocus(listening.signal),
      (request) => handleFocusChange(request, travelTo, openFileAt),
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
          // The pass after this one asks again.
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
