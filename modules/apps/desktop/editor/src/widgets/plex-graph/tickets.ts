/**
 * What the picture calls each note it draws.
 *
 * A ticket is minted the first time a note is drawn, and stays with that note
 * while its file moves. Everything the plex hands back names a node by its
 * ticket, and is translated to a path at the plex's edge.
 */
import { movedTo, type Move } from '../../shared/core'

export function createTickets() {
  /** The ticket each note holds, by the path its file is at. */
  let held = new Map<string, string>()
  /** What the last ticket was minted as. A number is spent once. */
  let minted = 0

  /** The ticket a note holds. A note drawn for the first time is minted one. */
  const of = (path: string): string => {
    if (!path) return ''
    const ticket = held.get(path) ?? String(++minted)
    held.set(path, ticket)
    return ticket
  }

  /** The note a ticket stands for, and nothing for a ticket nothing holds. */
  const note = (ticket: string): string => {
    if (!ticket) return ''
    for (const [path, holds] of held) if (holds === ticket) return path
    return ''
  }

  /** The tickets one picture drew. Every other note is let go of. */
  const keeps = (drawn: readonly string[]): void => {
    const drawing = new Set(drawn)
    for (const [path, ticket] of held) if (!drawing.has(ticket)) held.delete(path)
  }

  /**
   * Notes whose files went elsewhere. Each keeps the ticket it holds, and the
   * whole list is read against where the files were.
   */
  const moved = (renamed: readonly Move[]): void => {
    const went = new Map<string, string>()
    for (const [path, ticket] of held) went.set(movedTo(renamed, path) || path, ticket)
    held = went
  }

  return { of, note, keeps, moved }
}
