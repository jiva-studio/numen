import { describe, expect, it } from 'vitest'
import { ref } from 'vue'
import { createNoticeQueue } from '@numen/workbench'
import { createMessageLog } from '@/shared/model/notices/messages'
import { useNoticeTasks } from './useNoticeTasks'

describe('useNoticeTasks', () => {
  it('converts window state, error/warning messages, and plugin notices into failed tasks', () => {
    const log = createMessageLog()
    const queue = createNoticeQueue()
    const extraNotices = ref<readonly Record<string, unknown>[]>([])
    const unwatched = ref('')
    const error = ref('')
    const windowLost = ref('')
    const appearanceLost = ref('')

    const tasks = useNoticeTasks({
      log,
      plugins: {
        ...queue,
        notices: {
          get value() {
            return [...queue.notices.value, ...extraNotices.value]
          },
        },
      } as never,
      window: { unwatched, error, lost: windowLost } as never,
      settings: { appearance: { state: { lost: appearanceLost } } } as never,
    })

    expect(tasks.value).toStrictEqual([])

    unwatched.value = 'notes/sub'
    error.value = 'scan failed'
    appearanceLost.value = 'theme stream lost'
    log.getWriter('a')('info msg', 'info')
    log.getWriter('b')('warn msg', 'warning')
    log.getWriter('c')('err msg', 'error')
    queue.show('p', 'info notice', 'info')
    queue.show('p', '', 'error')
    queue.show('p', 'plugin warn', 'warning')
    queue.show('p', 'plugin err', 'error')
    extraNotices.value = [
      { id: 'n-empty', pluginId: 'p', text: '', severity: 'error', about: 'notes/empty.md' },
      {
        id: 'n-custom',
        pluginId: 'p',
        text: 'about notice',
        severity: 'error',
        about: 'notes/a.md',
      },
    ]

    expect(tasks.value).toHaveLength(8)
    expect(tasks.value[0]).toStrictEqual({
      id: 'unwatched',
      label: expect.any(String),
      about: 'notes/sub',
      progress: { completedCount: 0, total: 0, counting: 'items' },
      error: expect.any(String),
      source: { isUserStarted: true, pluginId: '', pluginName: '' },
      isCancellable: false,
      kind: 'work',
    })
    expect(tasks.value[1]?.id).toBe('unread')
    expect(tasks.value[2]?.id).toBe('lost')
    expect(tasks.value[2]?.label).toBe('theme stream lost')
    expect(tasks.value[2]?.about).toBe('')
    expect(tasks.value[3]?.label).toBe('warn msg')
    expect(tasks.value[4]?.label).toBe('err msg')
    expect(tasks.value[5]).toMatchObject({ label: 'plugin warn', about: '' })
    expect(tasks.value[6]).toMatchObject({ label: 'plugin err', about: '' })
    expect(tasks.value[7]).toMatchObject({ label: 'about notice', about: 'notes/a.md' })

    windowLost.value = 'window stream lost'
    expect(tasks.value[2]?.label).toBe('window stream lost')
  })
})
