/**
 * What a card model is while the window holds it, and what each gesture makes of
 * it.
 *
 * The components take names and text and hand them back under identities they
 * were given. Minting those identities, and turning what a person did into the
 * fields and the templates a file is written from, is here.
 */
import { orderNames, reorderFields, type InsertionPoint } from '../../ui/cards'
import {
  type VaultTemplate,
  type VaultCardModel,
  generateId,
  type IdMaker,
  type Surrounds,
} from '../../cards'

/** One template as the window holds it: what the file says, under an identity of its own. */
export interface BufferTemplate {
  /** The identity the window addresses it by, minted at every reading. */
  readonly id: string
  readonly name: string
  /** The prose between the template's heading and its first side. */
  readonly preamble: string
  readonly front: string
  readonly back: string
}

/** A card model as the window holds it: its fields, and its templates under identities. */
export interface BufferCardModel extends Surrounds {
  readonly fields: readonly string[]
  readonly templates: readonly BufferTemplate[]
}

/** A card model that is empty and shows nothing. */
export const NO_CARD_MODEL: BufferCardModel = { fields: [], preamble: '', templates: [], tail: '' }

/** A card model as the vault read it, each template under an identity this window mints. */
export const createBufferCardModel = (
  readItem: VaultCardModel,
  generateTemplateId: IdMaker = generateId,
): BufferCardModel => ({
  fields: readItem.fields,
  preamble: readItem.preamble,
  templates: readItem.templates.map((template) => ({
    id: generateTemplateId(),
    name: template.name,
    preamble: template.preamble,
    front: template.front,
    back: template.back,
  })),
  tail: readItem.tail,
})

/**
 * A card model as one string, which is what the tab holding it is dirty against.
 * The parts are written in a settled order, so a card model that came back
 * unchanged reads as the string it went in as.
 */
export const serializeCardModel = (cardModel: BufferCardModel): string =>
  JSON.stringify({
    fields: cardModel.fields,
    preamble: cardModel.preamble,
    tail: cardModel.tail,
    templates: cardModel.templates.map((template) => ({
      id: template.id,
      name: template.name,
      preamble: template.preamble,
      front: template.front,
      back: template.back,
    })),
  })

/** The card model a string stands for. A string holding nothing names no field. */
export const parseCardModel = (body: string): BufferCardModel =>
  body ? (JSON.parse(body) as BufferCardModel) : NO_CARD_MODEL

/** The templates of a card model, in the shape the vault takes them. */
export const createVaultTemplates = (cardModel: BufferCardModel): readonly VaultTemplate[] =>
  cardModel.templates.map(({ name, preamble, front, back }) => ({ name, preamble, front, back }))

/** Whether two card models read the same, the identities left out the same way. */
export const areCardModelsEqual = (one: BufferCardModel, other: BufferCardModel): boolean =>
  one.preamble === other.preamble &&
  one.tail === other.tail &&
  JSON.stringify(one.fields) === JSON.stringify(other.fields) &&
  JSON.stringify(createVaultTemplates(one)) === JSON.stringify(createVaultTemplates(other))

/** Name a field at the end of the order. */
export const addField = (cardModel: BufferCardModel, name: string): BufferCardModel => ({
  ...cardModel,
  fields: [...cardModel.fields, name],
})

/** Take a field out of the card model. What the templates stand in its braces stays. */
export const removeField = (cardModel: BufferCardModel, field: string): BufferCardModel => ({
  ...cardModel,
  fields: cardModel.fields.filter((one) => one !== field),
})

/**
 * Move a field to another place in the order. The first field names every card
 * the card model cuts, so it stays first and nothing lands above it.
 */
export const moveField = (
  cardModel: BufferCardModel,
  field: string,
  target: InsertionPoint,
): BufferCardModel => ({
  ...cardModel,
  fields: reorderFields(cardModel.fields, field, target),
})

/**
 * Move a template to another place in the order: before another, or at the end.
 * The order of the templates is the order a card's repetitions are taken from it,
 * and nothing among them is fixed.
 */
export const moveTemplate = (
  cardModel: BufferCardModel,
  id: string,
  target: InsertionPoint,
): BufferCardModel => {
  const order = orderNames(
    cardModel.templates.map((template) => template.id),
    id,
    target,
  )
  const templates = [...cardModel.templates].sort(
    (a, b) => order.indexOf(a.id) - order.indexOf(b.id),
  )
  return { ...cardModel, templates }
}

/** Add a template at the end, with both its halves empty. */
export const addTemplate = (
  cardModel: BufferCardModel,
  name: string,
  generateTemplateId: IdMaker = generateId,
): BufferCardModel => ({
  ...cardModel,
  templates: [
    ...cardModel.templates,
    { id: generateTemplateId(), name, preamble: '', front: '', back: '' },
  ],
})

/** Put a template under another name. */
export const renameTemplate = (
  cardModel: BufferCardModel,
  id: string,
  name: string,
): BufferCardModel => ({
  ...cardModel,
  templates: cardModel.templates.map((template) =>
    template.id === id ? { ...template, name } : template,
  ),
})

/** Take a template out of the card model. */
export const removeTemplate = (cardModel: BufferCardModel, id: string): BufferCardModel => ({
  ...cardModel,
  templates: cardModel.templates.filter((template) => template.id !== id),
})

/** Write one half of one template. */
export const writeTemplate = (
  cardModel: BufferCardModel,
  ...[id, half, text]: [id: string, half: 'front' | 'back', text: string]
): BufferCardModel => ({
  ...cardModel,
  templates: cardModel.templates.map((template) =>
    template.id === id ? { ...template, [half]: text } : template,
  ),
})
