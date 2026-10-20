import {
  ProblemKind as ProblemKinds,
  type Card as CardMessage,
  type Deck as DeckMessage,
  type Problem as ProblemMessage,
  type Stencil as StencilMessage,
} from '@numen/protocol'
import type {
  ProblemKind,
  DeckProblem,
  StencilSummary,
  VaultCard,
  VaultDeck,
  VaultStencil,
} from '../types'

/** What kind of problem it is, in the words the window uses. */
const kinds: Record<ProblemKinds, ProblemKind> = {
  [ProblemKinds.UNSPECIFIED]: 'unknown',
  [ProblemKinds.FIELD_DECLARED_TWICE]: 'fieldDeclaredTwice',
  [ProblemKinds.STENCIL_WITHOUT_FIELDS]: 'stencilWithoutFields',
  [ProblemKinds.FACE_MISSING_A_SIDE]: 'faceMissingASide',
  [ProblemKinds.PLACEHOLDER_UNDECLARED]: 'placeholderUndeclared',
  [ProblemKinds.CARD_WITHOUT_A_STENCIL]: 'cardWithoutAStencil',
  [ProblemKinds.STENCIL_IS_NOT_ONE]: 'stencilIsNotOne',
  [ProblemKinds.MARK_CARRIED_TWICE]: 'markCarriedTwice',
  [ProblemKinds.FIELD_WRITTEN_TWICE]: 'fieldWrittenTwice',
  [ProblemKinds.FIELD_NOT_RENAMED]: 'fieldNotRenamed',
}

/** One problem, with where it stands kept as a number or as nothing. */
export const deserializeDeckProblem = (one: ProblemMessage): DeckProblem => ({
  kind: kinds[one.kind],
  card: one.card ?? null,
  face: one.face ?? null,
  field: one.field,
  text: one.text,
})

/** One stencil of the list, kept as the plain value the window carries it as. */
export const deserializeStencilSummary = (one: {
  path: string
  title: string
  fields: string[]
}): StencilSummary => ({
  path: one.path,
  title: one.title,
  fields: one.fields,
})

export const deserializeCard = (one: CardMessage): VaultCard => ({
  mark: one.mark,
  sectionIndex: one.sectionIndex ?? null,
  heading: one.heading,
  stencilLink: one.stencilLink,
  stencilPath: one.stencilPath,
  preamble: one.preamble,
  values: one.values.map((value) => ({ field: value.field, text: value.text })),
})

/**
 * One card in the shape the schema carries it.
 */
export const serializeCardToWire = (one: VaultCard) => ({
  mark: one.mark,
  ...(one.sectionIndex === null ? {} : { sectionIndex: one.sectionIndex }),
  heading: one.heading,
  stencilLink: one.stencilLink,
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

/** One stencil as the vault reads it. */
export const deserializeStencil = (one: StencilMessage): VaultStencil => ({
  path: one.path,
  title: one.title,
  preamble: one.preamble,
  tail: one.tail,
  fields: one.fields,
  faces: one.faces.map((face) => ({
    name: face.name,
    preamble: face.preamble,
    front: face.front,
    back: face.back,
  })),
  problems: one.problems.map(deserializeDeckProblem),
})
