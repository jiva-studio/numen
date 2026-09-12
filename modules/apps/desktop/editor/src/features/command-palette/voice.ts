/**
 * The one line the window says a command's answer in: where it is said, what
 * there is to say, and how several things become one line.
 */
import type { Artifact, ArtifactState } from '@/shared/artifacts'
import type { ErrorCode } from '@/shared/errors'
import type { VaultErrorCode } from '@/shared/vaults'
import type { MessageWriter } from '@/shared/notices/messages'

/** The one line the window says a command's answer in. */
export interface Voice {
  /**
   * What was done, or could not be, in words a person reads. One command's
   * word replaces the last, and nothing said clears it.
   */
  readonly says: MessageWriter
}

/** Everything carrying a command out says in the window's voice. */
export interface Words {
  /** What the vault reported as error, in words a person reads. */
  readonly errors: Record<ErrorCode, string>
  /** What the list of vaults reported as error, in words a person reads. */
  readonly vaultErrors: Record<VaultErrorCode, string>
  /** What the machine's own folder picker is titled. */
  readonly folder: string
  /** The links that reach nothing now, which nothing repairs. */
  readonly dangling: string
  /** The vault opens with no note at all. */
  readonly nowhere: string
  /** The note is waiting on the person, and its file stays where it is. */
  readonly unanswered: string
  /** The note holds prose nobody here has seen, so nothing was written. */
  readonly stale: string
  /** A name at the destination is taken, and the file stayed where it was. */
  readonly occupied: string
  /** This build cannot do the run at all, and stops offering it. */
  readonly unrunnable: string
  /** What an artifact of a file now stands at, in words a person reads. */
  readonly made: Record<Artifact, Record<ArtifactState, string>>
  /** What a run over the address a note points at came to. */
  readonly fetched: Record<ArtifactState, string>
}

/** What a command did to other notes, named once each under what it did. */
export const formatNames = (says: string, notes: readonly string[]): string => {
  const named = [...new Set(notes)]
  return named.length === 0 ? '' : `${says} ${named.join(', ')}`
}

/** Everything one command has to say, as the one line the window says it in. */
export const all = (...says: readonly string[]): string => says.filter((one) => one).join('. ')
