/** The target variables of when conditions, derived from the path a place passes. Every path is invented. */
import { describe, expect, it } from 'vitest'
import { getTargetValues } from './targetValues'

const TYPES = { md: 'text/markdown', pdf: 'application/pdf', mp3: 'audio/mpeg' }

describe('the target values', () => {
  it('are those of the root folder for the folder of the empty path', () => {
    expect(getTargetValues({ path: '', isFolder: true, others: [] }, TYPES)).toStrictEqual({
      'target.isRoot': true,
      'target.isFolder': true,
      'target.isMultiple': false,
      'target.isNote': false,
      'target.isFile': false,
      'target.contentType': '',
      'target.noteType': '',
      'target.properties': {},
    })
  })

  it('are those of a note, with its type, and of one of several paths', () => {
    expect(
      getTargetValues(
        { path: 'notes/Bees.md', isFolder: false, others: ['notes/Wasps.md'] },
        TYPES,
        { type: 'link' },
      ),
    ).toStrictEqual({
      'target.isRoot': false,
      'target.isFolder': false,
      'target.isMultiple': true,
      'target.isNote': true,
      'target.isFile': false,
      'target.contentType': 'text/markdown',
      'target.noteType': 'link',
      'target.properties': {},
    })
  })

  it('carry the properties of a note, and none of a file that is no note', () => {
    const properties = { url: 'https://bees.example/hive' }
    const note = { path: 'notes/Bees.md', isFolder: false, others: [] }
    const paper = { path: 'papers/Hive.pdf', isFolder: false, others: [] }

    expect(getTargetValues(note, TYPES, { type: 'link', properties })['target.properties']).toBe(
      properties,
    )
    expect(getTargetValues(paper, TYPES, { properties })['target.properties']).toStrictEqual({})
  })

  it('are those of a file that is no note, with its content type by its extension in any case', () => {
    expect(
      getTargetValues({ path: 'papers/Hive.PDF', isFolder: false, others: [] }, TYPES),
    ).toMatchObject({
      'target.isNote': false,
      'target.isFile': true,
      'target.contentType': 'application/pdf',
      'target.noteType': '',
      'target.properties': {},
    })
  })

  it('are those of a folder that is not the root', () => {
    expect(getTargetValues({ path: 'notes', isFolder: true, others: [] }, TYPES)).toMatchObject({
      'target.isRoot': false,
      'target.isFolder': true,
      'target.isFile': false,
      'target.contentType': '',
    })
  })

  it('are all false or empty where the place passes no target', () => {
    expect(getTargetValues(undefined, TYPES)).toStrictEqual({
      'target.isRoot': false,
      'target.isFolder': false,
      'target.isMultiple': false,
      'target.isNote': false,
      'target.isFile': false,
      'target.contentType': '',
      'target.noteType': '',
      'target.properties': {},
    })
  })
})
