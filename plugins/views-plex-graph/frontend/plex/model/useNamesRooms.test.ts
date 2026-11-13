/** The room of the names of each node, and how long a node that left the graph keeps it. */
import { effectScope, nextTick, reactive, ref } from 'vue'
import { describe, expect, it } from 'vitest'
import { useNamesRooms } from './useNamesRooms'
import { DEFAULT_OPTIONS } from '../lib/arrange'
import type { PlexMetrics } from './sizing/measure'
import type { PlexGraph, PlexNames } from '../../types'

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

const getNamesOf = (id: string): PlexNames =>
  id === 'plain'
    ? { above: [], below: [] }
    : { above: [], below: [{ id: `n:${id}`, title: 'Heat' }] }

const setUp = (graph: PlexGraph) => {
  const props = reactive({ graph, names: getNamesOf, showNames: true })
  const scope = effectScope()
  const rooms = scope.run(() => useNamesRooms(props, ref(metrics), ref(DEFAULT_OPTIONS)))!
  return { props, rooms }
}

describe('the room of the names of a node', () => {
  it('is there for a node with names, and for none without, and none for the focus', () => {
    const { rooms } = setUp(graphOf('a', 'plain'))

    expect(rooms.getRoomOf('a')?.below).toHaveLength(1)
    expect(rooms.getRoomOf('plain')).toBeUndefined()
    expect(rooms.getRoomOf('focus')).toBeUndefined()
  })

  it('is given to the arrangement only while some node has names', () => {
    const { props, rooms } = setUp(graphOf('a'))
    expect(rooms.room.value).toBeDefined()

    props.showNames = false
    expect(rooms.room.value).toBeUndefined()
  })

  it('stays with a node that left the graph, so its names go with its pill', async () => {
    const { props, rooms } = setUp(graphOf('a', 'b'))

    props.graph = graphOf('b')
    await nextTick()

    expect(rooms.getRoomOf('a')?.below).toHaveLength(1)
    expect(rooms.getRoomOf('b')?.below).toHaveLength(1)
  })

  it('does not stay with a node that is still in the graph but has no names any more', async () => {
    const { props, rooms } = setUp(graphOf('a'))

    props.graph = graphOf('a')
    props.names = () => ({ above: [], below: [] })
    await nextTick()

    expect(rooms.getRoomOf('a')).toBeUndefined()
  })
})
