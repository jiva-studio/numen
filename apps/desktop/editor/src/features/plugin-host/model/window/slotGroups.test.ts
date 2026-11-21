import { describe, expect, it } from 'vitest'
import { ref } from 'vue'
import { createSlotGroups } from './slotGroups'

const PLEX = 'org.example.plex'
const TAGS = 'org.example.tags'

describe('the slots a plugin draws', () => {
  const groups = createSlotGroups({
    slots: ref([
      { pluginId: TAGS, slot: 'plexGraph.nodeBelow', view: 'tags', title: 'Tags', order: 2 },
      { pluginId: TAGS, slot: 'plexGraph.context', command: 'tags.set', group: '3-plex' },
      { pluginId: 'app', slot: 'plexGraph.context', command: 'app:note.createChild' },
      {
        pluginId: 'org.example.names',
        slot: 'plexGraph.nodeBelow',
        view: 'names',
        title: 'Names',
        order: 1,
      },
      { pluginId: 'org.example.bare', slot: 'plexGraph.nodeBelow', view: 'bareView' },
      { pluginId: PLEX, slot: 'titleBar.commands', command: 'open', group: 'a' },
      { pluginId: TAGS, slot: 'titleBar.commands', command: 'tags.set', title: 'Set tags' },
      { pluginId: PLEX, slot: 'titleBar.commands', command: 'app:note.create', group: 'b' },
    ]),
    commands: ref([{ pluginId: PLEX, id: 'open', title: 'Open graph', arguments: [] }]),
    contributions: {
      slotPoints: ref([
        { pluginId: PLEX, slot: 'plexGraph.nodeBelow', title: 'Below a node', menu: false },
        { pluginId: PLEX, slot: 'plexGraph.context', title: 'Node menu', menu: true },
        { pluginId: PLEX, slot: 'plexGraph.node', title: 'Over a node', menu: false },
        { pluginId: PLEX, slot: 'titleBar.commands', title: 'Buttons', menu: false },
        { pluginId: 'org.example.other', slot: 'other.row', title: 'Row', menu: false },
      ]),
    },
  })

  it('list the views of the slots of the plugin in the order the slot draws them', () => {
    expect(groups(PLEX).filter((one) => one.slot === 'plexGraph.nodeBelow')).toStrictEqual([
      {
        slot: 'plexGraph.nodeBelow',
        title: 'Below a node',
        contributors: [
          { key: 'org.example.names/view/names', label: 'Names', pluginId: 'org.example.names' },
          { key: `${TAGS}/view/tags`, label: 'Tags', pluginId: TAGS },
          {
            key: 'org.example.bare/view/bareView',
            label: 'Bare view',
            pluginId: 'org.example.bare',
          },
        ],
      },
    ])
  })

  it('list a slot of commands with the title of each command', () => {
    const buttons = groups(PLEX).find((one) => one.slot === 'titleBar.commands')
    expect(buttons?.contributors.map((one) => one.label)).toStrictEqual([
      'Open graph',
      'Note create',
      'Settings open',
      'Set tags',
    ])
  })

  it('leave out a menu slot and a slot with no entries', () => {
    expect(groups(PLEX).map((one) => one.slot)).not.toContain('plexGraph.context')
    expect(groups(PLEX).map((one) => one.slot)).not.toContain('plexGraph.node')
  })

  it('are none for a plugin that draws no slot', () => {
    expect(groups('org.example.none')).toStrictEqual([])
  })
})
