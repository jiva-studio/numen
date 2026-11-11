/** The target a run of a command is over. Every path is invented. */
import { describe, expect, it } from 'vitest'
import type { CommandTarget } from '@/shared/model/command'
import { getNamedTarget, toRunTarget } from './runTarget'

const FRONT: CommandTarget = {
  tab: 'note:1',
  kind: 'note',
  path: 'Front.md',
  title: 'Front',
  file: 'Front.md',
  vault: { id: 'v', name: 'V' },
  isReady: true,
}

describe('the target of a run', () => {
  it('is the tab in front where the run names no target', () => {
    expect(getNamedTarget(FRONT, undefined)).toStrictEqual({ ...FRONT, isTargetNamed: true })
  })

  it('is the target the run names, its file the path it names', () => {
    expect(
      getNamedTarget(FRONT, {
        path: 'books/Atlas.pdf',
        title: 'Atlas',
        isFolder: false,
        others: ['books/Reef.pdf'],
        tab: '',
      }),
    ).toStrictEqual({
      ...FRONT,
      path: 'books/Atlas.pdf',
      file: 'books/Atlas.pdf',
      title: 'Atlas',
      isFolder: false,
      others: ['books/Reef.pdf'],
      isTargetNamed: true,
    })
  })

  it('is over no file where the run names the root folder, not over the file in front', () => {
    const target = getNamedTarget(FRONT, {
      path: '',
      title: '',
      isFolder: true,
      others: [],
      tab: '',
    })

    expect([target.path, target.file]).toStrictEqual(['', ''])
  })

  it('reads back in the words of the SDK', () => {
    expect(toRunTarget(FRONT)).toStrictEqual({
      path: 'Front.md',
      title: 'Front',
      isFolder: false,
      others: [],
      tab: 'note:1',
    })
  })
})
