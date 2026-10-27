/** The entries of a file's menu, as the schema carries them and as the tab reads them. */
import { describe, expect, it } from 'vitest'
import { FILE_MENU_ACTION } from '@numen/plugin-sdk'
import { readMenuEntries, writeMenuAction } from './wire'

const entry = (action: number) => ({
  action,
  pluginId: 'md.numen.engines.transcription',
  versionId: 'md.numen.engines.transcription/abc',
  isLive: true,
})

describe('a menu entry', () => {
  it('crosses the wire and back as the action it is', () => {
    for (const action of [FILE_MENU_ACTION.CREATE_TEXT, FILE_MENU_ACTION.DELETE_VERSION]) {
      const [read] = readMenuEntries([entry(action)])
      expect(read && writeMenuAction(read.action)).toBe(action)
    }
  })

  it('is left out where the tab knows no such action', () => {
    expect(readMenuEntries([entry(FILE_MENU_ACTION.UNSPECIFIED)])).toStrictEqual([])
  })
})
