import { describe, expect, it, vi } from 'vitest'
import { CompletionContext } from '@codemirror/autocomplete'
import { EditorView } from '@codemirror/view'
import { createWikilinkSource } from './wikilinkCompletion'
import { createState } from '../fixtures/state'

describe('wikilink completion source', () => {
  const createTestContext = (doc: string, caret?: number) => {
    const pos = caret ?? doc.length
    const state = createState(doc, pos)
    return new CompletionContext(state, pos, true)
  }

  it('returns null when cursor is not preceded by double brackets', async () => {
    const source = createWikilinkSource({
      search: vi.fn().mockResolvedValue([]),
    })
    const context = createTestContext('Hello world')
    const result = await source(context)
    expect(result).toBeNull()
  })

  it('returns null when inside inline code or code blocks', async () => {
    const source = createWikilinkSource({
      search: vi.fn().mockResolvedValue([{ title: 'Physics' }]),
    })
    const context = createTestContext('`[[Physics`', 10)
    const result = await source(context)
    expect(result).toBeNull()
  })

  it('suggests matching note titles on [[query', async () => {
    const search = vi.fn().mockResolvedValue([
      { title: 'Quantum Mechanics' },
      { title: 'Quantum Field Theory' },
    ])
    const source = createWikilinkSource({ search })
    const context = createTestContext('Read [[Quantum')
    const result = await source(context)

    expect(result).not.toBeNull()
    expect(search).toHaveBeenCalledWith('Quantum')
    expect(result?.options).toHaveLength(2)
    expect(result?.options[0]?.label).toBe('Quantum Mechanics')
    expect(result?.options[1]?.label).toBe('Quantum Field Theory')
  })

  it('suggests section headings when query contains #', async () => {
    const headings = vi.fn().mockResolvedValue(['Introduction', 'Hamiltonian', 'Applications'])
    const source = createWikilinkSource({
      search: vi.fn(),
      headings,
    })
    const context = createTestContext('Read [[Quantum Mechanics#Ham')
    const result = await source(context)

    expect(result).not.toBeNull()
    expect(headings).toHaveBeenCalledWith('Quantum Mechanics')
    expect(result?.options).toHaveLength(1)
    expect(result?.options[0]?.label).toBe('Quantum Mechanics#Hamiltonian')
  })

  it('offers new note creation when title does not exist', async () => {
    const search = vi.fn().mockResolvedValue([])
    const createNote = vi.fn().mockResolvedValue('Quantum Gravity')
    const source = createWikilinkSource({ search, createNote })
    const context = createTestContext('Read [[Quantum Gravity')
    const result = await source(context)

    expect(result).not.toBeNull()
    const createOption = result?.options.find((opt) => opt.type === 'new')
    expect(createOption).toBeDefined()
    expect(createOption?.label).toBe('Create "Quantum Gravity"')
  })

  it('does not duplicate create option when exact match exists', async () => {
    const search = vi.fn().mockResolvedValue([{ title: 'Ontology' }])
    const createNote = vi.fn()
    const source = createWikilinkSource({ search, createNote })
    const context = createTestContext('Read [[Ontology')
    const result = await source(context)

    expect(result).not.toBeNull()
    const createOption = result?.options.find((opt) => opt.type === 'new')
    expect(createOption).toBeUndefined()
  })

  it('replaces trigger and inserts closing brackets upon selection', async () => {
    const search = vi.fn().mockResolvedValue([{ title: 'Relativity' }])
    const source = createWikilinkSource({ search })
    const context = createTestContext('See [[Rel')
    const result = await source(context)

    const option = result?.options[0]
    expect(option).toBeDefined()

    const view = new EditorView({ state: context.state })
    if (typeof option?.apply === 'function') {
      option.apply(view, option, result!.from, context.pos)
    }

    expect(view.state.doc.toString()).toBe('See [[Relativity]]')
    expect(view.state.selection.main.anchor).toBe('See [[Relativity]]'.length)
  })

  it('prevents duplicate closing brackets when already enclosed in [[]]', async () => {
    const search = vi.fn().mockResolvedValue([{ title: 'Relativity' }])
    const source = createWikilinkSource({ search })
    const text = 'See [[Rel]] and more'
    const caret = 'See [[Rel'.length
    const context = createTestContext(text, caret)
    const result = await source(context)

    const option = result?.options[0]
    const view = new EditorView({ state: context.state })
    if (typeof option?.apply === 'function') {
      option.apply(view, option, result!.from, caret)
    }

    expect(view.state.doc.toString()).toBe('See [[Relativity]] and more')
  })
})
