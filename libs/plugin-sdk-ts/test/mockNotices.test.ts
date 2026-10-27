import { describe, expect, it } from 'vitest'
import { createMockNoticeHost } from '../src/testing/mockNotices'

describe('the mock notice host', () => {
  it('keeps every notice in order, info where no severity is given', () => {
    const notices = createMockNoticeHost()
    notices.show('Saved.')
    notices.show('Could not save.', 'error')

    expect(notices.getShown()).toStrictEqual([
      { text: 'Saved.', severity: 'info' },
      { text: 'Could not save.', severity: 'error' },
    ])
  })

  it('keeps the list of work given last', () => {
    const notices = createMockNoticeHost()
    notices.setTasks([{ id: 'a', label: 'Reading' }])
    notices.setTasks([{ id: 'b', label: 'Writing' }])

    expect(notices.getTasks()).toStrictEqual([{ id: 'b', label: 'Writing' }])
  })
})
