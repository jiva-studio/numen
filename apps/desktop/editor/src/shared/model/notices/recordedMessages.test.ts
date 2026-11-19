import { describe, expect, it, vi } from 'vitest'
import { createMessageLog } from './messages'
import { createRecordedMessageLog } from './recordedMessages'

describe('window message recording', () => {
  it('records changed warnings and errors without their text', () => {
    const writeLog = vi.fn(async () => undefined)
    const messages = createRecordedMessageLog(createMessageLog(), { writeLog })
    const write = messages.getWriter('source')

    write('private text', 'warning')
    write('private text', 'warning')
    write('', 'warning')
    write('another message', 'info')
    write('private error', 'error')

    expect(writeLog).toHaveBeenCalledTimes(2)
    expect(writeLog).toHaveBeenNthCalledWith(1, {
      level: 'warning',
      source: 'window',
      message: 'Window message',
      properties: { component: 'source' },
    })
    expect(writeLog).toHaveBeenNthCalledWith(2, {
      level: 'error',
      source: 'window',
      message: 'Window message',
      properties: { component: 'source' },
    })
    expect(messages.messages.value[0]?.text).toBe('private error')
  })
})
