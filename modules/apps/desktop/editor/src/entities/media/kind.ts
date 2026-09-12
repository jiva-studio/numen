/**
 * What one recording tab holds: the recording being played, its transcript, and
 * the run that writes one down.
 *
 * A transcript grows while a run goes, so the tab is told when work on the
 * recording it holds moves and asks for the words again.
 */
import { computed, type Component } from 'vue'
import type { TranscriptState } from './transcript'
import type { Source } from '@/shared/file'
import type { Span } from '@/shared/span'
import type { Task } from '@/shared/notices/task'
import type {
  FileOpeners,
  OpenTab,
  SourceReader,
  TabKind,
  WindowHandle,
} from '@/entities/tab/@x/media'
import { DELETE_TEXT, PROOFREAD, TRANSCRIBE } from './words'
import { fileOf } from '@/shared/paths'

/** What a recording tab asks of the window it is drawn in. */
export interface MediaTabDeps {
  /**
   * A command asked for over the recording the tab holds, carried out where the
   * commands are. `called` is what the tab calls the recording, which is what a
   * step asking for an answer names.
   */
  runs(id: string, path: string, called: string): void
  /**
   * Whether this build can do a run at all, which decides whether the tab
   * offers it. A window that says nothing offers every run.
   */
  canRun?(run: string): boolean
}

/** What one recording tab holds. */
export type MediaTabState = ReturnType<typeof useTranscriptTab>

/**
 * One recording, with what can be asked about its words: writing them down
 * where there are none, and putting them right or taking them away where there
 * are. None is offered while a run is going, or where this build cannot do it
 * at all.
 */
export function useTranscriptTab(read: TranscriptState, asks: MediaTabDeps) {
  /** Whether this build can do a run. A window that says nothing offers every run. */
  const canRun = (run: string): boolean => asks.canRun?.(run) ?? true

  const written = computed(() => read.times.value.length > 0)

  // The words of a url are fetched from the address, so nothing here writes
  // them down.
  const transcribable = computed(
    () => !read.points.value && !written.value && !read.working.value && canRun(TRANSCRIBE),
  )
  const proofreadable = computed(() => written.value && !read.working.value && canRun(PROOFREAD))
  const deletable = computed(() => written.value && !read.working.value && canRun(DELETE_TEXT))

  const called = fileOf(read.path)
  const transcribes = () => asks.runs(TRANSCRIBE, read.path, called)
  const proofreads = () => asks.runs(PROOFREAD, read.path, called)
  const deletes = () => asks.runs(DELETE_TEXT, read.path, called)

  return {
    ...read,
    called,
    transcribable,
    transcribes,
    proofreadable,
    proofreads,
    deletable,
    deletes,
  }
}

/**
 * What a tab holding words with the times they were said at stands under: the
 * kind the window keeps it by, the source a command over it is over, and how
 * the openers hand it the files it holds. A recording and a url hold the same
 * thing and are drawn the same, and each is its own tab.
 */
export interface Medium<K extends string = string> {
  readonly tab: K
  readonly source: Source
  readonly draws: Component
  hands(puts: FileOpeners, opens: SourceReader): void
}

/**
 * The recording tabs of a window, or its url tabs. A file is its own tab, so
 * the same one opened again is the tab it is already played in.
 */
export function recordingKind<K extends string>(
  handle: WindowHandle,
  opens: (path: string) => TranscriptState,
  asks: MediaTabDeps,
  puts: FileOpeners,
  as: Medium<K>,
) {
  const kind: TabKind<MediaTabState, K> = {
    kind: as.tab,
    opens: (path) => useTranscriptTab(opens(path), asks),
    called: (state) => state.called,
    draws: as.draws,
    identity: (path) => path,
    shuts: (state) => {
      state.close()
      return true
    },
    over: (state) => ({ file: state.path, source: as.source }),
    attends: (state) =>
      ({
        path: state.path,
        recording: {
          transcribedDurationMs: state.transcribedDuration.value,
          durationMs: state.duration.value,
        },
      }) as OpenTab<K>,
    getAttention: (state) =>
      ({
        path: state.path,
        recording: {
          transcribedDurationMs: state.transcribedDuration.value,
          durationMs: state.duration.value,
        },
      }) as OpenTab<K>,
  }

  // The person is taken to the moment the first of the spans asked for was
  // spoken at.
  const hears = async (path: string, spans: readonly Span[]) => {
    const id = await handle.opens(as.tab, path)
    void handle.holds<MediaTabState>(as.tab, id)?.reach(...spans)
  }
  as.hands(puts, (path, spans) => void hears(path, spans))

  /**
   * What the application is doing, as it last said. A tab whose recording is
   * named there is being transcribed, and asks for the words again.
   */
  const ticked = (tasks: readonly Task[]) => {
    for (const one of handle.each<MediaTabState>(as.tab)) {
      one.state.ticks(tasks.some((task) => task.about === one.state.path))
    }
  }

  /**
   * The transcript of a recording went. Every tab standing on it reads the
   * words again, and finds there are none.
   */
  const deleted = (path: string) => {
    for (const one of handle.each<MediaTabState>(as.tab)) {
      if (one.state.path === path) one.state.again()
    }
  }

  return { kind, ticked, deleted }
}
