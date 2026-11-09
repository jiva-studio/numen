import { describe, expect, it } from 'vitest'
import {
  parseApplication,
  getRating,
  parseReviewScope,
  assertEntry,
  assertPosition,
  assertSingleTab,
  getAddressTitle,
  getAncestors,
  getStem,
  getTitlePattern,
  getPaletteField,
  parseCount,
  parseReference,
  parsePlace,
  parseTabKind,
  parseTime,
  pickKindTab,
  pickByTitle,
  readNoteType,
  assertWorldApplication,
} from './rules.js'

describe('parseReference', () => {
  it('takes at, titled and for the web address', () => {
    expect(parseReference('at', 'notes/A.md')).toEqual({ reference: 'at', name: 'notes/A.md' })
    expect(parseReference('titled', 'A')).toEqual({ reference: 'titled', name: 'A' })
    expect(parseReference('for the web address', 'http://a.test/')).toEqual({
      reference: 'for the web address',
      name: 'http://a.test/',
    })
  })

  it('throws for any other word', () => {
    expect(() => parseReference('named', 'A')).toThrow(
      '"named" does not name a tab or a note: use "at", "titled" or "for the web address"',
    )
  })
})

describe('getAddressTitle', () => {
  it('replaces the characters a file name cannot have', () => {
    expect(getAddressTitle('http://numen.dev/one')).toBe('http---numen.dev-one')
    expect(getAddressTitle('https://www.youtube.com/watch?v=tidalmill01')).toBe('https---www.youtube.com-watch-v=tidalmill01')
  })
})

describe('getTitlePattern', () => {
  it('matches a title, and the name of a tab with its close button', () => {
    expect(getTitlePattern('rance').test('rance')).toBe(true)
    expect(getTitlePattern('rance').test('rance Close rance')).toBe(true)
    expect(getTitlePattern('rance').test('Linked tab rance Close rance')).toBe(true)
  })

  it('does not match a title that starts or goes on beyond the words', () => {
    expect(getTitlePattern('rance').test('http---a.test-history-rance')).toBe(false)
    expect(getTitlePattern('rance').test('rances')).toBe(false)
  })

  it('takes the words as they are, not as a pattern', () => {
    expect(getTitlePattern('a.c').test('abc')).toBe(false)
  })
})

describe('assertPosition', () => {
  it('passes the position of each kind of tab', () => {
    expect(() => assertPosition('PDF Reader', 'page', 'a.pdf')).not.toThrow()
    expect(() => assertPosition('EPUB Reader', 'page', 'a.epub')).not.toThrow()
    expect(() => assertPosition('Audio Player', 'time', 'a.mp3')).not.toThrow()
  })

  it('throws for a position of the wrong kind', () => {
    expect(() => assertPosition('PDF Reader', 'time', 'a.pdf')).toThrow(
      'the tab "a.pdf" is a PDF Reader tab: its position is a page, not a time',
    )
    expect(() => assertPosition('Audio Player', 'page', 'a.mp3')).toThrow(
      'its position is a time, not a page',
    )
  })
})

describe('parseTime', () => {
  it('reads minutes and seconds', () => {
    expect(parseTime('2:00')).toBe(120)
    expect(parseTime('0:45')).toBe(45)
  })

  it('throws for text that is not a time', () => {
    expect(() => parseTime('2')).toThrow('"2" is not a time: write minutes and seconds, like "2:00"')
    expect(() => parseTime('2:75')).toThrow('is not a time')
  })
})

describe('getPaletteField', () => {
  it('reads the placeholders of a name and of a list', () => {
    expect(getPaletteField('Name')).toBe('name')
    expect(getPaletteField('Search')).toBe('item')
    expect(getPaletteField('Search notes')).toBe('note')
  })

  it('reads any other placeholder as a text', () => {
    expect(getPaletteField('Web address')).toBe('text')
  })
})

describe('parseApplication', () => {
  it('takes the editor', () => {
    expect(parseApplication('editor')).toBe('editor')
  })

  it('accepts the flashcards window', () => {
    expect(parseApplication('flashcards')).toBe('flashcards')
  })

  it('throws for a word that is no application', () => {
    expect(() => parseApplication('mail')).toThrow('"mail" is not an application')
  })
})

describe('parseCount', () => {
  it('reads digits as a number', () => {
    expect(parseCount('12')).toBe(12)
  })

  it('throws for other text', () => {
    expect(() => parseCount('two')).toThrow('"two" is not a number')
  })
})

describe('readNoteType', () => {
  it('reads the type key of the frontmatter', () => {
    expect(readNoteType('---\ntype: link\nurl: http://a.test/\n---\ntext')).toBe('link')
    expect(readNoteType('---\ntype: "deck"\n---\n')).toBe('deck')
  })

  it('gives nothing for a note without a type', () => {
    expect(readNoteType('---\ntitle: A\n---\n')).toBe('')
    expect(readNoteType('type: link\n')).toBe('')
  })
})

describe('pickByTitle', () => {
  it('finds the one path with the title', () => {
    expect(getStem('assets/shore.pdf')).toBe('shore')
    expect(pickByTitle('shore', ['assets/shore.pdf', 'assets/talk.mp3'])).toBe('assets/shore.pdf')
  })

  it('throws when no path has the title', () => {
    expect(() => pickByTitle('shore', ['assets/talk.mp3'])).toThrow(
      'no tab is titled "shore": use "at" with its path',
    )
  })

  it('throws when two paths have the title', () => {
    expect(() => pickByTitle('shore', ['a/shore.pdf', 'b/shore.md'])).toThrow(
      'more than one tab is titled "shore": use "at" with its path',
    )
  })
})

describe('assertSingleTab', () => {
  it('passes one tab and throws for two', () => {
    expect(() => assertSingleTab('Shopping list', 1)).not.toThrow()
    expect(() => assertSingleTab('Shopping list', 2)).toThrow(
      '2 tabs are titled "Shopping list": use "at" with the path',
    )
  })
})

describe('assertEntry', () => {
  it('passes a file that is a file and a folder that is a folder', () => {
    expect(() => assertEntry('file', 'notes/A.md', false)).not.toThrow()
    expect(() => assertEntry('folder', 'notes', true)).not.toThrow()
  })

  it('throws when the entry is the other kind or not a kind', () => {
    expect(() => assertEntry('folder', 'notes/A.md', false)).toThrow('"notes/A.md" is a file, not a folder')
    expect(() => assertEntry('file', 'notes', true)).toThrow('"notes" is a folder, not a file')
    expect(() => assertEntry('note', 'notes', true)).toThrow('"note" is not a file or a folder')
  })
})

describe('getAncestors', () => {
  it('lists the folders above a path, the top one first', () => {
    expect(getAncestors('a/b/c.md')).toEqual(['a', 'a/b'])
    expect(getAncestors('a.md')).toEqual([])
  })
})

describe('getRating', () => {
  it('numbers the four answers', () => {
    expect(['Again', 'Hard', 'Good', 'Easy'].map(getRating)).toEqual([1, 2, 3, 4])
  })

  it('throws for a word that is no answer', () => {
    expect(() => getRating('Maybe')).toThrow('"Maybe" is not an answer')
  })
})

describe('parseReviewScope', () => {
  it('passes a vault and a deck', () => {
    expect(['vault', 'deck'].map(parseReviewScope)).toEqual(['vault', 'deck'])
  })

  it('throws for any other word', () => {
    expect(() => parseReviewScope('card')).toThrow('"card" is not reviewed')
  })
})

describe('parseTabKind', () => {
  it('takes the tab as a tab of any kind', () => {
    expect(parseTabKind('tab')).toBe('any')
  })

  it('takes the word before tab as the kind', () => {
    expect(parseTabKind('note tab')).toBe('note')
    expect(parseTabKind('Plex tab')).toBe('Plex')
    expect(parseTabKind('PDF tab')).toBe('PDF')
    expect(parseTabKind('book tab')).toBe('book')
    expect(parseTabKind('recording tab')).toBe('recording')
    expect(parseTabKind('Files tab')).toBe('Files')
    expect(parseTabKind('Agent tab')).toBe('Agent')
  })

  it('fails on a word that is not a kind and lists the words', () => {
    expect(() => parseTabKind('chart tab')).toThrow(
      'the tab kind "chart" is not a tab kind: use note, Plex, PDF, book, recording, Files, Agent',
    )
  })
})

describe('pickKindTab', () => {
  const plex = { id: 'plex:1', kind: 'plex' }
  const note = { id: 'note:1', kind: 'note' }

  it('picks the tab of the kind among tabs that share the title', () => {
    expect(pickKindTab('note', [plex, note], 'at', 'Entropy.md')).toBe(note)
    expect(pickKindTab('Plex', [plex, note], 'titled', 'Entropy')).toBe(plex)
  })

  it('picks the file tree and the chat by the ids of their manifests', () => {
    const files = { id: 'files:1', kind: 'files' }
    const agent = { id: 'agent:1', kind: 'agent' }
    expect(pickKindTab('Files', [agent, files], 'titled', 'Files')).toBe(files)
    expect(pickKindTab('Agent', [files, agent], 'titled', 'Agent')).toBe(agent)
  })

  it('finds none where no tab matches the reference', () => {
    expect(pickKindTab('note', [], 'at', 'Entropy.md')).toBeUndefined()
  })

  it('fails with the kind the tab has when it is of another kind', () => {
    expect(() => pickKindTab('note', [plex], 'at', 'Entropy.md')).toThrow(
      'the tab at "Entropy.md" is a Plex tab, not a Note tab',
    )
    expect(() => pickKindTab('PDF', [{ id: 'x', kind: 'reader:reader-epub' }], 'at', 'a.pdf')).toThrow(
      'is a EPUB Reader tab, not a PDF Reader tab',
    )
  })

  it('fails when two tabs of the kind match', () => {
    expect(() => pickKindTab('note', [note, { id: 'note:2', kind: 'note' }], 'titled', 'A')).toThrow(
      '2 Note tabs match "A": use "at" with the path',
    )
  })
})

describe('parsePlace', () => {
  it('takes a page or a time with its value', () => {
    expect(parsePlace('page "2"')).toEqual({ position: 'page', where: '2' })
    expect(parsePlace('time "2:00"')).toEqual({ position: 'time', where: '2:00' })
  })

  it('fails on anything else', () => {
    expect(() => parsePlace('line "2"')).toThrow('"line "2"" is not a place: write page "2" or time "2:00"')
  })
})

describe('assertWorldApplication', () => {
  it('takes the editor on any number of vaults, and the flashcards window on one', () => {
    expect(() => assertWorldApplication('editor', ['Test', 'Archive'], undefined)).not.toThrow()
    expect(() => assertWorldApplication('flashcards', ['Test'], undefined)).not.toThrow()
  })

  it('takes the application that runs again', () => {
    expect(() => assertWorldApplication('editor', ['Test'], 'editor')).not.toThrow()
  })

  it('says the flashcards window opens one vault', () => {
    expect(() => assertWorldApplication('flashcards', ['Test', 'Archive'], undefined)).toThrow(
      'opens one vault, and the world has Test and Archive',
    )
  })

  it('refuses a second application', () => {
    expect(() => assertWorldApplication('flashcards', ['Test'], 'editor')).toThrow('a world has one application')
  })
})
