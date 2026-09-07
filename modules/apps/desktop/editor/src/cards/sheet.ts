/**
 * What a stencil is while the window holds it, and what each gesture makes of
 * it.
 *
 * The components take names and text and hand them back under identities they
 * were given. Minting those identities, and turning what a person did into the
 * fields and the faces a file is written from, is here.
 */
import { ordered, reordered, type InsertionPoint } from '@numen/ui'
import type { VaultFace, VaultStencil } from './vault'
import type { IdMaker } from './body'

const minting: IdMaker = () => crypto.randomUUID()

/** One face as the window holds it: what the file says, under an identity of its own. */
export interface Face extends VaultFace {
  readonly id: string
}

/** A stencil as the window holds it: its fields, and its faces under identities. */
export interface PageSize {
  readonly fields: readonly string[]
  readonly preamble: string
  readonly faces: readonly Face[]
  readonly tail: string
}

/** A stencil that names nothing and shows nothing. */
export const NO_SHEET: PageSize = { fields: [], preamble: '', faces: [], tail: '' }

/** A stencil as the vault read it, each face under an identity this window mints. */
export const sheetOf = (read: VaultStencil, mint: IdMaker = minting): PageSize => ({
  fields: read.fields,
  preamble: read.preamble,
  faces: read.faces.map((face) => ({ id: mint(), ...face })),
  tail: read.tail,
})

/**
 * A stencil as one string, which is what the tab holding it is dirty against.
 * The parts are written in a settled order, so a stencil that came back
 * unchanged reads as the string it went in as.
 */
export const sheetBodyOf = (sheet: PageSize): string =>
  JSON.stringify({
    fields: sheet.fields,
    preamble: sheet.preamble,
    tail: sheet.tail,
    faces: sheet.faces.map((face) => ({
      id: face.id,
      name: face.name,
      lead: face.lead,
      front: face.front,
      back: face.back,
    })),
  })

/** The stencil a string stands for. A string holding nothing names no field. */
export const sheetIn = (body: string): PageSize => (body ? (JSON.parse(body) as PageSize) : NO_SHEET)

/** The faces of a stencil, in the shape the vault takes them. */
export const facesOf = (sheet: PageSize): readonly VaultFace[] =>
  sheet.faces.map(({ name, lead, front, back }) => ({ name, lead, front, back }))

/** Whether two stencils read the same, the identities left out the same way. */
export const sameSheet = (one: PageSize, other: PageSize): boolean =>
  one.preamble === other.preamble &&
  one.tail === other.tail &&
  JSON.stringify(one.fields) === JSON.stringify(other.fields) &&
  JSON.stringify(facesOf(one)) === JSON.stringify(facesOf(other))

/** A field named at the end of the order. */
export const fieldAdded = (sheet: PageSize, name: string): PageSize => ({
  ...sheet,
  fields: [...sheet.fields, name],
})

/** A field the stencil no longer names. What the faces stand in its braces stays. */
export const fieldGone = (sheet: PageSize, field: string): PageSize => ({
  ...sheet,
  fields: sheet.fields.filter((one) => one !== field),
})

/**
 * A field let go somewhere in the order. The first field names every card the
 * stencil cuts, so it stays first and nothing lands above it.
 */
export const fieldDropped = (sheet: PageSize, field: string, at: InsertionPoint): PageSize => ({
  ...sheet,
  fields: reordered(sheet.fields, field, at),
})

/**
 * A face let go somewhere in the order: before another, or at the end. The
 * order of the faces is the order a card's repetitions are taken from it, and
 * nothing among them is fixed.
 */
export const faceDropped = (sheet: PageSize, id: string, at: InsertionPoint): PageSize => {
  const order = ordered(
    sheet.faces.map((face) => face.id),
    id,
    at,
  )
  const held = new Map(sheet.faces.map((face) => [face.id, face]))
  return { ...sheet, faces: order.flatMap((one) => held.get(one) ?? []) }
}

/** A face added at the end, with both its halves empty. */
export const faceAdded = (sheet: PageSize, name: string, mint: IdMaker = minting): PageSize => ({
  ...sheet,
  faces: [...sheet.faces, { id: mint(), name, lead: '', front: '', back: '' }],
})

/** A face under another name. */
export const faceNamed = (sheet: PageSize, id: string, name: string): PageSize => ({
  ...sheet,
  faces: sheet.faces.map((face) => (face.id === id ? { ...face, name } : face)),
})

/** A face taken out of the stencil. */
export const faceGone = (sheet: PageSize, id: string): PageSize => ({
  ...sheet,
  faces: sheet.faces.filter((face) => face.id !== id),
})

/** One half of one face as it now reads. */
export const faceWritten = (
  sheet: PageSize,
  id: string,
  half: 'front' | 'back',
  text: string,
): PageSize => ({
  ...sheet,
  faces: sheet.faces.map((face) => (face.id === id ? { ...face, [half]: text } : face)),
})
