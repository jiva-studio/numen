/**
 * A message writer a test hands over, which keeps what was written through it.
 *
 * Everything in the window that has something to say takes a `MessageWriter`,
 * so a test that wants to read what was written hands one of these over.
 */
import type { Severity, MessageWriter } from '@/shared/notices/messages'

export function writer() {
  const notifiedCalls: { text: string; kind: Severity }[] = []
  const write: MessageWriter = (text, kind = 'info') => void notifiedCalls.push({ text, kind })
  return {
    write,
    /** Everything written through it, each with the kind it was written as. */
    messages: notifiedCalls,
    /** The same, as the words alone, in the order they were written. */
    get messagesText(): readonly string[] {
      return notifiedCalls.map((one) => one.text)
    },
    /**
     * The last message given to it. A writer nothing has been written through
     * has none, which is not the same as having written nothing.
     */
    last: () => notifiedCalls.at(-1)?.text,
  }
}
