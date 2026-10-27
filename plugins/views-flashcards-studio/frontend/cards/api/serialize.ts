import {
  ProblemKind as ProblemKinds,
  type Card as CardMessage,
  type Deck as DeckMessage,
  type Problem as ProblemMessage,
  type CardModel as CardModelMessage,
} from '../../gen/numen/flashcards/v1/cards_pb.js'
import type {
  ProblemKind,
  DeckProblem,
  CardModelSummary,
  VaultCard,
  VaultDeck,
  VaultCardModel,
} from '../types'

/** What kind of problem it is, in the words the window uses. */
const kinds: Record<ProblemKinds, ProblemKind> = {
  [ProblemKinds.UNSPECIFIED]: 'unknown',
  [ProblemKinds.FIELD_DECLARED_TWICE]: 'fieldDeclaredTwice',
  [ProblemKinds.CARD_MODEL_WITHOUT_FIELDS]: 'cardModelWithoutFields',
  [ProblemKinds.TEMPLATE_MISSING_A_SIDE]: 'templateMissingASide',
  [ProblemKinds.PLACEHOLDER_UNDECLARED]: 'placeholderUndeclared',
  [ProblemKinds.CARD_WITHOUT_A_CARD_MODEL]: 'cardWithoutACardModel',
  [ProblemKinds.CARD_MODEL_IS_NOT_ONE]: 'cardModelIsNotOne',
  [ProblemKinds.CARD_ID_CARRIED_TWICE]: 'cardIdCarriedTwice',
  [ProblemKinds.FIELD_WRITTEN_TWICE]: 'fieldWrittenTwice',
  [ProblemKinds.FIELD_NOT_RENAMED]: 'fieldNotRenamed',
}

/** One problem, with at its current position kept as a number or as nothing. */
export const deserializeDeckProblem = (one: ProblemMessage): DeckProblem => ({
  kind: kinds[one.kind],
  card: one.card ?? null,
  template: one.template ?? null,
  field: one.field,
  text: one.text,
})

/** One card model of the list, kept as the plain value the window carries it as. */
export const deserializeCardModelSummary = (one: {
  path: string
  title: string
  fields: string[]
}): CardModelSummary => ({
  path: one.path,
  title: one.title,
  fields: one.fields,
})

export const deserializeCard = (one: CardMessage): VaultCard => ({
  cardId: one.cardId,
  sectionIndex: one.sectionIndex ?? null,
  heading: one.heading,
  cardModelLink: one.cardModelLink,
  cardModelPath: one.cardModelPath,
  preamble: one.preamble,
  values: one.values.map((value) => ({ field: value.field, text: value.text })),
})

/**
 * One card in the shape the schema carries it.
 */
export const serializeCardToWire = (one: VaultCard) => ({
  cardId: one.cardId,
  ...(one.sectionIndex === null ? {} : { sectionIndex: one.sectionIndex }),
  heading: one.heading,
  cardModelLink: one.cardModelLink,
  preamble: one.preamble,
  values: one.values.map((value) => ({ field: value.field, text: value.text })),
})

/** A deck as the window carries it. */
export const deserializeDeck = (one: DeckMessage): VaultDeck => ({
  path: one.path,
  title: one.title,
  preamble: one.preamble,
  cards: one.cards.map(deserializeCard),
  sections: one.sections.map((section) => ({ name: section.name, preamble: section.preamble })),
  tail: one.tail,
  problems: one.problems.map(deserializeDeckProblem),
})

/** One card model as the vault reads it. */
export const deserializeCardModel = (one: CardModelMessage): VaultCardModel => ({
  path: one.path,
  title: one.title,
  preamble: one.preamble,
  tail: one.tail,
  fields: one.fields,
  templates: one.templates.map((template) => ({
    name: template.name,
    preamble: template.preamble,
    front: template.front,
    back: template.back,
  })),
  problems: one.problems.map(deserializeDeckProblem),
})
