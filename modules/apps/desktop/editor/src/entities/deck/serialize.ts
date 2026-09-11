import { Fault as Faults } from '@numen/protocol'
import type {
  Card as CardMessage,
  Deck as DeckMessage,
  Problem as ProblemMessage,
  Stencil as StencilMessage,
} from '@numen/protocol'
import type {
  Fault,
  DeckProblem,
  StencilSummary,
  VaultCard,
  VaultDeck,
  VaultFace,
  VaultStencil,
} from './types'

/** What a problem is, in the words the window uses. */
const faulted: Record<Faults, Fault> = {
  [Faults.UNSPECIFIED]: 'unknown',
  [Faults.FIELD_DECLARED_TWICE]: 'fieldDeclaredTwice',
  [Faults.STENCIL_WITHOUT_FIELDS]: 'stencilWithoutFields',
  [Faults.FACE_MISSING_A_SIDE]: 'faceMissingASide',
  [Faults.PLACEHOLDER_UNDECLARED]: 'placeholderUndeclared',
  [Faults.CARD_WITHOUT_A_STENCIL]: 'cardWithoutAStencil',
  [Faults.STENCIL_IS_NOT_ONE]: 'stencilIsNotOne',
  [Faults.MARK_CARRIED_TWICE]: 'markCarriedTwice',
  [Faults.FIELD_WRITTEN_TWICE]: 'fieldWrittenTwice',
  [Faults.FIELD_NOT_RENAMED]: 'fieldNotRenamed',
}

/** One problem, with where it stands kept as a number or as nothing. */
export const deserializeDeckProblem = (one: ProblemMessage): DeckProblem => ({
  fault: faulted[one.fault],
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
