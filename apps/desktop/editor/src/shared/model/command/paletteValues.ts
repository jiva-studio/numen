/**
 * The when values of the palette. Its target is what is in front: the note in front, or else the file
 * in front or chosen as a result; there is none while the vault does not accept operations.
 */
import {
  getWhenValues,
  type JsonObject,
  type WhenSettings,
  type WhenValues,
} from '@numen/plugin-loader'

/** What the palette knows of what is in front, as its command target holds it. */
interface PaletteInFront {
  readonly tab: string
  readonly path: string
  readonly file: string
  readonly others?: readonly string[]
  readonly vault: { readonly id: string }
  readonly isReady: boolean
}

/** The type and properties of the note in front, where the palette has read them. */
interface PaletteNote {
  readonly type?: string
  readonly properties?: JsonObject
}

/** The when values of the palette over what is in front. */
export function getPaletteValues(
  target: PaletteInFront,
  contentTypes: Readonly<Record<string, string>>,
  config: WhenSettings = {},
  note: PaletteNote = {},
): WhenValues {
  const path = target.path || target.file
  const inFront =
    target.isReady && path !== ''
      ? { path, isFolder: false, others: target.others ?? [] }
      : undefined
  return getWhenValues({
    window: {
      'window.isVaultOpen': target.vault.id !== '',
      'window.isVaultReady': target.isReady,
      'window.isTabOpen': target.tab !== '',
    },
    target: inFront,
    note,
    contentTypes,
    config,
  })
}
