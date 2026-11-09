import { describe, expect, it } from 'vitest'
import { createPluginNotices, type NoticeList, type NoticeListTask } from './pluginNotices'

const createRecordingList = () => {
  const showCalls: [string, string, string][] = []
  const tasks: [string, readonly NoticeListTask[]][] = []
  const list: NoticeList = {
    show: (source, text, severity) => showCalls.push([source, text, severity]),
    setTasks: (source, given) => tasks.push([source, given]),
  }
  return { list, showCalls, tasks }
}

describe('the notices of one plugin', () => {
  it('are showCalls in the window list under the plugin, info where no severity is given', () => {
    const { list, showCalls } = createRecordingList()
    const notices = createPluginNotices(list, 'md.numen.cards')

    notices.show('Saved.')
    notices.show('Could not save.', 'error')

    expect(showCalls).toEqual([
      ['md.numen.cards', 'Saved.', 'info'],
      ['md.numen.cards', 'Could not save.', 'error'],
    ])
  })

  it('carry the work a plugin does, with how far it has got', () => {
    const { list, tasks } = createRecordingList()
    createPluginNotices(list, 'md.numen.cards').setTasks([
      { id: 'index', label: 'Reading the vault', about: 'Physics', done: 2, total: 5 },
      { id: 'sync', label: 'Syncing', error: 'The server is gone' },
    ])

    expect(tasks).toStrictEqual([
      [
        'md.numen.cards',
        [
          {
            id: 'index',
            label: 'Reading the vault',
            about: 'Physics',
            error: '',
            isUserStarted: false,
            done: 2,
            total: 5,
          },
          {
            id: 'sync',
            label: 'Syncing',
            about: '',
            error: 'The server is gone',
            isUserStarted: false,
          },
        ],
      ],
    ])
    expect('done' in (tasks[0]?.[1][1] ?? {})).toBe(false)
    expect('total' in (tasks[0]?.[1][1] ?? {})).toBe(false)
  })
})
