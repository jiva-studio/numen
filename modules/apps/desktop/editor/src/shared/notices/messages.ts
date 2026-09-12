/**
 * What the window has to say, and for how long it holds on to it.
 *
 * Every part of the window that answers a person writes here under a name of
 * its own, and the corner draws what stands. Nothing here knows how any of it
 * is drawn.
 */
import { shallowRef, type Ref } from 'vue'

/**
 * What kind of thing the window said.
 *
 * An error is what did not happen. A caution is worth reading and needs
 * nothing done at once. A report is what a command did. A state is so until
 * something else makes it not so.
 */
export type MessageKind = 'error' | 'caution' | 'report' | 'state'

/** One message the window holds. */
export interface WindowMessage {
  /** What this message is addressed by, which no two of them share. */
  readonly id: string
  /** The part of the window that wrote it, which holds one message at a time. */
  readonly name: string
  readonly kind: MessageKind
  readonly text: string
}

/** How one part of the window writes a message, replacing what it last wrote. */
export type MessageWriter = (text: string, kind?: MessageKind) => void

/** The messages the window holds, and what changes them. */
export interface MessageLog {
  readonly messages: Ref<readonly WindowMessage[]>
  /** A writer under a name of its own. */
  under(name: string): MessageWriter
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
export function messageLog(): MessageLog {
  const messages = shallowRef<readonly WindowMessage[]>([])
  let minted = 0

  const under =
    (name: string): MessageWriter =>
    (text, kind = 'report') => {
      // The same text written again is nothing new, and what stands keeps its
      // identity and its place.
      const stood = messages.value.find((one) => one.name === name)
      if (stood === undefined && text === '') return
      if (stood?.text === text && stood.kind === kind) return

      const rest = messages.value.filter((one) => one.name !== name)
      if (text === '') {
        messages.value = rest
        return
      }
      minted += 1
      messages.value = [...rest, { id: `${name}#${minted}`, name, kind, text }]
    }

  const dismiss = (id: string): void => {
    messages.value = messages.value.filter((one) => one.id !== id)
  }

  return { messages, under, dismiss }
}
