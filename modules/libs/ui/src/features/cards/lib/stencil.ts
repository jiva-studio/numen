/**
 * What a stencil is, as plain values: the slots it names, the faces that show
 * them, and the rows and blocks and panes those are drawn as. What any of it
 * means is the caller's. No DOM, no measurement, no clock.
 */

import type { FieldValue } from './card'
import { renderPreview, strayIn } from './fill'
import {
  getDeclaredFields,
  HALVES,
  createSealedMap,
  type Problems,
  type Half,
  type Objection,
  type HeadingObjection,
} from './order'

/** One way a stencil shows a card. */
export interface StencilFace {
  readonly id: string
  readonly name: string
  readonly front: string
  readonly back: string
}

/** The words a stencil is drawn with, declared once. */
export interface StencilWords {
  readonly fields: string
  readonly faces: string
  readonly front: string
  readonly back: string
  readonly preview: string
  readonly addField: string
  readonly addFace: string
  readonly remove: string
  readonly drag: string
  readonly insert: string
  readonly noFields: string
  readonly noFaces: string
  readonly fieldStem: string
  readonly faceStem: string
  /** What is said of the first field, which is not moved and not removed. */
  readonly pinned: string
  /** What is said of the slots a face names that the fields do not. */
  readonly stray: (fields: readonly string[]) => string
  /** What is said of a field's name that cannot be used. */
  readonly objection: (objection: Objection) => string
  /** What is said of a face's name that cannot be used. */
  readonly faceObjection: (objection: HeadingObjection) => string
  /** What a list of things wrong is called to a reader. */
  readonly wrong: string
}

export const STENCIL_WORDS: StencilWords = {
  fields: 'Fields',
  faces: 'Faces',
  front: 'Front',
  back: 'Back',
  preview: 'Preview',
  addField: 'Add a field',
  addFace: 'Add a face',
  remove: 'Remove',
  drag: 'Reorder',
  insert: 'Insert',
  noFields: 'No fields yet',
  noFaces: 'No faces yet',
  fieldStem: 'Field',
  faceStem: 'Face',
  pinned: 'The first field names every card, and stays first',
  stray: (fields) => `Not a field: ${fields.join(', ')}`,
  objection: (objection) =>
    objection === 'blank'
      ? 'A field needs a name'
      : objection === 'taken'
        ? 'That name is taken'
        : 'A name cannot hold a brace',
  faceObjection: (objection) =>
    objection === 'blank' ? 'A face needs a name' : 'That name is taken',
  wrong: 'What is wrong',
}

/**
 * What the caller found wrong with a stencil. A face's stands beside its name
 * and a field's stands under that field's row.
 */
export interface StencilWrong {
  /** What is wrong with each face, under the identity it was drawn by. */
  readonly at: Problems
  /** What is wrong with each field, under the name it is declared by. */
  readonly fields: Problems
}

/** Nothing wrong with any face and nothing wrong with any field. */
export const NOTHING_AMISS: StencilWrong = Object.freeze({
  at: createSealedMap<string, readonly string[]>(),
  fields: createSealedMap<string, readonly string[]>(),
})

/** One field of a stencil, as its row is drawn. */
export interface FieldRow {
  readonly field: string
  /** Where it stands, counting from one, which is what it is announced as. */
  readonly at: number
  /** How many fields stand with it. */
  readonly of: number
  /** It stands first, so it is what a card cut by this stencil is named by. */
  readonly names: boolean
  /** It is on its way somewhere else in the order. */
  readonly dragged: boolean
}

/** The rows a stencil's fields are drawn as, one to a field. */
export function fieldRows(
  fields: readonly string[],
  dragField: string | null,
): readonly FieldRow[] {
  const stood = getDeclaredFields(fields)
  return stood.map((field, index) => ({
    field,
    at: index + 1,
    of: stood.length,
    names: index === 0,
    dragged: field === dragField,
  }))
}

/** One face of a stencil, as its row is drawn. */
export interface FaceRow {
  readonly id: string
  readonly name: string
  /** Where it stands, counting from one, which is what it is announced as. */
  readonly at: number
  /** How many faces stand with it. */
  readonly of: number
  /** The fields a card is asked for, which are what may be written into a half. */
  readonly fields: readonly string[]
  /** The names the other faces carry, which this one's is measured against. */
  readonly taken: readonly string[]
  /** What is written in the two boxes. */
  readonly front: string
  readonly back: string
  /** The same two, with the sample values standing in the braces. */
  readonly frontPreview: string
  readonly backPreview: string
  /** The slots each half names that the fields do not, each said once. */
  readonly frontStray: readonly string[]
  readonly backStray: readonly string[]
}

/**
 * A stencil's faces as they are drawn, each carrying what its preview shows and
 * what is wrong in each half of it.
 */
export function faceRows(
  faces: readonly StencilFace[],
  fields: readonly string[],
  sample: readonly FieldValue[],
): readonly FaceRow[] {
  const names = faces.map((each) => each.name)
  return faces.map((face, index) => {
    return {
      id: face.id,
      name: face.name,
      at: index + 1,
      of: faces.length,
      fields,
      taken: names.filter((_, at) => at !== index),
      front: face.front,
      back: face.back,
      frontPreview: renderPreview(face.front, sample, fields),
      backPreview: renderPreview(face.back, sample, fields),
      frontStray: strayIn(face.front, fields),
      backStray: strayIn(face.back, fields),
    }
  })
}

/** What one part of the window a face is edited in holds. */
export type PaneMode = 'written' | 'preview'

/** One part of the window a face is edited in. */
export interface Pane {
  readonly half: Half
  readonly mode: PaneMode
  /** What the part is called while nothing stands in it. */
  readonly said: string
  /** What the part is announced as. */
  readonly named: string
  /** The markup as it is written, or the sample standing in its braces. */
  readonly text: string
  /** Nothing but space stands in it. */
  readonly blank: boolean
  /** The slots the half names that the fields do not, said under the markup. */
  readonly stray: readonly string[]
}

/**
 * The four parts of one face, in the order they are drawn: each half's markup
 * and then what that markup comes to. Two parts to a row stand the front above
 * the back; one to a row stands each preview under the half it is of.
 */
export function getPanes(face: FaceRow, words: StencilWords = STENCIL_WORDS): readonly Pane[] {
  return HALVES.flatMap((half): readonly Pane[] => {
    const said = half === 'front' ? words.front : words.back
    const written = half === 'front' ? face.front : face.back
    const shown = half === 'front' ? face.frontPreview : face.backPreview
    return [
      {
        half,
        mode: 'written',
        said,
        named: said,
        text: written,
        blank: written.trim() === '',
        stray: half === 'front' ? face.frontStray : face.backStray,
      },
      {
        half,
        mode: 'preview',
        said: words.preview,
        named: `${words.preview}: ${face.name} ${said}`,
        text: shown,
        blank: shown.trim() === '',
        stray: [],
      },
    ]
  })
}
