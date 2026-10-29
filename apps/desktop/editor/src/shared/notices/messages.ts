/**
 * The messages the window displays, and how long it keeps them.
 *
 * Each part of the window that reports to the user writes under its own name,
 * and the corner renders the current messages. Nothing here depends on how they
 * are rendered.
 */
import { shallowRef, type Ref } from 'vue'
import type { Severity, MessageWriter } from '@numen/plugin-sdk'

export type { Severity, MessageWriter }

/** One message the window holds. */
export interface WindowMessage {
  /** What this message is addressed by, which no two of them share. */
  readonly id: string
  /** The part of the window that wrote it, which holds one message at a time. */
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
