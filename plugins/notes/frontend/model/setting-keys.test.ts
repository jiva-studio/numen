import { describe, expect, it, vi } from 'vitest'
import type { JsonValue, PluginContext } from '@numen/plugin-sdk'
import { bindMarkdownEditorPreferences, readMarkdownEditorPreferences } from './preferences'

describe('the keys of the notes plugin settings', () => {
  it('reads the wrap and the line numbers from the camelCase keys', () => {
    const settings: Record<string, JsonValue> = { softWrap: false, lineNumbers: true }

    expect(readMarkdownEditorPreferences((key: string) => settings[key])).toMatchObject({
      softWrap: false,
      lineNumbers: true,
    })
  })

  it('ignores the snake_case keys of the same settings', () => {
    const settings: Record<string, JsonValue> = { soft_wrap: false, line_numbers: true }

    expect(readMarkdownEditorPreferences((key: string) => settings[key])).toMatchObject({
      softWrap: true,
      lineNumbers: false,
    })
  })

  it('follows a change of the camelCase keys and listens to no snake_case key', () => {
    const watchedKeys: string[] = []
    const context = {
      settings: {
        get: () => undefined,
        onChange: (key: string) => {
          watchedKeys.push(key)
          return { dispose: vi.fn() }
        },
      },
    } as unknown as Pick<PluginContext, 'settings'>

    bindMarkdownEditorPreferences(context)

    expect(watchedKeys).toContain('softWrap')
    expect(watchedKeys).toContain('lineNumbers')
    expect(watchedKeys).not.toContain('soft_wrap')
    expect(watchedKeys).not.toContain('line_numbers')
  })
})
