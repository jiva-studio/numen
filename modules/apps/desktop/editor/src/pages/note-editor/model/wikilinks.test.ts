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

  describe('headings of a written title', () => {
    const entry = (path: string, title: string, heading = '') => ({
      path,
      title,
      heading,
      line: heading ? 3 : -1,
      at: [],
      type: 'note' as const,
    })
    const createHeadings = (paths: Record<string, string[]>) =>
      new Map(
        Object.entries(paths).map(([path, texts]) => [
          path,
          texts.map((text) => ({ text, line: 1 })),
        ]),
      )

    it('does not look any note up when the title is empty', async () => {
      const deps = mockDeps({
        headings: vi.fn().mockResolvedValue(createHeadings({ 'cur.md': ['Own'] })),
      })

      await createWikilinkOptions(deps, () => 'cur.md').headings?.('')

      expect(deps.names).not.toHaveBeenCalled()
    })

    it('takes the note titled exactly the written title, wherever it stands in the matches', async () => {
      const deps = mockDeps({
        names: vi
          .fn()
          .mockResolvedValue([entry('a/Other.md', 'Other'), entry('b/Target.md', 'Target')]),
        headings: vi.fn().mockResolvedValue(createHeadings({ 'b/Target.md': ['Right'] })),
      })

      const result = await createWikilinkOptions(deps, () => 'cur.md').headings?.('Target')

      expect(deps.headings).toHaveBeenCalledWith(['b/Target.md'])
      expect(result).toEqual(['Right'])
    })

    it('compares titles without regard to case', async () => {
      const deps = mockDeps({
        names: vi
          .fn()
          .mockResolvedValue([entry('a/Other.md', 'Other'), entry('b/Target.md', 'Target')]),
        headings: vi.fn().mockResolvedValue(createHeadings({ 'b/Target.md': ['Right'] })),
      })

      await createWikilinkOptions(deps, () => 'cur.md').headings?.('tArGeT')

      expect(deps.headings).toHaveBeenCalledWith(['b/Target.md'])
    })

    it('skips a heading match of the same title in favour of the note', async () => {
      const deps = mockDeps({
        names: vi
          .fn()
          .mockResolvedValue([
            entry('a/Other.md', 'Target', 'Intro'),
            entry('b/Target.md', 'Target'),
          ]),
        headings: vi.fn().mockResolvedValue(createHeadings({ 'b/Target.md': ['Right'] })),
      })

      await createWikilinkOptions(deps, () => 'cur.md').headings?.('Target')

      expect(deps.headings).toHaveBeenCalledWith(['b/Target.md'])
    })

    it('falls back to the first match when no title is exact', async () => {
      const deps = mockDeps({
        names: vi
          .fn()
          .mockResolvedValue([entry('a/First.md', 'First note'), entry('b/Second.md', 'Second')]),
        headings: vi.fn().mockResolvedValue(createHeadings({ 'a/First.md': ['One'] })),
      })

      const result = await createWikilinkOptions(deps, () => 'cur.md').headings?.('Fir')

      expect(deps.headings).toHaveBeenCalledWith(['a/First.md'])
      expect(result).toEqual(['One'])
    })

    it('falls back to the current note when nothing matches', async () => {
      const deps = mockDeps({
        headings: vi.fn().mockResolvedValue(createHeadings({ 'cur.md': ['Own'] })),
      })

      const result = await createWikilinkOptions(deps, () => 'cur.md').headings?.('Nothing')

      expect(deps.headings).toHaveBeenCalledWith(['cur.md'])
      expect(result).toEqual(['Own'])
    })

    it('lists nothing when the note has no headings recorded', async () => {
      const deps = mockDeps({ headings: vi.fn().mockResolvedValue(new Map()) })

      const result = await createWikilinkOptions(deps, () => 'cur.md').headings?.('')

      expect(result).toStrictEqual([])
    })

    it('lists nothing when there is no path at all', async () => {
      const deps = mockDeps()

      const result = await createWikilinkOptions(deps, () => '').headings?.('')

      expect(result).toEqual([])
      expect(deps.headings).not.toHaveBeenCalled()
    })
  })

  it('creates no note, and answers nothing, when the vault refuses or fails', async () => {
    const refusing = mockDeps({ create: vi.fn().mockResolvedValue({ ok: false, error: 'no' }) })
    const failing = mockDeps({ create: vi.fn().mockRejectedValue(new Error('down')) })

    expect(await createWikilinkOptions(refusing, () => 'a.md').createNote?.('X')).toBeNull()
    expect(await createWikilinkOptions(failing, () => 'a.md').createNote?.('X')).toBeNull()
  })

  it('creates a CodeMirror extension', () => {
    const deps = mockDeps()
    const extension = createNoteWikilinkExtension(deps, () => 'current.md')
    expect(extension).toBeDefined()
  })
})
