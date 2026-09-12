/** The recording and url tab kinds of a window. */
import { watch } from 'vue'
import { recordingKind, recordings, useTranscript, type MediaTabDeps } from '@/entities/media'
import { RECORDINGS } from '@/pages/media-recording'
import { URLS } from '@/pages/media-url'
import type { Source } from '@/shared/file'
import type { WindowKindsDeps } from './deps'

export type MediaKindsDeps = Pick<
  WindowKindsDeps,
  'puts' | 'held' | 'runs' | 'plays' | 'vaults' | 'window' | 'where' | 'carries'
>

export function createMediaKinds({
  puts,
  held,
  runs,
  plays,
  vaults,
  window,
  where,
  carries,
}: MediaKindsDeps) {
  const over = (source: Source): MediaTabDeps => ({
    runs: (id, path, called) =>
      carries(id, {
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
    puts,
    RECORDINGS,
  )

  const pointed = recordingKind(
    held.handle,
    (path) => useTranscript(recordings, path, { plays }),
    over('url'),
    puts,
    URLS,
  )

  watch(window.tasks, () => {
    recorded.updateTasks(window.tasks.value)
    pointed.updateTasks(window.tasks.value)
  })

  return { recorded, pointed }
}
