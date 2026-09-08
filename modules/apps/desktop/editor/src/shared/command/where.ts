/**
 * When a command is offered, and when it is not.
 *
 * A command is offered over what is in front of the person: a note, a vault, a
 * file of a kind, a file nothing has been made from yet. Each answer here is
 * one such question, and the table in `commands.ts` names which it asks.
 */
import type { ArtifactState, ArtifactStates, Source } from '../core'
import type { RunSupport } from './runs'
import type { CommandTarget } from './target'

/** A command over the note in front, which there has to be one of. */
export const onNote = (at: CommandTarget): boolean => at.ready && at.path !== ''

/** A command over the vault in front, which there has to be one of. */
export const onVault = (at: CommandTarget): boolean => at.vault.id !== ''

/**
 * A run over the file in front, which the vault has to hold that kind of and
 * this build has to be able to do.
 */
export const onSource =
  (run: string, source: Source) =>
  (at: CommandTarget, runs: RunSupport): boolean =>
    at.ready && at.file !== '' && at.source === source && runs.canRun(run)

/**
 * A run over the file in front that is offered on what has been made from it,
 * and not on its kind alone: a book already read is not offered to be read. A
 * file nothing has been asked about carries nothing, and is offered.
 */
export const onEvidence =
  (run: string, source: Source, made: (carries: ArtifactStates) => boolean) =>
  (at: CommandTarget, runs: RunSupport): boolean =>
    onSource(run, source)(at, runs) && (isEmpty(at.made) || made(at.made))

const isEmpty = (carries: ArtifactStates): boolean => Object.keys(carries).length === 0

/** An artifact a run over the file would begin. */
export const owed = (made: ArtifactState | undefined): boolean =>
  made === undefined || made === 'none' || made === 'stopped'

export const always = (): boolean => true

