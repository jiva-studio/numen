import { describe, expect, it, vi } from 'vitest'
import { createMessageLog, createRecordedMessageLog } from './messages'

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
      message: 'Window message from source',
      properties: { component: 'source' },
    })
    expect(writeLog).toHaveBeenNthCalledWith(2, {
      level: 'error',
      source: 'window',
      message: 'Window message from source',
      properties: { component: 'source' },
    })
    expect(messages.messages.value[0]?.text).toBe('private error')
  })

  it('names the command a message is about', () => {
    const writeLog = vi.fn(async () => undefined)
    const messages = createRecordedMessageLog(createMessageLog(), { writeLog })

    messages.getWriter('command')('private text', 'error', { command: 'app:note.create' })

    expect(writeLog).toHaveBeenCalledWith({
      level: 'error',
      source: 'window',
      message: 'Window message from command app:note.create',
      properties: { component: 'command', command: 'app:note.create' },
    })
  })
})
