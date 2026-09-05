/**
 * A voice a test speaks through, which keeps what it was told.
 *
 * Everything in the window that says something takes a `Voice`, so a test that
 * wants to read what was said hands one of these over.
 */
import type { MessageKind, Voice } from '../notices/telling'

export function voice() {
  const told: { text: string; kind: MessageKind }[] = []
  const says: Voice = (text, kind = 'report') => void told.push({ text, kind })
  return {
    says,
    /** Everything it was told, each with the voice it was said in. */
    told,
    /** The same, as the words alone, in the order it was told them. */
    get said(): readonly string[] {
      return told.map((one) => one.text)
    },
    /**
     * The last word it was given. A voice nothing has been said through has
     * none, which is not the same as having said nothing.
     */
    last: () => told.at(-1)?.text,
  }
}
