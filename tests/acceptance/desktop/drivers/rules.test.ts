import { describe, expect, it } from 'vitest'
import {
  assertApplication,
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
  parseTime,
  pickByTitle,
  readNoteType,
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

describe('assertApplication', () => {
  it('passes the editor', () => {
    expect(() => assertApplication('editor')).not.toThrow()
  })

  it('accepts the flashcards window', () => {
    expect(() => assertApplication('flashcards')).not.toThrow()
  })

  it('throws for a word that is no application', () => {
    expect(() => assertApplication('mail')).toThrow('"mail" is not an application')
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
