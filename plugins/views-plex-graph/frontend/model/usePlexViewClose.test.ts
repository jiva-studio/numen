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

describe('a closed plex', () => {
  it('reads nothing when the depth changes', async () => {
    const depth = ref<1 | 2>(1)
    const calls: string[] = []
    const plex = usePlexView({ listRelatedNotes: async (path) => {
      calls.push(path)
      return answer(path)
    } }, depth)
    await plex.go('Root.md')
    plex.close()
    calls.length = 0

    depth.value = 2
    await nextTick()

    expect(calls).toEqual([])
  })

  it('reads again when the depth of an open plex changes', async () => {
    const depth = ref<1 | 2>(1)
    const calls: string[] = []
    const plex = usePlexView({ listRelatedNotes: async (path) => {
      calls.push(path)
      return answer(path)
    } }, depth)
    await plex.go('Root.md')
    calls.length = 0

    depth.value = 2
    await nextTick()

    expect(calls).toEqual(['Root.md'])
  })
})
