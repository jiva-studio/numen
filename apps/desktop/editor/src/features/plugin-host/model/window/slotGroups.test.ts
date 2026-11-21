import { describe, expect, it } from 'vitest'
import { ref } from 'vue'
import { createSlotGroups } from './slotGroups'

const PLEX = 'org.example.plex'
const TAGS = 'org.example.tags'

describe('the slots a plugin draws', () => {
  const groups = createSlotGroups(
    {
      slots: ref([
        { pluginId: TAGS, slot: 'plexGraph.nodeBelow', view: 'tags', title: 'Tags', order: 2 },
        {
          pluginId: TAGS,
          slot: 'plexGraph.context',
          command: 'tags.set',
          group: '3-plex',
          order: 1,
        },
        {
          pluginId: 'app',
          slot: 'plexGraph.context',
          command: 'app:note.createChild',
          group: '1-note',
        },
        {
          pluginId: 'app',
          slot: 'plexGraph.context',
          command: 'app:note.createJump',
          group: '1-note',
          order: 5,
        },
        {
          pluginId: 'org.example.names',
          slot: 'plexGraph.nodeBelow',
          view: 'names',
          title: 'Names',
          order: 1,
        },
      ]),
      commands: ref([{ pluginId: TAGS, id: 'tags.set', title: 'Set tags', arguments: [] }]),
      contributions: {
        slotPoints: ref([
          { pluginId: PLEX, slot: 'plexGraph.nodeBelow', title: 'Below a node' },
          { pluginId: PLEX, slot: 'plexGraph.context', title: 'Node menu' },
          { pluginId: 'org.example.other', slot: 'other.row', title: 'Row' },
        ]),
      },
    },
    () => [{ id: 'app:note.createChild', text: 'Create child note' }],
  )

  it('list the entries of each slot of the plugin in the order the slot draws them', () => {
    expect(groups(PLEX)).toStrictEqual([
      {
        slot: 'plexGraph.nodeBelow',
        title: 'Below a node',
        contributors: [
          {
            key: 'org.example.names/view/names',
            label: 'Names',
            pluginId: 'org.example.names',
            group: '',
          },
          { key: `${TAGS}/view/tags`, label: 'Tags', pluginId: TAGS, group: '' },
        ],
      },
      {
        slot: 'plexGraph.context',
        title: 'Node menu',
        contributors: [
          {
            key: 'app/command/app:note.createJump',
            label: 'Note create jump',
            pluginId: 'app',
            group: '1-note',
          },
          {
            key: 'app/command/app:note.createChild',
            label: 'Create child note',
            pluginId: 'app',
            group: '1-note',
          },
          {
            key: `${TAGS}/command/${TAGS}:tags.set`,
            label: 'Set tags',
            pluginId: TAGS,
            group: '3-plex',
          },
        ],
      },
    ])
  })

  it('are none for a plugin that draws no slot', () => {
    expect(groups('org.example.none')).toStrictEqual([])
  })
})
