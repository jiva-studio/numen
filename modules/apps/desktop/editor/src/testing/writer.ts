/**
 * A message writer a test hands over, which keeps what was written through it.
 *
 * Everything in the window that has something to say takes a `MessageWriter`,
 * so a test that wants to read what was written hands one of these over.
 */
import type { MessageKind, MessageWriter } from '../notices/messages'

export function writer() {
  const told: { text: string; kind: MessageKind }[] = []
  const says: MessageWriter = (text, kind = 'report') => void told.push({ text, kind })
  return {
    says,
    /** Everything written through it, each with the kind it was written as. */
    told,
    /** The same, as the words alone, in the order they were written. */
    get said(): readonly string[] {
      return told.map((one) => one.text)
    },
    /**
     * The last message given to it. A writer nothing has been written through
     * has none, which is not the same as having written nothing.
     */
    last: () => told.at(-1)?.text,
  }
}
