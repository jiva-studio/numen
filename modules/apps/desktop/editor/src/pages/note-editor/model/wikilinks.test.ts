import { describe, expect, it, vi } from 'vitest'
import { createWikilinkOptions, createNoteWikilinkExtension } from './wikilinks'
import type { NoteTabDeps } from '../types'

describe('note wikilink completion adapter', () => {
  const mockDeps = (overrides?: Partial<NoteTabDeps>): NoteTabDeps => ({
    neighbourhood: vi.fn().mockResolvedValue({ focus: { title: '' } }),
    names: vi.fn().mockResolvedValue([]),
    headings: vi.fn().mockResolvedValue(new Map()),
    create: vi.fn().mockResolvedValue({ ok: true, value: { path: 'notes/Created.md' } }),
    resolve: vi.fn().mockResolvedValue(new Map()),
    ...overrides,
  })

  it('searches note names and maps to WikilinkOption items', async () => {
    const deps = mockDeps({
      names: vi.fn().mockResolvedValue([
        {
          path: 'physics/Quantum.md',
          title: 'Quantum Mechanics',
          heading: '',
          line: -1,
          at: [],
          type: 'note',
        },
        {
          path: 'physics/Quantum.md',
          title: 'Quantum Mechanics',
          heading: 'Introduction',
          line: 5,
          at: [],
          type: 'note',
        },
      ]),
    })

    const options = createWikilinkOptions(deps, () => 'current.md')
    const results = await options.search('Quantum')

    expect(deps.names).toHaveBeenCalledWith('Quantum', 20)
    expect(results).toHaveLength(2)
    expect(results[0]).toEqual({
      title: 'Quantum Mechanics',
      type: 'note',
    })
    expect(results[1]).toEqual({
      title: 'Quantum Mechanics',
      heading: 'Introduction',
      detail: 'Quantum Mechanics',
      type: 'heading',
    })
  })

  it('fetches headings for current note when title is empty', async () => {
    const headingsMap = new Map([
      [
        'notes/Current.md',
        [
          { text: 'Section 1', line: 1 },
          { text: 'Section 2', line: 10 },
        ],
      ],
    ])
    const deps = mockDeps({
      headings: vi.fn().mockResolvedValue(headingsMap),
    })

    const options = createWikilinkOptions(deps, () => 'notes/Current.md')
    const results = await options.headings?.('')

    expect(deps.headings).toHaveBeenCalledWith(['notes/Current.md'])
    expect(results).toEqual(['Section 1', 'Section 2'])
  })

  it('resolves note path and fetches headings when note title is provided', async () => {
    const headingsMap = new Map([['notes/Relativity.md', [{ text: 'Spacetime', line: 4 }]]])
    const deps = mockDeps({
      names: vi.fn().mockResolvedValue([
        {
          path: 'notes/Relativity.md',
          title: 'General Relativity',
          heading: '',
          line: -1,
          at: [],
          type: 'note',
        },
      ]),
      headings: vi.fn().mockResolvedValue(headingsMap),
    })

    const options = createWikilinkOptions(deps, () => 'notes/Current.md')
    const results = await options.headings?.('General Relativity')

    expect(deps.names).toHaveBeenCalledWith('General Relativity', 5)
    expect(deps.headings).toHaveBeenCalledWith(['notes/Relativity.md'])
    expect(results).toEqual(['Spacetime'])
  })

  it('creates new note and returns path', async () => {
    const deps = mockDeps({
      create: vi.fn().mockResolvedValue({ ok: true, value: { path: 'notes/Gravity.md' } }),
    })

    const options = createWikilinkOptions(deps, () => 'notes/Current.md')
    const createdPath = await options.createNote?.('Gravity')

    expect(deps.create).toHaveBeenCalledWith({ title: 'Gravity', folder: '', links: [] })
    expect(createdPath).toBe('notes/Gravity.md')
  })

  it('creates a CodeMirror extension', () => {
    const deps = mockDeps()
    const extension = createNoteWikilinkExtension(deps, () => 'current.md')
    expect(extension).toBeDefined()
  })
})
