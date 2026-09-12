/** The recording and url tab kinds of a window. */
import { watch } from 'vue'
import { recordingKind, recordings, useTranscript, type MediaTabDeps } from '@/entities/media'
import { RECORDINGS } from '@/pages/media-recording'
import { URLS } from '@/pages/media-url'
import type { Source } from '@/shared/file'
import type { WindowKindsDeps } from './deps'

export type MediaKindsDeps = Pick<
  WindowKindsDeps,
  'tabOpeners' | 'held' | 'runs' | 'plays' | 'vaults' | 'window' | 'where' | 'runCommand'
>

export function createMediaKinds({
  tabOpeners,
  held,
  runs,
  plays,
  vaults,
  window,
  where,
  runCommand,
}: MediaKindsDeps) {
  const over = (source: Source): MediaTabDeps => ({
    runs: (id, path, called) =>
      runCommand(id, {
        ...where(),
        path: '',
        title: called,
        file: path,
        source,
        made: vaults.makes.value.get(path) ?? {},
      }),
    canRun: (run) => runs.canRun(run),
  })

  const recorded = recordingKind(
    held.handle,
    (path) => useTranscript(recordings, path, { plays }),
    over('recording'),
    tabOpeners,
    RECORDINGS,
  )

  const pointed = recordingKind(
    held.handle,
    (path) => useTranscript(recordings, path, { plays }),
    over('url'),
    tabOpeners,
    URLS,
  )

  watch(window.tasks, () => {
    recorded.updateTasks(window.tasks.value)
    pointed.updateTasks(window.tasks.value)
  })

  return { recorded, pointed }
}
