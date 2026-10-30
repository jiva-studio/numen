/**
 * What a card model tab holds, tested without a screen: what it reads, what it
 * writes back, and where its problems are shown.
 */
import { describe, expect, it, vi } from 'vitest'
import { asFailure, asValue } from '@numen/plugin-sdk'
import type { ErrorCode } from '../cards/wire/errors'
import type { Cards, VaultTemplate, DeckProblem, FieldRenameResult } from '../cards'
import { createMockBufferHost, createMockTabWindow } from '@numen/plugin-sdk/testing'
import { ERRORS } from '../cards/wire/errors'
import { CARD_MODEL, createCardModelTabs, type CardModelTabState } from './kind'
import { WORDS as words } from '../cards'

/** Waits for the tab's pending vault requests to resolve. */
const settle = () => new Promise((isDone) => setTimeout(isDone, 0))

const TEMPLATES: readonly VaultTemplate[] = [
  { name: 'Recognise', preamble: '', front: '{{Height}}', back: '**Height:** {{Height}}' },
]

/** A vault holding one card model, recording every write it receives. */
const vault = (
  answers: {
    error?: ErrorCode
    problems?: readonly DeckProblem[]
    isChanged?: boolean
    /** The result of renaming a field, when a test needs a different result. */
    renaming?: FieldRenameResult
    /** The vault is unreachable, and reading the card model fails. */
    isUnreachable?: boolean
    /** What a write of the card model fails for. */
    wrote?: ErrorCode
  } = {},
) => {
  const content: string[] = []
  /** Each field rename the tab requested from the vault. */
  const renameCalls: string[] = []
  let fields: readonly string[] = ['Height', 'Life span']
  let templates: readonly VaultTemplate[] = TEMPLATES

  const core: Cards = {
    cardModels: async () => ({ cardModels: [], total: 0 }),
    createDeck: async (title) => ({ ok: true, value: { path: `${title}.md` } }),
    createCardModel: async (title) => ({ ok: true, value: { path: `${title}.md` } }),
    // The vault writes the name wherever it occurs: in the fields and in the
    // braces of every template.
    renameField: async (path, field, observed) => {
      const { from, to } = field
      renameCalls.push(`${path} ${from} ${to} ${observed ?? '—'}`)
      if (answers.renaming) return answers.renaming
      fields = fields.map((one) => (one === from ? to : one))
      const rewriteBraces = (text: string) => text.split(`{{${from}}}`).join(`{{${to}}}`)
      templates = templates.map((template) => ({
        ...template,
        front: rewriteBraces(template.front),
        back: rewriteBraces(template.back),
      }))
      return asValue({ decks: [], cards: 0, notWritten: [], at: 'renamed' })
    },
    readDeck: async () => asFailure({ code: 'missing' as const, bound: 0 }),
    writeDeck: async () => asValue({ at: '' }),
    readCardModel: async (path) => {
      if (answers.isUnreachable) throw new Error('out of reach')
      if (answers.error) return asFailure(answers.error)
      return asValue({
        cardModel: {
          path,
          title: 'Animal',
          fields,
          preamble: '',
          templates,
          tail: '',
          problems: answers.problems ?? [],
        },
        at: 'read',
      })
    },
    writeCardModel: async (path, cardModel, _seen) => {
      content.push(
        `${path} ${cardModel.fields.join(', ') || '—'} | ${cardModel.templates.map((one) => one.back).join(' ')}`,
      )
      if (answers.wrote) return asFailure(answers.wrote)
      if (answers.isChanged) return asFailure('changed' as const)
      fields = cardModel.fields
      templates = cardModel.templates
      return asValue({ at: 'written' })
    },
  }

  return {
    core,
    written: content,
    writtenFiles: content,
    renamed: renameCalls,
    /** The file written from somewhere else, which the next read answers with. */
    setTemplates: (next: readonly VaultTemplate[]) => {
      templates = next
    },
  }
}

/** A window with one card model open on a file, and what that tab holds. */
const open = async (answers: Parameters<typeof vault>[0] = {}, path = 'Animal.md') => {
  const one = vault(answers)
  const windowTabs = createMockTabWindow()
  const buffers = createMockBufferHost()
  /** Everything the window was given to say about this card model. */
  const messages: string[] = []
  const cardModels = createCardModelTabs(one.core, {
    buffers,
    closeTab: windowTabs.finishClose,
    writeMessage: (text) => {
      messages.push(text)
    },
  })
  windowTabs.register(cardModels.kind)
  const id = await windowTabs.openTab(CARD_MODEL, path)
  await settle()
  const tab = windowTabs.getTabState<CardModelTabState>(id) as CardModelTabState
  return { ...one, windowTabs, buffers, cardModels, id, tab, messages }
}

describe('a card model opened', () => {
  it('draws the fields in the order a person is requested for them', async () => {
    const { tab } = await open()

    expect(tab.cardModel.value.fields).toStrictEqual(['Height', 'Life span'])
  })

  it('gives every template an identity, which the file carries none of', async () => {
    const { tab } = await open()

    expect(tab.cardModel.value.templates[0]?.id).toBeTruthy()
    expect(tab.cardModel.value.templates[0]?.name).toBe('Recognise')
  })

  it('is called what the file is called', async () => {
    const { cardModels, tab } = await open()

    expect(cardModels.kind.getTitle?.(tab)).toBe('Animal')
  })
})

describe('a field renamed in a card model', () => {
  it('is the vault that renames it, presenting the file the tab read', async () => {
    const { tab, renamed } = await open()

    tab.renameField('Height', 'Shoulder')
    await settle()

    expect(renamed).toStrictEqual(['Animal.md Height Shoulder read'])
  })

  it('writes the card model from the tab nowhere, so nothing goes over the vault', async () => {
    const { cardModels, tab, writtenFiles } = await open()

    tab.renameField('Height', 'Shoulder')
    await settle()
    await cardModels.store.flush()

    expect(writtenFiles).toStrictEqual([])
  })

  it('shows the fields and the braces as the vault left them', async () => {
    const { tab } = await open()

    tab.renameField('Height', 'Shoulder')
    await settle()
    await settle()

    expect(tab.cardModel.value.fields).toStrictEqual(['Shoulder', 'Life span'])
    expect(tab.cardModel.value.templates[0]?.back).toBe('**Height:** {{Shoulder}}')
  })

  it('asks for nothing where the name is the one the field carries', async () => {
    const { tab, renamed } = await open()

    tab.renameField('Height', 'Height')
    tab.renameField('Height', '')
    await settle()

    expect(renamed).toStrictEqual([])
  })

  it('says how far the new name reached', async () => {
    const { tab, messages } = await open({
      renaming: asValue({
        decks: ['Animals.md', 'More.md'],
        cards: 3,
        notWritten: [],
        at: 'renamed',
      }),
    })

    tab.renameField('Height', 'Shoulder')
    await settle()

    expect(messages).toContain(words.renamed(3, 2))
  })

  it('says which decks keep the old heading, which nothing else would tell', async () => {
    const { tab, messages } = await open({
      renaming: asValue({
        decks: ['Animals.md'],
        cards: 1,
        notWritten: [{ path: 'Broken.md', text: 'the frontmatter cannot be read' }],
        at: 'renamed',
      }),
    })

    tab.renameField('Height', 'Shoulder')
    await settle()

    expect(messages).toContain(words.notWritten(['Broken.md']))
  })

  it('says the error, and says nothing of decks reached, where none was', async () => {
    const { tab, messages } = await open({
      renaming: asFailure('notACardModel' as const),
    })

    tab.renameField('Height', 'Shoulder')
    await settle()

    expect(messages).toStrictEqual([ERRORS.notACardModel])
  })

  it('says nothing was renamed where the file moved past the card model that was read', async () => {
    const { tab, messages } = await open({
      renaming: asFailure('changed' as const),
    })

    tab.renameField('Height', 'Shoulder')
    await settle()

    expect(messages).toStrictEqual([words.notRenamed])
  })

  it('writes nothing where nothing was touched', async () => {
    const { cardModels, writtenFiles } = await open()

    await cardModels.store.flush()

    expect(writtenFiles).toStrictEqual([])
  })
})

describe('a field carried in a card model', () => {
  it('lands where it was let go', async () => {
    const { tab } = await open()

    tab.moveField('Life span', null)

    expect(tab.cardModel.value.fields).toStrictEqual(['Height', 'Life span'])
  })

  it('leaves the first field first, wherever it was let go', async () => {
    const { tab } = await open()

    tab.moveField('Height', null)

    expect(tab.cardModel.value.fields).toStrictEqual(['Height', 'Life span'])
  })

  it('lands nothing above the first field', async () => {
    const { tab } = await open()

    tab.addField('Weight')
    tab.moveField('Weight', 'Height')

    expect(tab.cardModel.value.fields).toStrictEqual(['Height', 'Life span', 'Weight'])
  })
})

describe('a card model whose file moved past what was read', () => {
  it('is stale once the write arrives back saying the file changed', async () => {
    const { cardModels, tab } = await open({ isChanged: true })

    tab.addField('Weight')
    await cardModels.store.flush()

    expect(tab.note.value.state).toBe('stale')
  })

  it('keeps what the person wrote when they say so', async () => {
    const { cardModels, tab, writtenFiles } = await open({ isChanged: true })

    tab.addField('Weight')
    await cardModels.store.flush()
    tab.keepMine()
    await settle()

    expect(writtenFiles).toHaveLength(2)
  })
})

describe('a card model the vault rejected', () => {
  it('says the note is not a card model where that is what it is', async () => {
    const { tab } = await open({ error: 'notACardModel' })

    expect(tab.errorMessage.value).toBe(words.notACardModel)
    expect(tab.cardModel.value.fields).toStrictEqual([])
  })

  it('says nothing where the card model was read', async () => {
    const { tab } = await open()

    expect(tab.errorMessage.value).toBe('')
  })

  it('says the vault could not be reached, where the read reached nothing', async () => {
    const { tab } = await open({ isUnreachable: true })

    expect(tab.errorMessage.value).toBe(words.unreachable)
  })

  it('says the file could not be written, where that is what failed', async () => {
    const { cardModels, tab } = await open({ wrote: 'unreadable' })

    tab.addField('Weight')
    await cardModels.store.flushPending(cardModels.store.getOpenIds()[0] ?? '')

    expect(tab.errorMessage.value).toBe(words.notSaved)
  })
})

describe('what is wrong with a card model', () => {
  it('remains against the template it was read against', async () => {
    const { tab } = await open({
      problems: [
        { kind: 'templateMissingASide', card: null, template: 0, field: '', text: 'no back' },
      ],
    })

    const template = tab.cardModel.value.templates[0]?.id ?? ''
    expect(tab.marks.value.at.get(template)).toStrictEqual(['no back'])
  })

  it('remains against the field it names at its current position against no template', async () => {
    const { tab } = await open({
      problems: [
        {
          kind: 'fieldDeclaredTwice',
          card: null,
          template: null,
          field: 'Height',
          text: 'declared twice',
        },
      ],
    })

    expect(tab.marks.value.fields.get('Height')).toStrictEqual(['declared twice'])
    expect(tab.marks.value.at.size).toBe(0)
  })

  it('is nothing at all where the vault reported none', async () => {
    const { tab } = await open()

    expect(tab.marks.value.at.size).toBe(0)
    expect(tab.marks.value.fields.size).toBe(0)
    expect(tab.marks.value.whole).toStrictEqual([])
  })
})

describe('a card model read again under the window', () => {
  // The file has a problem that applies to one template, so a mark that
  // disappears is observable.
  const sideless: DeckProblem = {
    kind: 'templateMissingASide',
    card: null,
    template: 0,
    field: '',
    text: 'no back',
  }

  it('leaves what the editor is drawing standing, where the file reads the same', async () => {
    const one = await open({ problems: [sideless] })
    const was = { cardModel: one.tab.cardModel.value, marks: one.tab.marks.value }

    one.cardModels.applyPathChanges(['Animal.md'])
    await settle()

    // Each of them the same thing, and not merely a thing that reads the same:
    // a field under the keyboard is redrawn by anything else.
    expect(one.tab.cardModel.value).toBe(was.cardModel)
    expect(one.tab.marks.value).toBe(was.marks)
  })

  it('leaves each mark standing on the template the editor is drawing', async () => {
    const one = await open({ problems: [sideless] })
    // The template the editor is drawing, under the identity it was drawn with.
    const template = one.tab.cardModel.value.templates[0]?.id ?? ''

    one.cardModels.applyPathChanges(['Animal.md'])
    await settle()

    expect(one.tab.marks.value.at.get(template)).toStrictEqual(['no back'])
  })

  it('draws the file again where it was written from somewhere else', async () => {
    const one = await open()
    const was = one.tab.cardModel.value

    one.setTemplates([{ name: 'Recall', preamble: '', front: '{{Height}}', back: '{{Life span}}' }])
    one.cardModels.applyPathChanges(['Animal.md'])
    await settle()

    expect(one.tab.cardModel.value.templates.map((template) => template.name)).toStrictEqual([
      'Recall',
    ])
    expect(one.tab.cardModel.value).not.toBe(was)
  })
})

describe('a card model renamed under the window', () => {
  it('is the tab it has when it is requested for at the name it now carries', async () => {
    const one = await open()

    one.cardModels.applyPathChanges(['Beast.md'], [{ from: 'Animal.md', to: 'Beast.md' }])
    await settle()
    await one.windowTabs.openTab(CARD_MODEL, 'Beast.md')
    await settle()

    expect(one.cardModels.store.getOpenIds()).toHaveLength(1)
    expect(one.windowTabs.each(CARD_MODEL)).toHaveLength(1)
  })

  it('is called what the file was called, before the name it went to is read', async () => {
    const one = await open()

    one.cardModels.applyPathChanges(['Beast.md'], [{ from: 'Animal.md', to: 'Beast.md' }])

    expect(one.cardModels.getTitle('Beast.md')).toBe('Animal')
  })
})

describe('a card model whose tab has gone', () => {
  it('is nothing the window still says a word about', async () => {
    const one = await open()
    expect(one.cardModels.getTitle('Animal.md')).toBe('Animal')

    one.windowTabs.requestClose(one.id)
    await settle()

    expect(one.cardModels.getTitle('Animal.md')).toBe('Animal.md')
  })
})

/** A card model whose write failed because its file moved: the tab is stale. */
const openStale = async () => {
  const one = await open({ isChanged: true })
  one.tab.addField('Weight')
  await one.cardModels.store.flush()
  return one
}

describe('a card model tab as the window lists it', () => {
  it('is the identity it opened under', async () => {
    const { cardModels } = await open()

    expect(cardModels.kind.identity?.('anything')).toBe('anything')
  })

  it('names the file it remains at and what it is called', async () => {
    const { cardModels, tab } = await open()

    expect(cardModels.kind.getTarget?.(tab)).toStrictEqual({ path: 'Animal.md', title: 'Animal' })
  })

  it('names the file it remains at where it is requested what is open', async () => {
    const { cardModels, tab } = await open()

    expect(cardModels.kind.getOpenTab?.(tab)).toStrictEqual({ path: 'Animal.md' })
  })

  it('carries no mark while its file is read, and the mark of its state once stale', async () => {
    const clean = await open()
    expect(clean.cardModels.kind.getMark?.(clean.tab)).toBeUndefined()

    const staleSession = await openStale()
    expect(staleSession.cardModels.kind.getMark?.(staleSession.tab)).toBe('stale')
  })

  it('closes card model and suppresses window prompt', async () => {
    const { cardModels, tab } = await open()

    expect(cardModels.kind.onClose?.(tab, 'no such tab')).toBe(false)
  })
})

describe('what the window asks of a card model tab', () => {
  it('is whether the file reads as it did, or moved past it', async () => {
    const clean = await open()
    expect(clean.cardModels.kind.isAsking?.(clean.tab)).toBe(false)

    const staleSession = await openStale()
    expect(staleSession.cardModels.kind.isAsking?.(staleSession.tab)).toBe(true)
  })

  it('is to write what is unwritten out, and answer once it is', async () => {
    const one = await open()
    one.tab.addField('Weight')

    await one.cardModels.kind.flushPending?.(one.tab)

    expect(one.written).toStrictEqual([
      'Animal.md Height, Life span, Weight | **Height:** {{Height}}',
    ])
  })

  it('is to be written out with the other open files when the window quits', async () => {
    const one = await open()
    one.tab.addField('Weight')

    await one.buffers.flush()

    expect(one.written).toStrictEqual([
      'Animal.md Height, Life span, Weight | **Height:** {{Height}}',
    ])
  })

  it('is to make a card model of the name and the folder, carrying the first field', async () => {
    const one = await open()
    const createdItem = vi.spyOn(one.core, 'createCardModel')

    await one.cardModels.kind.createFile?.('plants', 'Plant')

    expect(createdItem).toHaveBeenCalledWith('Plant', 'plants', [words.newField])
  })
})

describe('a card model tab closed', () => {
  it('stays open, still called what it was, where the file moved past what it holds', async () => {
    const one = await openStale()

    one.tab.close(one.id)
    await settle()

    expect(one.windowTabs.each(CARD_MODEL)).toHaveLength(1)
    expect(one.cardModels.store.getOpenIds()).toStrictEqual([one.tab.id])
    expect(one.cardModels.getTitle('Animal.md')).toBe('Animal')
  })

  it('is read afresh where the file is opened again', async () => {
    const one = await open()
    const was = one.tab.cardModel.value.templates[0]?.id

    one.tab.close(one.id)
    await settle()
    const id = await one.windowTabs.openTab(CARD_MODEL, 'Animal.md')
    await settle()
    const tab = one.windowTabs.getTabState<CardModelTabState>(id) as CardModelTabState

    expect(tab.cardModel.value.templates[0]?.id).not.toBe(was)
  })

  it('lets go of its title while a tab relies on another file', async () => {
    const one = await open()
    await one.windowTabs.openTab(CARD_MODEL, 'Plant.md')
    await settle()

    one.windowTabs.requestClose('card-model:Animal.md')
    await settle()

    expect(one.cardModels.getTitle('Animal.md')).toBe('Animal.md')
  })

  it('leaves the title with the other tab standing on the same file', async () => {
    const one = await open()
    await one.windowTabs.openTab(CARD_MODEL, 'Beast.md')
    await settle()
    one.cardModels.applyPathChanges(['Animal.md'], [{ from: 'Beast.md', to: 'Animal.md' }])
    await settle()

    one.tab.close(one.id)
    await settle()

    expect(one.cardModels.getTitle('Animal.md')).toBe('Animal')
  })
})

describe('a card model taken from its file', () => {
  it('shows the file as it now reads, and is no longer stale', async () => {
    const one = await openStale()

    one.tab.takeFile()
    await settle()

    expect(one.tab.note.value.state).not.toBe('stale')
    expect(one.tab.cardModel.value.fields).toStrictEqual(['Height', 'Life span'])
  })
})

describe('a card model opened from a file', () => {
  it('remains at one tab however many times it is requested for before it is open', async () => {
    const one = await open()

    await one.windowTabs.openTab(CARD_MODEL, 'Plant.md')
    await one.windowTabs.openTab(CARD_MODEL, 'Plant.md')
    await settle()

    expect(one.cardModels.store.getOpenIds()).toHaveLength(2)
    expect(one.windowTabs.each(CARD_MODEL)).toHaveLength(2)
  })

  it('remains at the file again once its tab was closed and it is opened again', async () => {
    const one = await open()
    const first = await one.windowTabs.openTab(CARD_MODEL, 'Plant.md')
    await settle()
    one.windowTabs.requestClose(first)
    await settle()

    const again = await one.windowTabs.openTab(CARD_MODEL, 'Plant.md')
    await settle()

    expect(one.cardModels.store.getPath(again)).toBe('Plant.md')
  })
})
