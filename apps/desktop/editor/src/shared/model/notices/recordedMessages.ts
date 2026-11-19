import type { LogClient } from '@numen/core-client'
import type { MessageLog } from './messages'

/** Records warning and error messages from the window in the application log. */
export function createRecordedMessageLog(
  messages: MessageLog,
  logs: Pick<LogClient, 'writeLog'>,
): MessageLog {
  return {
    messages: messages.messages,
    dismiss: (id) => messages.dismiss(id),
    getWriter: (name) => {
      const write = messages.getWriter(name)
      return (text, severity = 'info') => {
        const previous = messages.messages.value
        write(text, severity)
        if (messages.messages.value === previous || text === '' || severity === 'info') return
        logs
          .writeLog({
            level: severity,
            source: 'window',
            message: 'Window message',
            properties: { component: name },
          })
          .catch(() => undefined)
      }
    },
  }
}
