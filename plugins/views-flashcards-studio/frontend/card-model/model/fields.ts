/**
 * Buffer parsing, marks caching, and field/template editing actions for card model tabs.
 */
import type { Half } from '../../ui/cards'
import {
  addTemplate,
  addField,
  moveTemplate,
  moveField,
  removeTemplate,
  removeField,
  renameTemplate,
  areCardModelsEqual,
  serializeCardModel,
  parseCardModel,
  writeTemplate,
  type BufferCardModel,
} from '../lib/card-model'
import { createMarks, areMarksEqual, type Marks, type DeckProblem } from '../../cards'

export interface CardModelFieldsDeps {
  getBody: (id: string) => string
  onTyped: (id: string, body: string) => void
  getProblems: (id: string) => readonly DeckProblem[]
  renameFieldOnWire: (id: string, field: string, name: string) => void
}

function createCardModelActions(
  id: string,
  deps: {
    getCardModel: (id: string) => BufferCardModel
    applyCardModel: (id: string, cardModel: BufferCardModel) => void
    renameFieldOnWire: (id: string, field: string, name: string) => void
  },
) {
  const { getCardModel, applyCardModel, renameFieldOnWire } = deps
  return {
    addField: (name: string) => applyCardModel(id, addField(getCardModel(id), name)),
    renameField: (field: string, name: string) => void renameFieldOnWire(id, field, name),
    removeField: (field: string) => applyCardModel(id, removeField(getCardModel(id), field)),
    moveField: (field: string, target: string | null) =>
      applyCardModel(id, moveField(getCardModel(id), field, target)),
    addTemplate: (name: string) => applyCardModel(id, addTemplate(getCardModel(id), name)),
    renameTemplate: (template: string, name: string) =>
      applyCardModel(id, renameTemplate(getCardModel(id), template, name)),
    removeTemplate: (template: string) =>
      applyCardModel(id, removeTemplate(getCardModel(id), template)),
    moveTemplate: (template: string, target: string | null) =>
      applyCardModel(id, moveTemplate(getCardModel(id), template, target)),
    writeTemplateHalf: (template: string, half: Half, text: string) =>
      applyCardModel(id, writeTemplate(getCardModel(id), template, half, text)),
  }
}

function resolveCardModelFieldsDeps(
  args:
    | [deps: CardModelFieldsDeps]
    | [
        getBody: (id: string) => string,
        onTyped: (id: string, body: string) => void,
        getProblems: (id: string) => readonly DeckProblem[],
        renameFieldOnWire: (id: string, field: string, name: string) => void,
      ],
): CardModelFieldsDeps {
  return args.length === 1
    ? args[0]
    : {
        getBody: args[0],
        onTyped: args[1],
        getProblems: args[2],
        renameFieldOnWire: args[3],
      }
}

export function createCardModelFields(
  ...args:
    | [deps: CardModelFieldsDeps]
    | [
        getBody: (id: string) => string,
        onTyped: (id: string, body: string) => void,
        getProblems: (id: string) => readonly DeckProblem[],
        renameFieldOnWire: (id: string, field: string, name: string) => void,
      ]
) {
  const { getBody, onTyped, getProblems, renameFieldOnWire } = resolveCardModelFieldsDeps(args)
  const parsed = new Map<string, { body: string; cardModel: BufferCardModel }>()
  const marked = new Map<string, { problems: readonly DeckProblem[]; marks: Marks }>()

  const getCardModel = (id: string): BufferCardModel => {
    const body = getBody(id)
    const target = parsed.get(id)
    const readItem = parseCardModel(body)
    const cardModel =
      target && areCardModelsEqual(target.cardModel, readItem) ? target.cardModel : readItem
    parsed.set(id, { body, cardModel })
    return cardModel
  }

  const getMarks = (id: string): Marks => {
    const problems = getProblems(id)
    const target = marked.get(id)
    if (target && target.problems === problems) return target.marks
    const readItem = createMarks(
      problems,
      [],
      getCardModel(id).templates.map((template) => template.id),
    )
    const marks = target && areMarksEqual(target.marks, readItem) ? target.marks : readItem
    marked.set(id, { problems, marks })
    return marks
  }

  const applyCardModel = (id: string, cardModel: BufferCardModel): void => {
    const body = serializeCardModel(cardModel)
    parsed.set(id, { body, cardModel })
    onTyped(id, body)
  }

  const forget = (id: string): void => {
    parsed.delete(id)
    marked.delete(id)
  }

  return {
    getCardModel,
    getMarks,
    applyCardModel,
    forget,
    createActions: (id: string) =>
      createCardModelActions(id, { getCardModel, applyCardModel, renameFieldOnWire }),
  }
}
