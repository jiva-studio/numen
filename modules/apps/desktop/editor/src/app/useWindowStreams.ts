/**
 * Background streaming subscriptions for the window.
 */
import type { Ref, ShallowRef } from 'vue'
import { following } from '@numen/ui'
import type { NotePort } from '@/app/ports/notes'
import type { VaultPort } from '@/app/ports/vault'
import type { NoteEdit } from '@/entities/note'
import type { Task } from '@/shared/notices/task'
import type { PathRename } from '@/shared/paths'
import type { Span } from '@/shared/span'

export interface WindowStreamsDeps {
  readonly core: Pick<VaultPort, 'changes' | 'focus' | 'tasks'> & Pick<NotePort, 'editing'>
  readonly listening: AbortController
  readonly isOpen: () => boolean
  readonly setLost: (said: string) => void
  readonly wait: (ms: number) => Promise<unknown>
  readonly opening: Ref<string>
  readonly tasks: ShallowRef<readonly Task[]>
  readonly isVaultSwapped: () => Promise<boolean>
  readonly reloads: () => void
  readonly told: (paths: readonly string[], renamed: readonly PathRename[]) => Promise<void> | void
  readonly first: () => Promise<string>
  readonly ask: () => Promise<unknown>
  readonly drawing: (edit: NoteEdit) => void
  readonly wanted: (path: string) => Promise<void> | void
  readonly reads: (path: string, spans: readonly Span[]) => void
}

export function useWindowStreams(deps: WindowStreamsDeps) {
  const {
    core,
    listening,
    isOpen,
    setLost,
    wait,
    opening,
    tasks,
    isVaultSwapped,
    reloads,
    told,
    first,
    ask,
    drawing,
    wanted,
    reads,
  } = deps

  /** Every stream is read the same way, and taken up again the same way. */
  const follows = following({
    open: isOpen,
    lost: setLost,
    wait,
  })

  /**
   * Follow the vault.
   *
   * Every change is told, including changes to notes that are nowhere on
   * screen: a link is written at one end and shows at both, so a note edited
   * somewhere else is exactly how a new parent arrives. What is drawn cannot
   * answer whether a change reaches it.
   */
  const follow = () =>
    follows(
      () => core.changes(listening.signal),
      async (change) => {
        if (change.paths.length === 0 && !change.shouldReload && change.renamed.length === 0) return
        // A reload standing at another folder is another vault under this
        // window, and the page is drawn again on it.
        if (change.shouldReload && (await isVaultSwapped())) return void reloads()
        await told(change.shouldReload ? [] : change.paths, change.renamed)
        // The note the vault opens with is asked for again when it moves.
        if (change.renamed.some((went) => went.from === opening.value)) {
          try {
            await first()
          } catch {
            // The next change asks again.
          }
        }
        try {
          await ask()
        } catch {
          // What a change means is already drawn; the counts come round with
          // the next one.
        }
      },
    )

  /** Follows the changes being made to notes. */
  const draw = () =>
    follows(
      () => core.editing(listening.signal),
      (said) => {
        // The stream opens by saying nothing, which is how an open one is told
        // from one that never opened.
        if (said.path) drawing(said)
      },
    )

  /**
   * Travels to whatever is asked for while the window is open — an agent
   * working the vault beside the person naming the note it is talking about. A
   * focus naming a span of a source's text opens that source at it.
   */
  const watch = () =>
    follows(
      () => core.focus(listening.signal),
      async (asked) => {
        if (!asked.path) return
        const spans = asked.spans
          .map((one) => ({ from: one.from, to: one.to }))
          .filter((one) => one.to > one.from)
        if (!spans.length) return void (await wanted(asked.path))
        reads(asked.path, spans)
      },
    )

  /**
   * Keeps the list of what is being done up to date while the window is open.
   *
   * Nothing is asked for on a timer. Work begins without the window: an agent
   * is told to read a document, and the list says so the moment it starts.
   */
  const attend = () =>
    follows(
      () => core.tasks(listening.signal),
      async (list) => {
        const ran = tasks.value.length > 0
        tasks.value = list
        // What the vault holds moves while a pass runs and settles when it
        // ends, so it is asked for again the moment the list empties.
        if (!ran || list.length > 0) return
        try {
          await ask()
        } catch {
          // The pass after this one asks again.
        }
      },
    )

  return {
    follow,
    draw,
    watch,
    attend,
  }
}
