/**
 * A message writer a test hands over, which keeps what was written through it.
 *
 * Everything in the window that has something to say takes a `MessageWriter`,
 * so a test that wants to read what was written hands one of these over.
 */
import type { MessageKind, MessageWriter } from '@/shared/notices/messages'

export function writer() {
  const notified: { text: string; kind: MessageKind }[] = []
  const write: MessageWriter = (text, kind = 'report') => void notified.push({ text, kind })
  return {
    write,
    /** Everything written through it, each with the kind it was written as. */
    messages: notified,
    /** The same, as the words alone, in the order they were written. */
    get messagesText(): readonly string[] {
      return notified.map((one) => one.text)
    },
    /**
     * The last message given to it. A writer nothing has been written through
     * has none, which is not the same as having written nothing.
     */
    last: () => notified.at(-1)?.text,
  }
}
