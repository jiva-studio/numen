/**
 * What the window has to say, and for how long it holds on to it.
 *
 * Every part of the window that answers a person writes here under a name of
 * its own, and the corner draws what stands. Nothing here knows how any of it
 * is drawn.
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
  /** A message the person is finished with, by the identity it was given. */
  dismiss(id: string): void
}

/**
 * Every message gets an identity of its own.
 *
 * What draws these remembers by identity when each arrived and which the person
 * put away, so a replacement under one name is a new message and is read as
 * one.
 */
export function createMessageLog(): MessageLog {
  const messages = shallowRef<readonly WindowMessage[]>([])
  let nextId = 0

  const getWriter =
    (name: string): MessageWriter =>
    (text, severity = 'info') => {
      // The same text written again is nothing new, and what stands keeps its
      // identity and its place.
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
