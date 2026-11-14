/**
 * Whether the plex shows the names of the notes related to each related note, and how many
 * of them a note shows, as the plugin's own settings hold them.
 */
import type { Ref } from 'vue'
import type { PluginContext, Unvalidated } from '@numen/plugin-sdk'
import { followSetting, NO_SETTINGS } from './followSetting'

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
  typeof value === 'number' && Number.isFinite(value)
    ? Math.min(Math.max(Math.round(value), NAMES_PER_NOTE_MIN), NAMES_PER_NOTE_MAX)
    : NAMES_PER_NOTE

/** The two settings read from the context, and followed while the plugin is active. */
export function readNames(context: Pick<PluginContext, 'settings'>): Names {
  return {
    showNames: followSetting(context, SHOW_NAMES_KEY, parseShowNames),
    namesPerNote: followSetting(context, NAMES_PER_NOTE_KEY, parseNamesPerNote),
  }
}

/** The names settings at their defaults, for a plex no settings are read for. */
export const getDefaultNames = (): Names => readNames(NO_SETTINGS)
