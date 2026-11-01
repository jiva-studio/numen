/**
 * The palettes of the preset themes plugin, which a picture of a theme draws the window with.
 */
import { readFileSync } from 'node:fs'
import { fileURLToPath } from 'node:url'

const PRESETS = new URL('../../plugins/themes-presets/themes/', import.meta.url)

/** The stylesheet of the palette called `name`; throws when the plugin holds none by that name. */
export const readPreset = (name) => readFileSync(new URL(`${name}.css`, PRESETS), 'utf8')

/** Whether the plugin holds a palette called `name`. */
export const hasPreset = (name) => {
  try {
    readPreset(name)
    return true
  } catch {
    return false
  }
}

/** The folder the palettes stand in. */
export const PRESETS_DIR = fileURLToPath(PRESETS)
