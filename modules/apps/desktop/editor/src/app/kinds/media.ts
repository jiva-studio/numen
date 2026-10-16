/** The url tab kind of a window. */
import { watch } from 'vue'
import { createRecordingKind, recordings, useTranscript, type MediaTabDeps } from '@/entities/media'
import { URLS } from '@/pages/media-url'
import type { Source } from '@/entities/file'
import type { WindowKindsDeps } from './deps'

export type MediaKindsDeps = Pick<
  WindowKindsDeps,
  'tabOpeners' | 'held' | 'runs' | 'plays' | 'vaults' | 'window' | 'getTarget' | 'runCommand'
>

export function createMediaKinds({
  tabOpeners,
  held,
  runs,
  plays,
  vaults,
  window,
  getTarget,
  runCommand,
}: MediaKindsDeps) {
  const createTabDeps = (source: Source): MediaTabDeps => ({
    runCommand: (id, path, called) =>
      runCommand(id, {
        ...getTarget(),
        path: '',
        title: called,
        file: path,
        source,
        made: vaults.makes.value.get(path) ?? {},
      }),
    canRun: (run) => runs.canRun(run),
  })

  const pointed = createRecordingKind(
    held.handle,
    (path) => useTranscript(recordings, path, { plays }),
    {
      deps: createTabDeps('url'),
      tabOpeners,
      as: URLS,
    },
  )

  watch(window.tasks, () => {
    pointed.updateTasks(window.tasks.value)
  })

  return { pointed }
}
