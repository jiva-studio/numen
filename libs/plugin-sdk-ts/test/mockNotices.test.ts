import { describe, expect, it } from 'vitest'
import { createMockNoticeHost } from '../src/testing/mockNotices'

describe('the mock notice host', () => {
  it('keeps every notice in order, plain where no tone is given', () => {
    const notices = createMockNoticeHost()
    notices.show('Saved.')
    notices.show('Could not save.', 'alarm')

    expect(notices.getShown()).toStrictEqual([
      { text: 'Saved.', tone: 'plain' },
      { text: 'Could not save.', tone: 'alarm' },
    ])
  })

  it('keeps the list of work given last', () => {
    const notices = createMockNoticeHost()
    notices.setTasks([{ id: 'a', label: 'Reading' }])
    notices.setTasks([{ id: 'b', label: 'Writing' }])

    expect(notices.getTasks()).toStrictEqual([{ id: 'b', label: 'Writing' }])
  })
})
