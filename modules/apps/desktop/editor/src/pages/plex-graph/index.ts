/** The plex tab: the graph of notes around one note, and the menu over a relation in it. */
export { default as PlexTab } from './ui/PlexTab.vue'
export { createPlexKind } from './kind'
export { ITEMS, NONE } from './lib/menu'
export { usePlexView } from './model/usePlexView'
export { WORDS } from './words'
export { isPlexPluginActive, setPlexPluginActive } from './model/pluginState'
export type { PlexTabState } from './types'
