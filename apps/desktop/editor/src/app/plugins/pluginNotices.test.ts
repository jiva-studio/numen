import { describe, expect, it } from 'vitest'
import { createNoticeQueue } from '@numen/shell'
import { createPluginNotices } from './pluginNotices'

describe('the notices of one plugin', () => {
  it('are shown in the window list under the plugin, info where no severity is given', () => {
    const queue = createNoticeQueue()
    const notices = createPluginNotices(queue, 'md.numen.cards')

    notices.show('Saved.')
    notices.show('Could not save.', 'error')

    expect(queue.notices.value.map((one) => [one.text, one.severity, one.stay])).toEqual([
      ['Saved.', 'info', 'read'],
      ['Could not save.', 'error', 'kept'],
    ])
  })

  it('carry the work a plugin does, with how far it has got', () => {
    const queue = createNoticeQueue()
    createPluginNotices(queue, 'md.numen.cards').setTasks([
      { id: 'index', label: 'Reading the vault', about: 'Physics', done: 2, total: 5 },
      { id: 'sync', label: 'Syncing', error: 'The server is gone' },
    ])

    expect(queue.notices.value).toMatchObject([
      { text: 'Reading the vault', about: 'Physics', done: 2, total: 5, isWorking: true },
      { text: 'The server is gone', severity: 'error', isWorking: false },
    ])
  })
})
