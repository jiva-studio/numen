import { describe, expect, it, vi } from 'vitest'
import type { JsonValue } from '@numen/plugin-sdk'
import { bindPDFReaderPreferences, readPDFReaderPreferences } from './preferences'

describe('the keys of the PDF reader settings', () => {
  it('reads the default zoom from the camelCase key', () => {
    const settings: Record<string, JsonValue> = { defaultZoom: 'width' }

    expect(readPDFReaderPreferences((key) => settings[key])).toStrictEqual({ defaultZoom: 'width' })
  })

  it('ignores the snake_case key of the default zoom', () => {
    const settings: Record<string, JsonValue> = { default_zoom: 'width' }

    expect(readPDFReaderPreferences((key) => settings[key])).toStrictEqual({ defaultZoom: 'page' })
  })

  it('follows a change of the camelCase key and listens to no snake_case key', () => {
    const watchedKeys: string[] = []
    bindPDFReaderPreferences({
      settings: {
        get: <T>() => undefined as T,
        onChange: <T>(key: string, _handler: (value: T) => void) => {
          watchedKeys.push(key)
          return { dispose: vi.fn() }
        },
      },
    })

    expect(watchedKeys).toStrictEqual(['defaultZoom'])
  })
})
