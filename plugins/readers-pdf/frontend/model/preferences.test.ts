import { describe, expect, it } from 'vitest'
import type { JsonValue } from '@numen/plugin-sdk'
import {
  bindPDFReaderPreferences,
  readPDFReaderPreferences,
  readerPreferences,
} from './preferences'

describe('the PDF reader settings', () => {
  it('reads the default zoom the settings name', () => {
    const settings: Record<string, JsonValue> = { defaultZoom: 'width' }

    expect(readPDFReaderPreferences((key) => settings[key])).toStrictEqual({ defaultZoom: 'width' })
  })

  it('opens a whole page where the setting is missing or names no fit', () => {
    expect(readPDFReaderPreferences()).toStrictEqual({ defaultZoom: 'page' })
    expect(readPDFReaderPreferences(() => 'half')).toStrictEqual({ defaultZoom: 'page' })
  })

  it('follows a change of the default zoom', () => {
    const handlers = new Map<string, (value: unknown) => void>()
    bindPDFReaderPreferences({
      settings: {
        get: <T>() => '100%' as T,
        onChange: <T>(key: string, handler: (value: T) => void) => {
          handlers.set(key, handler as (value: unknown) => void)
          return { dispose: () => handlers.delete(key) }
        },
      },
    })
    expect(readerPreferences.defaultZoom).toBe('100%')

    handlers.get('defaultZoom')?.('width')
    expect(readerPreferences.defaultZoom).toBe('width')

    handlers.get('defaultZoom')?.(null)
    expect(readerPreferences.defaultZoom).toBe('page')
  })
})
