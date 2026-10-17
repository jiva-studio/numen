/**
 * The runs over the file in front are offered on the kind of file it is, on
 * what this build can do, and on what has already been made from it.
 */
import { describe, expect, it } from 'vitest'
import { WORDS as words } from '@/shared/words'
import type { ArtifactStates } from '@/entities/artifact'
import type { Source } from '@/entities/file'
import type { Command, CommandTarget, RunSupport } from '../types'
import { createFileCommands } from './fileCommands'

const getCommand = (id: string): Command => {
  const found = createFileCommands(words).find((one) => one.id === id)
  if (!found) throw new Error(`no command ${id}`)
  return found
}

const at = (source: Source, made: ArtifactStates, isReady = true): CommandTarget => ({
  tab: 'tab-1',
  kind: 'file',
  path: 'talk.md',
  title: 'Talk',
  file: 'talk.md',
  source,
  made,
  vault: { id: 'v1', name: 'Zoo' },
  isReady,
})

/** A build that can do only the one run named. */
const only = (run: string): RunSupport => ({
  canRun: (asked) => asked === run,
  cannotRun: () => {},
})

/** A build that can do every run but the one named. */
const allBut = (run: string): RunSupport => ({
  canRun: (asked) => asked !== run,
  cannotRun: () => {},
})

const isOffered = (id: string, target: CommandTarget, runs: RunSupport = only(id)) =>
  getCommand(id).isOffered(target, runs)

describe('the runs over a file', () => {
  it('are all filed with the file commands', () => {
    const commands = createFileCommands(words)

    expect(commands.map((one) => one.group)).toStrictEqual(commands.map(() => 'file'))
  })
})

describe('a run over the file in front', () => {
  it('is not offered where the tab names no file', () => {
    expect(isOffered('recognise', { ...at('book', {}), file: '' })).toBe(false)
  })
})

describe('reading a book', () => {
  it('is offered on a build that can read one', () => {
    expect(isOffered('recognise', at('book', {}))).toBe(true)
    expect(isOffered('recognise', at('book', {}), allBut('recognise'))).toBe(false)
  })
})

describe('downloading the text of an address', () => {
  it('is offered on an address carrying a transcript or an article', () => {
    expect(isOffered('downloadText', at('url', { transcript: 'none' }))).toBe(true)
    expect(isOffered('downloadText', at('url', { article: 'done' }))).toBe(true)
  })

  it('is not offered on an address carrying neither', () => {
    expect(isOffered('downloadText', at('url', { copy: 'done' }))).toBe(false)
  })

  it('is offered on an address alone, and on a build that can download text', () => {
    expect(isOffered('downloadText', at('book', {}))).toBe(false)
    expect(isOffered('downloadText', at('url', {}), allBut('downloadText'))).toBe(false)
  })
})

describe('downloading a copy of a video', () => {
  it('is offered where the address carries a copy not yet made', () => {
    expect(isOffered('downloadCopy', at('url', { copy: 'none' }))).toBe(true)
  })

  it('is not offered while the vault cannot be asked', () => {
    expect(isOffered('downloadCopy', at('url', { copy: 'none' }, false))).toBe(false)
  })

  it('is not offered on a build that cannot download a copy', () => {
    expect(isOffered('downloadCopy', at('url', { copy: 'none' }), allBut('downloadCopy'))).toBe(
      false,
    )
  })

  it('is not offered where the address carries no copy, or one already made', () => {
    expect(isOffered('downloadCopy', at('url', {}))).toBe(false)
    expect(isOffered('downloadCopy', at('url', { copy: 'done' }))).toBe(false)
  })
})

describe('deleting a copy', () => {
  it('is asked about before it happens', () => {
    expect(getCommand('deleteCopy').needs).toBe('asking')
    expect(getCommand('deleteCopy').answers).toStrictEqual({
      keeps: words.keepsCopy,
      kept: words.kept,
      action: words.deletes,
      detail: words.deletedCopy,
    })
  })

  it('is offered on an address whose copy stands, on a build that can delete one', () => {
    expect(isOffered('deleteCopy', at('url', { copy: 'done' }))).toBe(true)
    expect(isOffered('deleteCopy', at('url', { copy: 'done' }), allBut('deleteCopy'))).toBe(false)
    expect(isOffered('deleteCopy', at('book', { copy: 'done' }))).toBe(false)
  })

  it('is not offered where no copy stands', () => {
    expect(isOffered('deleteCopy', at('url', { copy: 'none' }))).toBe(false)
  })
})

describe('deleting the text of an address', () => {
  it('is offered where the address carries a transcript', () => {
    expect(isOffered('deleteText', at('url', { transcript: 'done' }))).toBe(true)
  })

  it('is not offered where its transcript was never made', () => {
    expect(isOffered('deleteText', at('url', { transcript: 'none' }))).toBe(false)
  })
})
