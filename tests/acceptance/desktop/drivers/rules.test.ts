import { describe, expect, it } from 'vitest'
import {
  parseApplication,
  getRating,
  parseReviewScope,
  assertEntry,
  assertPosition,
  assertSingleTab,
  getAddressTitle,
  getLinkNotePath,
  getAncestors,
  getStem,
  getTitlePattern,
  assertPaletteField,
  getCurrentVersionLabel,
  getPaletteField,
  parseReference,
  parsePlace,
  parseTabKind,
  parseTime,
  pickKindTab,
  pickByTitle,
  readNoteType,
  assertWorldApplication,
  isAskedFor,
  parseArguments,
  parseHeaded,
  parseKeys,
  parseTabRef,
  parseTabRows,
  parseTarget,
  readLinkAddress,
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

describe('getLinkNotePath', () => {
  it('names the note file of an address', () => {
    expect(getLinkNotePath('https://youtu.be/mill01')).toBe('https---youtu.be-mill01.md')
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
    expect(getPaletteField('Search vaults')).toBe('vault')
  })

  it('reads any other placeholder as a text', () => {
    expect(getPaletteField('Web address')).toBe('text')
  })
})

describe('assertPaletteField', () => {
  it('lets a name go to a name, a pick to a list or a note search, a text to a text', () => {
    expect(() => assertPaletteField('name', 'name')).not.toThrow()
    expect(() => assertPaletteField('pick', 'item')).not.toThrow()
    expect(() => assertPaletteField('pick', 'note')).not.toThrow()
    expect(() => assertPaletteField('pick', 'vault')).not.toThrow()
    expect(() => assertPaletteField('type', 'text')).not.toThrow()
  })

  it('names the step that does not match the palette', () => {
    expect(() => assertPaletteField('type', 'item')).toThrow('the person types here, but the palette asks for an item to pick')
    expect(() => assertPaletteField('pick', 'name')).toThrow('the person picks here, but the palette asks for a name')
    expect(() => assertPaletteField('name', 'text')).toThrow('the person names it here, but the palette asks for a text')
  })
})

describe('getCurrentVersionLabel', () => {
  it('gives the title the product shows for the live version of each kind of text', () => {
    expect(getCurrentVersionLabel('recognized text')).toBe('Recognized text (current)')
    expect(getCurrentVersionLabel('transcript')).toBe('Transcript (current)')
    expect(getCurrentVersionLabel('proofread text')).toBe('Proofread text (current)')
    expect(getCurrentVersionLabel('article')).toBe('Article (current)')
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
    expect(parseTabKind('Deck tab')).toBe('Deck')
    expect(parseTabKind('Preset tab')).toBe('Preset')
  })

  it('takes two words before tab as the kind', () => {
    expect(parseTabKind('Card Model tab')).toBe('Card Model')
  })

  it('fails on a word that is not a kind and lists the words', () => {
    expect(() => parseTabKind('chart tab')).toThrow(
      'the tab kind "chart" is not a tab kind: use note, Plex, PDF, book, recording, Media, Files, Agent, Settings, Deck, Card Model, Preset',
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

describe('parseTarget', () => {
  it('takes nothing, a file, a folder and a tab', () => {
    expect(parseTarget('')).toEqual({ kind: 'none' })
    expect(parseTarget(' on the file at "notes/A.md"')).toEqual({ kind: 'entry', entry: 'file', path: 'notes/A.md' })
    expect(parseTarget(' on the folder at "Glaciers"')).toEqual({ kind: 'entry', entry: 'folder', path: 'Glaciers' })
    expect(parseTarget(' on the Plex tab titled "Heat"')).toEqual({ kind: 'tab', tab: 'Plex', reference: 'titled', name: 'Heat' })
  })

  it('fails on a target that is none of them', () => {
    expect(() => parseTarget(' on the vault')).toThrow('does not name what the command runs on')
  })
})

describe('parseArguments', () => {
  it('reads a name and a value from each row', () => {
    expect(parseArguments([['Name', 'Brine log']])).toEqual([['Name', 'Brine log']])
  })

  it('fails on a row that is not a pair', () => {
    expect(() => parseArguments([['Name']])).toThrow('must have a name and a value')
  })
})

describe('isAskedFor', () => {
  it('takes the placeholder and the title of a list', () => {
    expect(isAskedFor('Name', 'Name', [])).toBe(true)
    expect(isAskedFor('Theme', 'Search', ['Theme'])).toBe(true)
    expect(isAskedFor('Title', 'Name', ['Name'])).toBe(false)
  })
})

describe('parseKeys', () => {
  it('names the keys the way Playwright does', () => {
    expect(parseKeys('Ctrl+Shift+E')).toBe('Control+Shift+E')
  })

  it('fails on a modifier that is not known', () => {
    expect(() => parseKeys('Hyper+E')).toThrow('is not a key combination')
  })
})

describe('parseHeaded', () => {
  it('gives the rows below the header', () => {
    expect(parseHeaded([['vault'], ['Test'], ['Coastline']], 'vault')).toEqual(['Test', 'Coastline'])
  })

  it('fails on another header', () => {
    expect(() => parseHeaded([['name'], ['Test']], 'vault')).toThrow('must start with the header "| vault |"')
  })
})

describe('parseTabRows', () => {
  it('reads the kind and the path of each tab', () => {
    expect(parseTabRows([['kind', 'path'], ['note', 'notes/A.md']])).toEqual([{ kind: 'note', path: 'notes/A.md' }])
  })

  it('fails on a kind that is not a tab kind', () => {
    expect(() => parseTabRows([['kind', 'path'], ['chart', 'a']])).toThrow('is not a tab kind')
  })
})

describe('parseTabRef', () => {
  it('reads the kind, the reference and the name', () => {
    expect(parseTabRef('Deck tab titled "Shore birds"')).toEqual({ kind: 'Deck', reference: 'titled', name: 'Shore birds' })
  })
})

describe('readLinkAddress', () => {
  it('reads the address of a link note', () => {
    expect(readLinkAddress('---\ntype: link\nurl: http://numen.dev/tides\n---\n')).toBe('http://numen.dev/tides')
  })

  it('leaves a note that is not a link', () => {
    expect(readLinkAddress('---\ntype: note\nurl: http://a\n---\n')).toBeUndefined()
  })
})
