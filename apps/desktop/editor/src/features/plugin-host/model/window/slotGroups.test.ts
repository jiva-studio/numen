import { describe, expect, it } from 'vitest'
import { ref } from 'vue'
import { createSlotGroups } from './slotGroups'

const PLEX = 'org.example.plex'
const TAGS = 'org.example.tags'

describe('the slots a plugin draws', () => {
  const groups = createSlotGroups({
    slots: ref([
      { pluginId: TAGS, slot: 'plexGraph.nodeBelow', view: 'tags', order: 2 },
      { pluginId: TAGS, slot: 'plexGraph.context', command: 'tags.set', order: 1 },
      { pluginId: 'org.example.names', slot: 'plexGraph.nodeBelow', view: 'names', order: 1 },
    ]),
    commands: ref([{ pluginId: TAGS, id: 'tags.set', title: 'Set tags', arguments: [] }]),
    contributions: {
      slotPoints: ref([
        { pluginId: PLEX, slot: 'plexGraph.nodeBelow', title: 'Below a node' },
        { pluginId: PLEX, slot: 'plexGraph.context', title: 'Node menu' },
        { pluginId: 'org.example.other', slot: 'other.row', title: 'Row' },
      ]),
    },
  })

  it('list the entries of each slot of the plugin in the order the slot draws them', () => {
    expect(groups(PLEX)).toStrictEqual([
      {
        slot: 'plexGraph.nodeBelow',
        title: 'Below a node',
        contributors: [
          { key: 'org.example.names/view/names', label: 'names', pluginId: 'org.example.names' },
          { key: `${TAGS}/view/tags`, label: 'tags', pluginId: TAGS },
        ],
      },
      {
        slot: 'plexGraph.context',
        title: 'Node menu',
        contributors: [
          { key: `${TAGS}/command/${TAGS}:tags.set`, label: 'Set tags', pluginId: TAGS },
        ],
      },
    ])
  })

  it('are none for a plugin that draws no slot', () => {
    expect(groups('org.example.none')).toStrictEqual([])
  })
})
