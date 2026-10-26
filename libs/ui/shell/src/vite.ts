/**
 * The build settings every desktop window shares.
 */
import { Features } from 'lightningcss'

/**
 * The Lightning CSS options of every stylesheet a window is built into. `light-dark()` stays as
 * written: the window chooses a half with `color-scheme`, which the lowered form does not read.
 */
export const LIGHTNING_CSS = { exclude: Features.LightDark } as const
