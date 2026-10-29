/**
 * A message writer for tests that records what was written through it.
 *
 * Every part of the window that reports messages takes a `MessageWriter`, so a
 * test that checks the messages passes one of these.
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
