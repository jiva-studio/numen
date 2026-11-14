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

  it('reads the four editor settings from the camelCase keys', () => {
    const settings: Record<string, JsonValue> = {
      tabSize: 4,
      fontFamily: 'serif',
      lineHeight: 2,
      highlightActiveLine: true,
    }

    expect(readMarkdownEditorPreferences((key: string) => settings[key])).toMatchObject(settings)
  })

  it('ignores the snake_case keys of the four editor settings', () => {
    const settings: Record<string, JsonValue> = {
      tab_size: 4,
      font_family: 'serif',
      line_height: 2,
      highlight_active_line: true,
    }

    expect(readMarkdownEditorPreferences((key: string) => settings[key])).toMatchObject({
      tabSize: 2,
      fontFamily: '',
      lineHeight: 1.6,
      highlightActiveLine: false,
    })
  })

  it('listens to the camelCase keys of the four editor settings and to no snake_case key', () => {
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

    for (const key of ['tabSize', 'fontFamily', 'lineHeight', 'highlightActiveLine']) {
      expect(watchedKeys).toContain(key)
    }
    for (const key of ['tab_size', 'font_family', 'line_height', 'highlight_active_line']) {
      expect(watchedKeys).not.toContain(key)
    }
  })
})
