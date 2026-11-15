import type { PluginContext } from '@numen/plugin-sdk'
import { readLook } from '../model/look'
import { readNames, type Names } from '../model/names'
import type { PlexLook } from '../types'

/** The settings of a context that holds none, so every one is at its default. */
const NO_SETTINGS: Pick<PluginContext, 'settings'> = {
  settings: { get: () => undefined, onChange: () => ({ dispose: () => undefined }) },
}

/** The look at its defaults, for a plex no settings are read for. */
export const getDefaultLook = (): PlexLook => readLook(NO_SETTINGS)

/** The names settings at their defaults, for a plex no settings are read for. */
export const getDefaultNames = (): Names => readNames(NO_SETTINGS)
