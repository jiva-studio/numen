/**
 * What the menu on a row of the tree offers, asked without a screen.
 *
 * A run is offered on the rows the vault holds that kind of file at and on no
 * others, so the negatives are the ones worth having.
 */
import { describe, expect, it } from 'vitest'
import { groupItems } from '@numen/ui'
import type { ArtifactStates } from '@/entities/artifact'
import type { Source } from '@/entities/file'
import {
  getMenuItems,
  type RunGuard,
  NEW_DECK,
  NEW_FOLDER,
  NEW_NOTE,
  NEW_PRESET,
  NEW_STENCIL,
  OFFERED,
} from './menu'

/** A window that has been told nothing, which can do every run. */
const canRunAnything: RunGuard = () => true

/** What a row of that kind offers, by the identity of each item. */
const getItemIds = (
  source: Source,
  isFolder = false,
  canRun: RunGuard = canRunAnything,
): readonly string[] => getMenuItems({ source, isFolder }, false, canRun).map((one) => one.id)

/** The same, as it is drawn: each item, and the rule standing above it. */
const getGroupedIds = (source: Source): readonly string[] =>
  groupItems(getMenuItems({ source, isFolder: false }, false, canRunAnything)).map(
    (one) => `${one.isRule ? '— ' : ''}${one.id}`,
  )

describe('the menu on a row standing for a recording', () => {
  it('offers the recording to be transcribed', () => {
    expect(getItemIds('recording')).toContain('transcribe')
  })

  it('offers nothing to recognise, which is asked of a scan', () => {
    expect(getItemIds('recording')).not.toContain('recognise')
  })
})

describe('the menu on a row standing for an address', () => {
  it('offers the runs over what is at the address', () => {
    expect(getItemIds('url')).toStrictEqual(
      expect.arrayContaining(['downloadText', 'downloadCopy', 'deleteText', 'deleteCopy']),
    )
  })

  // A build that cannot reach one offers none of them, which is how a person
  // is told this build does not do it.
  it('offers none of them where the build cannot do them', () => {
    const offered = getItemIds('url', false, () => false)
    for (const one of ['downloadText', 'downloadCopy', 'deleteText', 'deleteCopy']) {
      expect(offered).not.toContain(one)
    }
  })

  // What is left is the menu any file gets, which is where a scan lands too
  // when nothing can be run on it.
  it('comes back to the plain file menu where none of them can be run', () => {
    expect(getItemIds('url', false, () => false)).toStrictEqual(
      getItemIds('book', false, () => false),
    )
  })
})

describe('the menu on a row standing for a scanned document', () => {
  it('offers the text of it to be recognised', () => {
    expect(getItemIds('book')).toContain('recognise')
  })

  it('offers nothing to transcribe, which is asked of a recording', () => {
    expect(getItemIds('book')).not.toContain('transcribe')
  })
})

describe('the menu on a row standing for anything else', () => {
  it('offers neither run on a note', () => {
    expect(getItemIds('note')).not.toContain('transcribe')
    expect(getItemIds('note')).not.toContain('recognise')
  })

  it('offers neither run on a file the vault holds no source for', () => {
    expect(getItemIds('other')).not.toContain('transcribe')
    expect(getItemIds('other')).not.toContain('recognise')
  })

  // A folder of recordings is not a recording, and a run is over one file.
  it('offers neither run on a folder', () => {
    expect(getItemIds('recording', true)).not.toContain('transcribe')
    expect(getItemIds('book', true)).not.toContain('recognise')
  })

  it('offers neither off every row, where there is nothing to run it over', () => {
    const made = getMenuItems(null, false, canRunAnything).map((one) => one.id)
    expect(made).not.toContain('transcribe')
    expect(made).not.toContain('recognise')
  })
})

describe('where a run stands in the menu', () => {
  it('stands after the path and before the remove, parted from both by a rule', () => {
    expect(getGroupedIds('recording')).toStrictEqual([
      'newNote',
      'newDeck',
      'newStencil',
      'newPreset',
      'newFolder',
      'rename',
      'copy',
      '— transcribe',
      'proofread',
      '— remove',
    ])
  })

  it('leaves no rule behind on a row that has no run', () => {
    expect(getGroupedIds('other')).toStrictEqual([
      'newNote',
      'newDeck',
      'newStencil',
      'newPreset',
      'newFolder',
      'rename',
      'copy',
      '— remove',
    ])
  })
})

describe('disabling of runs based on artifact states', () => {
  it('disables transcribe and enables proofread when transcript is done', () => {
    const items = getMenuItems(
      { source: 'recording', isFolder: false, made: { transcript: 'done' } },
      false,
      canRunAnything,
    )
    const transcribe = items.find((one) => one.id === 'transcribe')
    const proofread = items.find((one) => one.id === 'proofread')
    expect(transcribe?.disabled).toBe(true)
    expect(proofread?.disabled).toBe(false)
  })

  it('disables both when transcript and proofread are done', () => {
    const items = getMenuItems(
      {
        source: 'recording',
        isFolder: false,
        made: { transcript: 'done', 'transcript.corrected': 'done' },
      },
      false,
      canRunAnything,
    )
    const transcribe = items.find((one) => one.id === 'transcribe')
    const proofread = items.find((one) => one.id === 'proofread')
    expect(transcribe?.disabled).toBe(true)
    expect(proofread?.disabled).toBe(true)
  })

  it('disables recognise and enables proofread when OCR is done on a PDF', () => {
    const items = getMenuItems(
      { source: 'book', isFolder: false, format: 'pdf', made: { ocr: 'done' } },
      false,
      canRunAnything,
    )
    const recognise = items.find((one) => one.id === 'recognise')
    const proofread = items.find((one) => one.id === 'proofread')
    expect(recognise?.disabled).toBe(true)
    expect(proofread?.disabled).toBe(false)
  })

  it('disables both when OCR and reading proofread are done on a PDF', () => {
    const items = getMenuItems(
      {
        source: 'book',
        isFolder: false,
        format: 'pdf',
        made: { ocr: 'done', 'ocr.corrected': 'done' },
      },
      false,
      canRunAnything,
    )
    const recognise = items.find((one) => one.id === 'recognise')
    const proofread = items.find((one) => one.id === 'proofread')
    expect(recognise?.disabled).toBe(true)
    expect(proofread?.disabled).toBe(true)
  })

  it('offers no scan runs on an EPUB book', () => {
    const items = getMenuItems(
      { source: 'book', isFolder: false, format: 'epub' },
      false,
      canRunAnything,
    )
    expect(items.some((one) => one.id === 'recognise')).toBe(false)
    expect(items.some((one) => one.id === 'proofread')).toBe(false)
  })
})

describe('the menu where this build cannot do a run at all', () => {
  it('offers the recording nothing, and leaves the scan its own run', () => {
    const but: RunGuard = (run) => run !== 'transcribe' && run !== 'proofread'

    expect(getItemIds('recording', false, but)).not.toContain('transcribe')
    expect(getItemIds('recording', false, but)).not.toContain('proofread')
    expect(getItemIds('book', false, but)).toContain('recognise')
  })
})

describe('the four files the menu makes', () => {
  it('offers a preset off every row, beside the note, the deck and the stencil', () => {
    expect(getMenuItems(null, false, canRunAnything).map((one) => one.id)).toStrictEqual([
      NEW_NOTE,
      NEW_DECK,
      NEW_STENCIL,
      NEW_PRESET,
      NEW_FOLDER,
    ])
  })

  it('offers a preset on a row of every kind', () => {
    for (const source of ['note', 'book', 'recording', 'other'] as Source[]) {
      expect(getItemIds(source), source).toContain(NEW_PRESET)
      expect(getItemIds(source, true), source).toContain(NEW_PRESET)
    }
  })

  it('offers a preset in the group the files stand in', () => {
    expect(getGroupedIds('note')).toContain(NEW_PRESET)
  })
})

describe('what the menu offers anywhere', () => {
  it('holds both runs, so choosing one is the menu’s own', () => {
    expect(OFFERED.has('transcribe')).toBe(true)
    expect(OFFERED.has('recognise')).toBe(true)
    expect(OFFERED.has('proofread')).toBe(true)
  })

  it('holds the preset, so choosing it is the menu’s own', () => {
    expect(OFFERED.has(NEW_PRESET)).toBe(true)
  })
})

describe('a run that is under way or finished', () => {
  const getDisabled = (
    source: 'recording' | 'book',
    made: ArtifactStates,
    id: string,
  ): boolean | undefined =>
    getMenuItems({ source, isFolder: false, format: 'pdf', made }, false, canRunAnything).find(
      (one) => one.id === id,
    )?.disabled

  it('disables transcribe while the transcript is running', () => {
    expect(getDisabled('recording', { transcript: 'running' }, 'transcribe')).toBe(true)
  })

  it('disables transcribe while the transcript is queued', () => {
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

describe('the menu on a book that is an EPUB', () => {
  it('offers exactly what a file offers', () => {
    const epub = getMenuItems(
      { source: 'book', isFolder: false, format: 'epub' },
      false,
      canRunAnything,
    ).map((one) => one.id)
    expect(epub).toStrictEqual(getItemIds('other'))
  })
})
