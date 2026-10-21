import { describe, expect, it, vi } from 'vitest'
import { CompletionContext, currentCompletions, startCompletion } from '@codemirror/autocomplete'
import { EditorState } from '@codemirror/state'
import { EditorView } from '@codemirror/view'
import {
  createWikilinkCompletion,
  createWikilinkSource,
  type WikilinkOption,
} from './wikilinkCompletion'
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
    const search = vi
      .fn()
      .mockResolvedValue([{ title: 'Quantum Mechanics' }, { title: 'Quantum Field Theory' }])
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

describe('wikilink completion in other places of a document', () => {
  const getContext = (doc: string, caret: number) => {
    const state = createState(doc, caret)
    return new CompletionContext(state, caret, true)
  }

  /** The document after an option was chosen, with the caret at its current position. */
  const choose = (
    doc: string,
    caret: number,
    option: { apply?: unknown } | undefined,
    from: number,
  ) => {
    const view = new EditorView({ state: createState(doc, caret) })
    if (typeof option?.apply !== 'function') throw new Error('the option has nothing to apply')
    option.apply(view, option, from, caret)
    return { text: view.state.doc.toString(), caret: view.state.selection.main.anchor }
  }

  describe('code', () => {
    it('is not completed in a fenced block', async () => {
      const search = vi.fn().mockResolvedValue([{ title: 'Physics' }])
      const doc = '```\n[[Phys\n```'

      const result = await createWikilinkSource({ search })(getContext(doc, 10))

      expect(result).toBeNull()
      expect(search).not.toHaveBeenCalled()
    })

    it('is not completed in an indented block', async () => {
      const search = vi.fn().mockResolvedValue([{ title: 'Physics' }])
      const doc = 'Text.\n\n    [[Phys more'

      const result = await createWikilinkSource({ search })(
        getContext(doc, doc.length - ' more'.length),
      )

      expect(result).toBeNull()
      expect(search).not.toHaveBeenCalled()
    })

    it('is completed on the line after a fenced block is closed', async () => {
      const search = vi.fn().mockResolvedValue([{ title: 'Physics' }])
      const doc = '```\ncode\n```\n[[Phys'

      const result = await createWikilinkSource({ search })(getContext(doc, doc.length))

      expect(result?.options).toHaveLength(1)
    })
  })

  describe('the query', () => {
    it('is read from the line the caret relies on, up to the caret', async () => {
      const search = vi.fn().mockResolvedValue([])
      const doc = 'First line\nSee [[Rel and more'

      await createWikilinkSource({ search })(getContext(doc, 'First line\nSee [[Rel'.length))

      expect(search).toHaveBeenCalledWith('Rel')
    })

    it('is the one after the last opened link where an earlier one is closed', async () => {
      const search = vi.fn().mockResolvedValue([])
      const doc = 'See [[Alpha]] and [[Be'

      await createWikilinkSource({ search })(getContext(doc, doc.length))

      expect(search).toHaveBeenCalledWith('Be')
    })

    it('is the one after the last opened brackets where an earlier pair is left open', async () => {
      const search = vi.fn().mockResolvedValue([])
      const doc = 'See [[Alpha [[Be'

      await createWikilinkSource({ search })(getContext(doc, doc.length))

      expect(search).toHaveBeenCalledWith('Be')
    })

    it('is not read where the link before the caret is closed', async () => {
      const search = vi.fn().mockResolvedValue([])
      const doc = 'See [[Alpha] and more'

      const result = await createWikilinkSource({ search })(getContext(doc, doc.length))

      expect(result).toBeNull()
      expect(search).not.toHaveBeenCalled()
    })

    it('is trimmed of the space around it', async () => {
      const search = vi.fn().mockResolvedValue([])
      const doc = 'See [[  Quantum  '

      await createWikilinkSource({ search })(getContext(doc, doc.length))

      expect(search).toHaveBeenCalledWith('Quantum')
    })

    it('starts the replaced text after the two brackets', async () => {
      const search = vi.fn().mockResolvedValue([{ title: 'Relativity' }])
      const doc = 'First line\nSee [[Rel'

      const result = await createWikilinkSource({ search })(getContext(doc, doc.length))

      expect(result?.from).toBe('First line\nSee '.length + 2)
    })

    it('is not filtered again by the editor', async () => {
      const search = vi.fn().mockResolvedValue([{ title: 'Relativity' }])

      const result = await createWikilinkSource({ search })(getContext('[[Rel', 5))

      expect(result?.filter).toBe(false)
    })
  })

  describe('the notes', () => {
    it('are offered with the detail and the type they bring', async () => {
      const search = vi
        .fn()
        .mockResolvedValue([{ title: 'Physics', detail: 'in Science', type: 'heading' }])

      const result = await createWikilinkSource({ search })(getContext('[[Ph', 4))

      expect(result?.options[0]).toMatchObject({ detail: 'in Science', type: 'heading' })
    })

    it('are offered as a note where they bring neither and name no heading', async () => {
      const search = vi.fn().mockResolvedValue([{ title: 'Physics' }])

      const result = await createWikilinkSource({ search })(getContext('[[Ph', 4))

      expect(result?.options[0]).toMatchObject({ label: 'Physics', detail: 'note', type: 'note' })
    })

    it('are offered as a heading where they name one, and the link names both', async () => {
      const search = vi.fn().mockResolvedValue([{ title: 'Physics', heading: 'Optics' }])
      const doc = 'See [[Ph'

      const result = await createWikilinkSource({ search })(getContext(doc, doc.length))

      const option = result?.options[0]
      expect(option).toMatchObject({ label: 'Physics#Optics', detail: 'heading', type: 'heading' })
      expect(choose(doc, doc.length, option, result!.from).text).toBe('See [[Physics#Optics]]')
    })

    it('are chosen in a line after the first, replacing the trigger only', async () => {
      const search = vi.fn().mockResolvedValue([{ title: 'Relativity' }])
      const doc = 'First line\nSee [[Rel]] and more'
      const caret = 'First line\nSee [[Rel'.length

      const result = await createWikilinkSource({ search })(getContext(doc, caret))

      const chosen = choose(doc, caret, result?.options[0], result!.from)
      expect(chosen.text).toBe('First line\nSee [[Relativity]] and more')
      expect(chosen.caret).toBe('First line\nSee [[Relativity]]'.length)
    })
  })

  describe('a note to create', () => {
    const create = async (
      results: readonly { title: string; heading?: string }[],
      doc: string,
      createNote: ((title: string) => Promise<string | null>) | undefined,
    ) => {
      const search = vi.fn().mockResolvedValue(results)
      const result = await createWikilinkSource({
        search,
        ...(createNote ? { createNote } : {}),
      })(getContext(doc, doc.length))
      return { result, option: result?.options.find((one) => one.type === 'new') }
    }

    it('is offered alongside notes that are named otherwise', async () => {
      const { option } = await create([{ title: 'Optics' }], 'See [[Quantum', vi.fn())

      expect(option).toBeDefined()
    })

    it('is offered where the note of that name is only a heading', async () => {
      const { option } = await create(
        [{ title: 'Quantum', heading: 'Intro' }],
        'See [[Quantum',
        vi.fn(),
      )

      expect(option).toBeDefined()
    })

    it('is not offered where a note of that name remains in other letters', async () => {
      const { option } = await create([{ title: 'ONTOLOGY' }], 'See [[ontology', vi.fn())

      expect(option).toBeUndefined()
    })

    it('is not offered where the query is empty', async () => {
      const { option } = await create([], 'See [[  ', vi.fn())

      expect(option).toBeUndefined()
    })

    it('is not offered where nothing can create it', async () => {
      const { option } = await create([], 'See [[Quantum', undefined)

      expect(option).toBeUndefined()
    })

    it('is named, described and ranked below the notes', async () => {
      const { option, result } = await create([{ title: 'Optics' }], 'See [[Quantum', vi.fn())

      expect(option).toMatchObject({
        label: 'Create "Quantum"',
        detail: 'new note',
        type: 'new',
        boost: -1,
      })
      expect(result?.options[0]?.boost).toBe(1)
    })

    it('is linked and created under the title as it was typed, trimmed', async () => {
      const createNote = vi.fn().mockResolvedValue('Quantum')
      const doc = 'See [[ Quantum '
      const { option, result } = await create([], doc, createNote)

      const chosen = choose(doc, doc.length, option, result!.from)

      expect(chosen.text).toBe('See [[Quantum]]')
      expect(chosen.caret).toBe('See [[Quantum]]'.length)
      expect(createNote).toHaveBeenCalledWith('Quantum')
    })
  })

  describe('the headings', () => {
    const headings = ['Introduction', 'Hamiltonian', 'Applications']

    const ask = async (doc: string, caret = doc.length) => {
      const lookup = vi.fn().mockResolvedValue(headings)
      const result = await createWikilinkSource({ search: vi.fn(), headings: lookup })(
        getContext(doc, caret),
      )
      return { lookup, result }
    }

    it('of the note are requested for under its title without the space around it', async () => {
      const { lookup } = await ask('See [[ Quantum Mechanics #Ham')

      expect(lookup).toHaveBeenCalledWith('Quantum Mechanics')
    })

    it('are those the trimmed words after the hash are part of', async () => {
      const { result } = await ask('See [[Quantum#  ham  ')

      expect(result?.options.map((one) => one.label)).toStrictEqual(['Quantum#Hamiltonian'])
    })

    it('of this note are offered where the hash opens the query', async () => {
      const { lookup, result } = await ask('See [[#Int')

      expect(lookup).toHaveBeenCalledWith('')
      expect(result?.options.map((one) => one.label)).toStrictEqual(['#Introduction'])
    })

    it('are described, typed and left unfiltered', async () => {
      const { result } = await ask('See [[Quantum#Ham')

      expect(result?.options[0]).toMatchObject({ detail: 'heading', type: 'heading' })
      expect(result?.filter).toBe(false)
    })

    it('start the replaced text after the two brackets', async () => {
      const doc = 'First line\nSee [[Quantum#Ham'

      const { result } = await ask(doc)

      expect(result?.from).toBe('First line\nSee '.length + 2)
    })

    it('are linked under the note and the heading, replacing what was typed', async () => {
      const doc = 'First line\nSee [[Quantum#Ham]] and more'
      const caret = 'First line\nSee [[Quantum#Ham'.length

      const { result } = await ask(doc, caret)

      const chosen = choose(doc, caret, result?.options[0], result!.from)
      expect(chosen.text).toBe('First line\nSee [[Quantum#Hamiltonian]] and more')
      expect(chosen.caret).toBe('First line\nSee [[Quantum#Hamiltonian]]'.length)
    })

    it('are linked under the heading alone where the hash opens the query', async () => {
      const doc = 'See [[#Int'

      const { result } = await ask(doc)

      expect(choose(doc, doc.length, result?.options[0], result!.from).text).toBe(
        'See [[#Introduction]]',
      )
    })

    it('are none where nothing is given to look them up', async () => {
      const result = await createWikilinkSource({ search: vi.fn() })(
        getContext('See [[Quantum#Ham', 17),
      )

      expect(result?.options).toStrictEqual([])
    })
  })
})

describe('wikilink completion without headings to look up', () => {
  it('offers none for a hash with nothing after it', async () => {
    const doc = 'See [[Quantum#'

    const result = await createWikilinkSource({ search: vi.fn() })(
      new CompletionContext(createState(doc, doc.length), doc.length, true),
    )

    expect(result?.options).toStrictEqual([])
  })
})

describe('the wikilink completion extension', () => {
  const open = async (search: (query: string) => Promise<readonly WikilinkOption[]>) => {
    const doc = 'See [[Rel'
    const state = EditorState.create({
      doc,
      selection: { anchor: doc.length },
      extensions: [createWikilinkCompletion({ search })],
    })
    const view = new EditorView({ state, parent: document.body })
    startCompletion(view)
    await vi.waitFor(() => expect(currentCompletions(view.state)).toHaveLength(1))
    return view
  }

  it('offers the notes the search finds when completion is started', async () => {
    const search = vi.fn().mockResolvedValue([{ title: 'Relativity' }])

    const view = await open(search)

    expect(search).toHaveBeenCalledWith('Rel')
    expect(currentCompletions(view.state)[0]?.label).toBe('Relativity')
    view.destroy()
  })

  it('draws the offered notes without icons', async () => {
    const view = await open(vi.fn().mockResolvedValue([{ title: 'Relativity' }]))

    await vi.waitFor(() =>
      expect(view.dom.querySelector('.cm-tooltip-autocomplete')).not.toBeNull(),
    )
    expect(view.dom.querySelector('.cm-completionIcon')).toBeNull()
    view.destroy()
  })
})
