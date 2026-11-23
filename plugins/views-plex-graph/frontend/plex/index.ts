/** A graph of notes drawn as a plex, and the relations its edges run between. */
export { default as Plex } from './ui/Plex.vue'
export { getOptionsForType as optionsForType, useTypeSize } from './model/sizing/plexScale'
export { resolveOptions, type PlexOptionsInput } from './lib/arrange'
export { getInputStrategies, hasTouchPointer } from './model/behavior'
export { useDrawnWidgets } from './model/useDrawnWidgets'
