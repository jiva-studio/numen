/**
 * The window's commands over a row of the vault, asked without a screen.
 *
 * A run is offered on the rows the vault holds that kind of file at and on no
 * others, so the negatives are the ones worth having.
 */
import { describe, expect, it } from 'vitest'
import { groupItems } from '@numen/ui'
import type { ArtifactStates } from '@/entities/artifact'
import type { Source } from '@/entities/file'
import { createPathCommands, type RunGuard } from './pathMenu'

/** A window that has been told nothing, which can do every run. */
const canRunAnything: RunGuard = () => true

/** What a row of that kind offers, by the identity of each item. */
const getItemIds = (
  source: Source,
  isFolder = false,
  canRun: RunGuard = canRunAnything,
): readonly string[] => createPathCommands({ source, isFolder }, false, canRun).map((one) => one.id)

/** The same, as it is drawn: each item, and the rule standing above it. */
const getGroupedIds = (source: Source): readonly string[] =>
  groupItems(createPathCommands({ source, isFolder: false }, false, canRunAnything)).map(
    (one) => `${one.isRule ? '— ' : ''}${one.id}`,
  )

describe('the commands over a row standing for a recording', () => {
  it('offer the recording to be transcribed', () => {
    expect(getItemIds('recording')).toContain('transcribe')
  })

  it('offer nothing to recognise, which is asked of a scan', () => {
    expect(getItemIds('recording')).not.toContain('recognise')
  })
})

describe('the commands over a row standing for an address', () => {
  it('offer the runs over what is at the address', () => {
    expect(getItemIds('url')).toStrictEqual(
      expect.arrayContaining(['downloadText', 'downloadCopy', 'deleteText', 'deleteCopy']),
    )
  })

  // A build that cannot reach one offers none of them, which is how a person
  // is told this build does not do it.
  it('offer none of them where the build cannot do them', () => {
    const offered = getItemIds('url', false, () => false)
    for (const one of ['downloadText', 'downloadCopy', 'deleteText', 'deleteCopy']) {
      expect(offered).not.toContain(one)
    }
  })

  it('come back to the plain file commands where none of them can be run', () => {
    expect(getItemIds('url', false, () => false)).toStrictEqual(
      getItemIds('book', false, () => false),
    )
  })
})

describe('the commands over a row standing for a scanned document', () => {
  it('offer the text of it to be recognised', () => {
    expect(getItemIds('book')).toContain('recognise')
  })

  it('offer nothing to transcribe, which is asked of a recording', () => {
    expect(getItemIds('book')).not.toContain('transcribe')
  })
})

describe('the commands over a row standing for anything else', () => {
  it('offer neither run on a note', () => {
    expect(getItemIds('note')).not.toContain('transcribe')
    expect(getItemIds('note')).not.toContain('recognise')
  })

  it('offer neither run on a file the vault holds no source for', () => {
    expect(getItemIds('other')).not.toContain('transcribe')
    expect(getItemIds('other')).not.toContain('recognise')
  })

  // A folder of recordings is not a recording, and a run is over one file.
  it('offer neither run on a folder', () => {
    expect(getItemIds('recording', true)).not.toContain('transcribe')
    expect(getItemIds('book', true)).not.toContain('recognise')
  })

  it('offer only the removal over a selection of several', () => {
    expect(
      createPathCommands({ source: 'note', isFolder: false }, true, canRunAnything).map(
        (one) => one.id,
      ),
    ).toStrictEqual(['remove'])
  })
})

describe('the commands over a note', () => {
  it('are the palette’s, in the groups the menu draws them in', () => {
    expect(getGroupedIds('note')).toStrictEqual([
      'read',
      'travel',
      '— copy',
      '— child',
      'parent',
      'jump',
      'title',
      '— ask',
      '— remove',
    ])
  })
})

describe('where a run stands among the commands', () => {
  it('stands after the path and before the remove, parted from both by a rule', () => {
    expect(getGroupedIds('recording')).toStrictEqual([
      'copy',
      '— transcribe',
      'proofread',
      '— remove',
    ])
  })

  it('leaves no rule behind on a row that has no run', () => {
    expect(getGroupedIds('other')).toStrictEqual(['copy', '— remove'])
  })
})

describe('disabling of runs based on artifact states', () => {
  const getRuns = (source: 'recording' | 'book', made: ArtifactStates, format = 'pdf' as const) =>
    createPathCommands({ source, isFolder: false, format, made }, false, canRunAnything)

  it('disables transcribe and enables proofread when transcript is done', () => {
    const items = getRuns('recording', { transcript: 'done' })
    expect(items.find((one) => one.id === 'transcribe')?.disabled).toBe(true)
    expect(items.find((one) => one.id === 'proofread')?.disabled).toBe(false)
  })

  it('disables both when transcript and proofread are done', () => {
    const items = getRuns('recording', { transcript: 'done', 'transcript.corrected': 'done' })
    expect(items.find((one) => one.id === 'transcribe')?.disabled).toBe(true)
    expect(items.find((one) => one.id === 'proofread')?.disabled).toBe(true)
  })

  it('disables recognise and enables proofread when OCR is done on a PDF', () => {
    const items = getRuns('book', { ocr: 'done' })
    expect(items.find((one) => one.id === 'recognise')?.disabled).toBe(true)
    expect(items.find((one) => one.id === 'proofread')?.disabled).toBe(false)
  })

  it('disables both when OCR and reading proofread are done on a PDF', () => {
    const items = getRuns('book', { ocr: 'done', 'ocr.corrected': 'done' })
    expect(items.find((one) => one.id === 'recognise')?.disabled).toBe(true)
    expect(items.find((one) => one.id === 'proofread')?.disabled).toBe(true)
  })

  it('offers no scan runs on an EPUB book, which offers exactly what a file offers', () => {
    const epub = createPathCommands(
      { source: 'book', isFolder: false, format: 'epub' },
      false,
      canRunAnything,
    ).map((one) => one.id)
    expect(epub).toStrictEqual(getItemIds('other'))
  })
})

describe('the commands where this build cannot do a run at all', () => {
  it('offer the recording nothing, and leave the scan its own run', () => {
    const but: RunGuard = (run) => run !== 'transcribe' && run !== 'proofread'

    expect(getItemIds('recording', false, but)).not.toContain('transcribe')
    expect(getItemIds('recording', false, but)).not.toContain('proofread')
    expect(getItemIds('book', false, but)).toContain('recognise')
  })
})

describe('a run that is under way or finished', () => {
  const getDisabled = (
    source: 'recording' | 'book',
    made: ArtifactStates,
    id: string,
  ): boolean | undefined =>
    createPathCommands(
      { source, isFolder: false, format: 'pdf', made },
      false,
      canRunAnything,
    ).find((one) => one.id === id)?.disabled

  it('disables transcribe while the transcript is running or queued', () => {
    expect(getDisabled('recording', { transcript: 'running' }, 'transcribe')).toBe(true)
    expect(getDisabled('recording', { transcript: 'queued' }, 'transcribe')).toBe(true)
  })

  it('leaves transcribe enabled where the transcript failed or was never made', () => {
    expect(getDisabled('recording', { transcript: 'failed' }, 'transcribe')).toBe(false)
    expect(getDisabled('recording', {}, 'transcribe')).toBe(false)
  })

  it('disables recognise while the scan is running or queued', () => {
    expect(getDisabled('book', { ocr: 'running' }, 'recognise')).toBe(true)
    expect(getDisabled('book', { ocr: 'queued' }, 'recognise')).toBe(true)
  })

  it('leaves recognise enabled where the scan failed or was never made', () => {
    expect(getDisabled('book', { ocr: 'failed' }, 'recognise')).toBe(false)
    expect(getDisabled('book', {}, 'recognise')).toBe(false)
  })

  it('disables proofread while there is no transcript to proofread', () => {
    expect(getDisabled('recording', {}, 'proofread')).toBe(true)
    expect(getDisabled('recording', { transcript: 'running' }, 'proofread')).toBe(true)
  })

  it('disables proofread while the proofreading is running', () => {
    const made: ArtifactStates = { transcript: 'done', 'transcript.corrected': 'running' }
    expect(getDisabled('recording', made, 'proofread')).toBe(true)
  })

  it('leaves proofread enabled where the proofreading is queued or failed', () => {
    const queued: ArtifactStates = { transcript: 'done', 'transcript.corrected': 'queued' }
    const failed: ArtifactStates = { transcript: 'done', 'transcript.corrected': 'failed' }
    expect(getDisabled('recording', queued, 'proofread')).toBe(false)
    expect(getDisabled('recording', failed, 'proofread')).toBe(false)
  })

  it('disables proofread of a scan while the scan or the proofreading is running', () => {
    const made: ArtifactStates = { ocr: 'done', 'ocr.corrected': 'running' }
    expect(getDisabled('book', made, 'proofread')).toBe(true)
    expect(getDisabled('book', { ocr: 'running' }, 'proofread')).toBe(true)
  })
})
