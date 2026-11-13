/**
 * Whether the plex shows the names of the notes related to each related note, and how many
 * of them a note shows, as the plugin's own settings hold them.
 */
import { ref, type Ref } from 'vue'
import type { PluginContext, Unvalidated } from '@numen/plugin-sdk'

/** The keys the manifest declares the two settings under. */
export const SHOW_NAMES_KEY = 'showNames'
export const NAMES_PER_NOTE_KEY = 'namesPerNote'

/** The defaults the manifest declares, and the range of the count. */
export const SHOW_NAMES = true
export const NAMES_PER_NOTE = 6
export const NAMES_PER_NOTE_MIN = 1
export const NAMES_PER_NOTE_MAX = 12

/** The two settings, kept in sync as the user changes them. */
export interface Names {
  readonly showNames: Readonly<Ref<boolean>>
  readonly namesPerNote: Readonly<Ref<number>>
}

const parseShowNames = (value: Unvalidated): boolean =>
  typeof value === 'boolean' ? value : SHOW_NAMES

const parseNamesPerNote = (value: Unvalidated): number =>
  typeof value === 'number' &&
  Number.isInteger(value) &&
  value >= NAMES_PER_NOTE_MIN &&
  value <= NAMES_PER_NOTE_MAX
    ? value
    : NAMES_PER_NOTE

/** The two settings read from the context, and followed while the plugin is active. */
export function readNames(context: Pick<PluginContext, 'settings'>): Names {
  const showNames = ref(parseShowNames(context.settings.get(SHOW_NAMES_KEY)))
  const namesPerNote = ref(parseNamesPerNote(context.settings.get(NAMES_PER_NOTE_KEY)))
  context.settings.onChange(SHOW_NAMES_KEY, (value) => {
    showNames.value = parseShowNames(value)
  })
  context.settings.onChange(NAMES_PER_NOTE_KEY, (value) => {
    namesPerNote.value = parseNamesPerNote(value)
  })
  return { showNames, namesPerNote }
}

/** The names settings at their defaults, for a plex no settings are read for. */
export const getDefaultNames = (): Names =>
  readNames({ settings: { get: () => undefined, onChange: () => ({ dispose: () => undefined }) } })
