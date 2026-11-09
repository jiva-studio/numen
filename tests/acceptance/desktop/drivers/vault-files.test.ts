import { describe, expect, it } from 'vitest'
import { assertPathsAreFree, getSampleKind, parseVaultChecks, parseVaultFiles } from './vault-files.js'

describe('parseVaultFiles', () => {
  it('reads a note, a folder, an empty folder path, a sample and an empty file', () => {
    expect(
      parseVaultFiles([
        ['path', 'text', 'sample'],
        ['Entropy.md', 'Entropy is a measure of uncertainty.', ''],
        ['daily/', '', ''],
        ['assets/shore.pdf', '', 'shore.pdf'],
        ['empty.txt', '', ''],
      ]),
    ).toEqual([
      { kind: 'note', path: 'Entropy.md', text: 'Entropy is a measure of uncertainty.' },
      { kind: 'folder', path: 'daily' },
      { kind: 'sample', path: 'assets/shore.pdf', sample: 'shore.pdf', sampleKind: 'PDF' },
      { kind: 'empty', path: 'empty.txt' },
    ])
  })

  it('takes a table without the columns it does not use', () => {
    expect(parseVaultFiles([['path'], ['daily/'], ['a.md']])).toEqual([
      { kind: 'folder', path: 'daily' },
      { kind: 'empty', path: 'a.md' },
    ])
    expect(parseVaultFiles([['path', 'sample'], ['a/b.epub', 'ledger.epub']])).toEqual([
      { kind: 'sample', path: 'a/b.epub', sample: 'ledger.epub', sampleKind: 'EPUB' },
    ])
  })

  it('refuses a column that is not allowed, and names the allowed ones', () => {
    expect(() => parseVaultFiles([['path', 'body'], ['a.md', 'x']])).toThrow(
      '"body" is not a column of the files table: use path | text | sample',
    )
  })

  it('refuses columns in another order, a repeated column and a table without the path column', () => {
    expect(() => parseVaultFiles([['path', 'sample', 'text'], ['a.md', '', '']])).toThrow('are in the order path | text | sample, each once')
    expect(() => parseVaultFiles([['path', 'text', 'text'], ['a.md', '', '']])).toThrow('are in the order')
    expect(() => parseVaultFiles([['text', 'path'], ['x', 'a.md']])).toThrow('starts with the column "path"')
    expect(() => parseVaultFiles([[], ['a.md']])).toThrow('starts with the column "path"')
  })

  it('refuses a table without rows and a row without a path', () => {
    expect(() => parseVaultFiles([['path', 'text']])).toThrow('the files table has no row')
    expect(() => parseVaultFiles([['path', 'text'], ['', 'x']])).toThrow('a row of the files has no path')
  })

  it('refuses a text and a sample in one row', () => {
    expect(() => parseVaultFiles([['path', 'text', 'sample'], ['a.pdf', 'x', 'shore.pdf']])).toThrow(
      '"a.pdf" has a text and a sample: give one of them',
    )
  })

  it('refuses a text in a folder, and a text in a file that is not a note', () => {
    expect(() => parseVaultFiles([['path', 'text'], ['daily/', 'x']])).toThrow('"daily/" is a folder: a folder has no text and no sample')
    expect(() => parseVaultFiles([['path', 'text'], ['a.txt', 'x']])).toThrow('"a.txt" has a text but is not a .md file')
  })

  it('refuses a path twice, a folder with or without its slash alike', () => {
    expect(() => parseVaultFiles([['path'], ['a.md'], ['a.md']])).toThrow('"a.md" is in the files twice')
    expect(() => parseVaultFiles([['path'], ['daily/'], ['daily/']])).toThrow('"daily/" is in the files twice')
  })

  it('refuses a path that leaves the vault', () => {
    expect(() => parseVaultFiles([['path'], ['../a.md']])).toThrow('is not a path inside the vault')
    expect(() => parseVaultFiles([['path'], ['/a.md']])).toThrow('is not a path inside the vault')
  })

  it('refuses a sample that is not of a known kind', () => {
    expect(() => parseVaultFiles([['path', 'sample'], ['a.doc', 'a.doc']])).toThrow('"a.doc" is not a sample: a sample is a .pdf')
  })
})

describe('getSampleKind', () => {
  it.each([
    ['shore.pdf', 'PDF'],
    ['ledger.epub', 'EPUB'],
    ['sample.wav', 'recording'],
    ['talk.mp3', 'recording'],
    ['sample.flac', 'recording'],
    ['llama.png', 'image'],
    ['Shore.PDF', 'PDF'],
  ])('types %s as %s', (fileName, kind) => {
    expect(getSampleKind(fileName)).toBe(kind)
  })
})

describe('assertPathsAreFree', () => {
  const writes = parseVaultFiles([['path'], ['a.md'], ['daily/']])

  it('accepts paths the vault does not have', () => {
    expect(() => assertPathsAreFree(writes, () => false)).not.toThrow()
  })

  it('names the first path the vault has', () => {
    expect(() => assertPathsAreFree(writes, (entryPath) => entryPath === 'daily')).toThrow('"daily/" is in the vault already')
    expect(() => assertPathsAreFree(writes, (entryPath) => entryPath === 'a.md')).toThrow('"a.md" is in the vault already')
  })
})

describe('parseVaultChecks', () => {
  it('reads files, folders and notes with a text', () => {
    expect(parseVaultChecks([['path', 'text'], ['a.md', 'tea'], ['daily/', ''], ['b.pdf', '']])).toEqual([
      { path: 'a.md', isFolder: false, text: 'tea' },
      { path: 'daily', isFolder: true, text: '' },
      { path: 'b.pdf', isFolder: false, text: '' },
    ])
    expect(parseVaultChecks([['path'], ['a.md']])).toEqual([{ path: 'a.md', isFolder: false, text: '' }])
  })

  it('refuses a sample column, and names the allowed ones', () => {
    expect(() => parseVaultChecks([['path', 'sample'], ['a.pdf', 'shore.pdf']])).toThrow(
      '"sample" is not a column of the files table: use path | text',
    )
  })

  it('refuses a text for a folder and a row without a path', () => {
    expect(() => parseVaultChecks([['path', 'text'], ['daily/', 'x']])).toThrow('"daily/" is a folder: a folder has no text')
    expect(() => parseVaultChecks([['path', 'text'], ['', 'x']])).toThrow('a row of the files has no path')
  })
})
