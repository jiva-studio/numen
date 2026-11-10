import { ref } from 'vue'
import type { Core, NoteBaseline, Result } from '@/entities/core'

/** The note at `path`: the prose being edited, and the version of it the core last gave. */
export function useNote(core: Core, path: string) {
  const prose = ref('')
  const isReading = ref(true)
  let baseline: NoteBaseline | null = null

  /** Reads the note into `prose`. */
  async function read(): Promise<Result<null>> {
    try {
      const answer = await core.readNote(path)
      if (!answer.ok) return answer
      prose.value = answer.value.prose
      baseline = answer.value.stamp
        ? { prose: answer.value.prose, stamp: answer.value.stamp }
        : null
      return { ok: true, value: null }
    } finally {
      isReading.value = false
    }
  }

  /** Writes `prose` against the version `read` gave. */
  const keep = () => core.writeNote(path, prose.value, baseline)

  return { prose, isReading, read, keep }
}
