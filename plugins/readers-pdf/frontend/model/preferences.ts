/**
 * The PDF reader settings, read from the plugin settings and kept current while the plugin is active.
 */
import { reactive } from 'vue'
import type { PluginContext } from '@numen/plugin-sdk'
import type { Fit } from '../types'

export interface PDFReaderPreferences {
  defaultZoom: Fit
}

const DEFAULT_ZOOM = 'defaultZoom'

const FITS: ReadonlySet<unknown> = new Set<Fit>(['page', 'width', '100%'])

const isFit = (value: unknown): value is Fit => FITS.has(value)

export function readPDFReaderPreferences(
  getSetting?: (key: string) => unknown,
): PDFReaderPreferences {
  const defaultZoom = getSetting?.(DEFAULT_ZOOM)
  return { defaultZoom: isFit(defaultZoom) ? defaultZoom : 'page' }
}

/** The settings every document tab of the plugin reads. */
export const readerPreferences = reactive<PDFReaderPreferences>({ defaultZoom: 'page' })

/** Reads the settings into `readerPreferences`, and again whenever one of them changes. */
export function bindPDFReaderPreferences(
  context: Pick<Partial<PluginContext>, 'settings'>,
): PDFReaderPreferences {
  const getSetting = context.settings?.get.bind(context.settings)
  readerPreferences.defaultZoom = readPDFReaderPreferences(getSetting).defaultZoom
  context.settings?.onChange(DEFAULT_ZOOM, (value) => {
    readerPreferences.defaultZoom = isFit(value) ? value : 'page'
  })
  return readerPreferences
}
