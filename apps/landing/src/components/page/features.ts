/** The shape of a feature row: its texts with their translation keys and its picture. */
import type { ImageMetadata } from 'astro'

/** A window screenshot shown in a window frame. */
type WindowPicture = { app: string; title: string; light: ImageMetadata; dark: ImageMetadata }

/**
 * One feature row. `key` is the prefix of its translation keys: `<key>_kicker`, `<key>_title`,
 * `<key>_text` and `<key>_p1` to `<key>_p3`. `plugin` is a screenshot path without the
 * `-light.webp` ending.
 */
export type Feature = {
  key: string
  kicker: string
  title: string
  text: string
  points: [string, string, string]
  plugin?: string
  window?: WindowPicture
  more?: { href: string; key: string; text: string }
}
