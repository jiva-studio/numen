import type { PluginContext } from '@numen/plugin-sdk'
import { readLook } from '../model/look'
import type { PlexLook } from '../types'

/** The settings of a context that holds none, so every one is at its default. */
const NO_SETTINGS: Pick<PluginContext, 'settings'> = {
  settings: { get: () => undefined, onChange: () => ({ dispose: () => undefined }) },
}

export const getDefaultLook = (): PlexLook => readLook(NO_SETTINGS)
