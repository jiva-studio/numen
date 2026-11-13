/** The names of notes that left the graph in turn go with their pills, whatever follows. */
import { effectScope, nextTick, reactive, ref } from 'vue'
import { describe, expect, it } from 'vitest'
import { useNamesRooms } from './useNamesRooms'
import { DEFAULT_OPTIONS } from '../lib/arrange'
import type { PlexMetrics } from './sizing/measure'
import type { PlexGraph } from '../../types'

const metrics: PlexMetrics = {
  node: () => 100,
  label: () => 10,
  heading: () => 10,
  name: (title) => title.length * 6,
  nameSize: 10,
  labelType: { size: 10, width: () => 10 } as unknown as PlexMetrics['labelType'],
}

const graphOf = (...ids: string[]): PlexGraph => ({
  nodes: [
    { id: 'focus', title: 'Focus', relation: 'focus' },
    ...ids.map((id) => ({ id, title: id, relation: 'child' as const })),
  ],
  edges: [],
})

describe('the room of a note that left the graph', () => {
  it('stays through a second change of the graph', async () => {
    const props = reactive({
      graph: graphOf('a', 'b', 'c'),
      names: (id: string) => ({ above: [], below: [{ id: `n:${id}`, title: 'Heat' }] }),
      showNames: true,
    })
    const rooms = effectScope().run(() => useNamesRooms(props, ref(metrics), ref(DEFAULT_OPTIONS)))!

    props.graph = graphOf('b', 'c')
    await nextTick()
    props.graph = graphOf('c')
    await nextTick()

    expect(rooms.getRoomOf('a')?.below).toHaveLength(1)
    expect(rooms.getRoomOf('b')?.below).toHaveLength(1)
  })
})
