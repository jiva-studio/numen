/**
 * The bytes the application serves at addresses of its own.
 *
 * A page of a document, the markup of a book and a picture it carries are what
 * a browser's own elements speak, so they are fetched and not asked for over
 * the schema. What comes back is text or it is a refusal, and a refusal here
 * carries the address it was refused at: one address is one file, and which
 * one it was is the whole of what a caller can say about it.
 */

/** What one address answered with, or what it refused with. */
export async function fetched(at: string): Promise<string> {
  const answer = await fetch(at)
  if (!answer.ok) throw new Error(`${at}: ${answer.status}`)
  return await answer.text()
}
