/** The listeners of a note: the same functions every time, each reporting the id of its note. */
import { describe, expect, it, vi } from 'vitest'
import { useNodeHandlers, type NodeReports } from './useNodeHandlers'

const createReports = () =>
  ({
    activate: vi.fn(),
    show: vi.fn(),
    reach: vi.fn(),
    ask: vi.fn(),
    menu: vi.fn(),
    enter: vi.fn(),
    settle: vi.fn(),
    attend: vi.fn(),
  }) satisfies NodeReports

describe('the listeners of a note', () => {
  it('are the same functions each time they are asked for', () => {
    const getListeners = useNodeHandlers(createReports())

    expect(getListeners('a')).toBe(getListeners('a'))
    expect(getListeners('a')).not.toBe(getListeners('b'))
  })

  it('report the id of the note they belong to', () => {
    const reports = createReports()
    const getListeners = useNodeHandlers(reports)

    getListeners('a').activate()
    getListeners('b').enter('Setup')
    getListeners('b').attend(true)

    expect(reports.activate).toHaveBeenCalledWith('a')
    expect(reports.enter).toHaveBeenCalledWith('b', 'Setup')
    expect(reports.attend).toHaveBeenCalledWith('b', true)
  })
})
