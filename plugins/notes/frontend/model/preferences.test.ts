import { describe, expect, it, vi } from 'vitest'
import type { JsonValue, PluginContext } from '@numen/plugin-sdk'
import { bindMarkdownEditorPreferences, readMarkdownEditorPreferences } from './preferences'

const read = (settings: Record<string, JsonValue>) =>
  readMarkdownEditorPreferences((key: string) => settings[key])

describe('the editor preferences', () => {
  it('reads the defaults when nothing is stored', () => {
    expect(readMarkdownEditorPreferences()).toStrictEqual({
      softWrap: true,
      lineNumbers: false,
      tabSize: 2,
      fontFamily: '',
      lineHeight: 1.6,
      highlightActiveLine: false,
    })
  })

  it('reads the stored values of the four editor settings', () => {
    expect(
      read({
        tabSize: 4,
        fontFamily: 'serif',
        lineHeight: 1.8,
        highlightActiveLine: true,
      }),
    ).toMatchObject({
      tabSize: 4,
      fontFamily: 'serif',
      lineHeight: 1.8,
      highlightActiveLine: true,
    })
  })

  it('reads the bounds of the tab size and of the line height', () => {
    expect(read({ tabSize: 1 }).tabSize).toBe(1)
    expect(read({ tabSize: 8 }).tabSize).toBe(8)
    expect(read({ lineHeight: 1.2 }).lineHeight).toBe(1.2)
    expect(read({ lineHeight: 2.4 }).lineHeight).toBe(2.4)
  })

  it.each([0, 9, 2.5, '4', null, true])('reads 2 for a tab size of %j', (value) => {
    expect(read({ tabSize: value as JsonValue }).tabSize).toBe(2)
  })

  it.each([0.5, 5, 'tall', Number.NaN, Number.POSITIVE_INFINITY, null])(
    'reads 1.6 for a line height of %j',
    (value) => {
      expect(read({ lineHeight: value as JsonValue }).lineHeight).toBe(1.6)
    },
  )

  it('reads an empty font family for a value the font check refuses or of the wrong type', () => {
    expect(read({ fontFamily: 'serif; color: red' }).fontFamily).toBe('')
    expect(read({ fontFamily: 12 }).fontFamily).toBe('')
  })

  it('reads false for an active line of the wrong type', () => {
    expect(read({ highlightActiveLine: 1 }).highlightActiveLine).toBe(false)
  })

  it('listens to the four new keys and to no snake_case key', () => {
    const watched: string[] = []
    const context = {
      settings: {
        get: () => undefined,
        onChange: (key: string) => {
          watched.push(key)
          return { dispose: vi.fn() }
        },
      },
    } as unknown as Pick<PluginContext, 'settings'>

    bindMarkdownEditorPreferences(context)

    expect(watched).toEqual(
      expect.arrayContaining(['tabSize', 'fontFamily', 'lineHeight', 'highlightActiveLine']),
    )
    expect(watched.filter((key) => key.includes('_'))).toStrictEqual([])
  })

  it('resets a key to the default when a change brings a value of the wrong type or range', () => {
    const handlers = new Map<string, (value: unknown) => void>()
    const context = {
      settings: {
        get: () => undefined,
        onChange: (key: string, handler: (value: unknown) => void) => {
          handlers.set(key, handler)
          return { dispose: vi.fn() }
        },
      },
    } as unknown as Pick<PluginContext, 'settings'>

    const preferences = bindMarkdownEditorPreferences(context)
    handlers.get('tabSize')?.(4)
    handlers.get('lineHeight')?.(2)
    expect(preferences.tabSize).toBe(4)
    expect(preferences.lineHeight).toBe(2)

    handlers.get('tabSize')?.(9)
    handlers.get('lineHeight')?.('tall')
    handlers.get('fontFamily')?.('a;b')
    expect(preferences.tabSize).toBe(2)
    expect(preferences.lineHeight).toBe(1.6)
    expect(preferences.fontFamily).toBe('')
  })
})
