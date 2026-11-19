/**
 * The messages the window displays, and how long it keeps them.
 *
 * Each part of the window that reports to the user writes under its own name,
 * and the status bar lists the failed ones as tasks. Nothing here depends on how they
 * are rendered.
 */
import { shallowRef, type Ref } from 'vue'
import type { WindowLogClient } from '@/shared/api/clients'
import type { Severity, MessageWriter } from '@numen/plugin-loader'

export type { MessageWriter }

/** One message the window holds. */
interface WindowMessage {
  readonly id: string
  readonly name: string
  readonly severity: Severity
  readonly text: string
}

/** The messages the window holds, and what changes them. */
export interface MessageLog {
  readonly messages: Ref<readonly WindowMessage[]>
  /** A writer under a name of its own. */
  getWriter(name: string): MessageWriter
  /** Dismisses a message by its identity. */
  dismiss(id: string): void
}

/**
 * Every message gets its own identity.
 *
 * The renderer tracks by identity when each message arrived and which ones the
 * user dismissed, so a replacement under the same name is a new message.
 */
export function createMessageLog(): MessageLog {
  const messages = shallowRef<readonly WindowMessage[]>([])
  let nextId = 0

  const getWriter =
    (name: string): MessageWriter =>
    (text, severity = 'info') => {
      // The same text written again changes nothing: the existing message keeps
      // its identity and position.
      const status = messages.value.find((one) => one.name === name)
      if (status === undefined && text === '') return
      if (status?.text === text && status.severity === severity) return

      const rest = messages.value.filter((one) => one.name !== name)
      if (text === '') {
        messages.value = rest
        return
      }
      nextId += 1
      messages.value = [...rest, { id: `${name}#${nextId}`, name, severity, text }]
    }

  const dismiss = (id: string): void => {
    messages.value = messages.value.filter((one) => one.id !== id)
  }

  return { messages, getWriter, dismiss }
}

/** Records warning and error messages from the window in the application log. */
export function createRecordedMessageLog(
  messages: MessageLog,
  logs: Pick<WindowLogClient, 'writeLog'>,
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
