/**
 * A closed plex stops following its settings.
 */
import { describe, expect, it } from 'vitest'
import { nextTick, ref } from 'vue'
import { usePlexView } from './usePlexView'
import type { NoteGraph } from '../types'

const answer = (path: string): NoteGraph => ({
  focus: { path, title: path },
  focusType: 'note',
  related: [],
  connections: [],
})

const createReadingPlex = () => {
  const depth = ref<1 | 2>(1)
  const reads: string[] = []
  const plex = usePlexView(
    {
      listRelatedNotes: async (path) => {
        reads.push(path)
        return answer(path)
      },
    },
    depth,
  )
  return { depth, reads, plex }
}

describe('a plex', () => {
  it('reads the note again when the depth changes', async () => {
    const { depth, reads, plex } = createReadingPlex()
    await plex.go('Root.md')
    reads.length = 0

    depth.value = 2
    await nextTick()

    expect(reads).toEqual(['Root.md'])
  })

  it('reads nothing when the depth changes after it was closed', async () => {
    const { depth, reads, plex } = createReadingPlex()
    await plex.go('Root.md')
    plex.close()
    reads.length = 0

    depth.value = 2
    await nextTick()

    expect(reads).toEqual([])
  })
})
