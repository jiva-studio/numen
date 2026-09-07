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
import type { IdMaker } from './deck'

const minting: IdMaker = () => crypto.randomUUID()

/** One face as the window holds it: what the file says, under an identity of its own. */
export interface Face extends VaultFace {
  readonly id: string
}

/** A stencil as the window holds it: its fields, and its faces under identities. */
export interface Stencil {
  readonly fields: readonly string[]
  readonly preamble: string
  readonly faces: readonly Face[]
  readonly tail: string
}

/** A stencil that names nothing and shows nothing. */
export const NO_STENCIL: Stencil = { fields: [], preamble: '', faces: [], tail: '' }

/** A stencil as the vault read it, each face under an identity this window mints. */
export const stencilOf = (read: VaultStencil, mint: IdMaker = minting): Stencil => ({
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
export const stencilBodyOf = (stencil: Stencil): string =>
  JSON.stringify({
    fields: stencil.fields,
    preamble: stencil.preamble,
    tail: stencil.tail,
    faces: stencil.faces.map((face) => ({
      id: face.id,
      name: face.name,
      lead: face.lead,
      front: face.front,
      back: face.back,
    })),
  })

/** The stencil a string stands for. A string holding nothing names no field. */
export const stencilIn = (body: string): Stencil => (body ? (JSON.parse(body) as Stencil) : NO_STENCIL)

/** The faces of a stencil, in the shape the vault takes them. */
export const facesOf = (stencil: Stencil): readonly VaultFace[] =>
  stencil.faces.map(({ name, lead, front, back }) => ({ name, lead, front, back }))

/** Whether two stencils read the same, the identities left out the same way. */
export const sameStencil = (one: Stencil, other: Stencil): boolean =>
  one.preamble === other.preamble &&
  one.tail === other.tail &&
  JSON.stringify(one.fields) === JSON.stringify(other.fields) &&
  JSON.stringify(facesOf(one)) === JSON.stringify(facesOf(other))

/** A field named at the end of the order. */
export const fieldAdded = (stencil: Stencil, name: string): Stencil => ({
  ...stencil,
  fields: [...stencil.fields, name],
})

/** A field the stencil no longer names. What the faces stand in its braces stays. */
export const fieldGone = (stencil: Stencil, field: string): Stencil => ({
  ...stencil,
  fields: stencil.fields.filter((one) => one !== field),
})

/**
 * A field let go somewhere in the order. The first field names every card the
 * stencil cuts, so it stays first and nothing lands above it.
 */
export const fieldDropped = (stencil: Stencil, field: string, at: InsertionPoint): Stencil => ({
  ...stencil,
  fields: reordered(stencil.fields, field, at),
})

/**
 * A face let go somewhere in the order: before another, or at the end. The
 * order of the faces is the order a card's repetitions are taken from it, and
 * nothing among them is fixed.
 */
export const faceDropped = (stencil: Stencil, id: string, at: InsertionPoint): Stencil => {
  const order = ordered(
    stencil.faces.map((face) => face.id),
    id,
    at,
  )
  const held = new Map(stencil.faces.map((face) => [face.id, face]))
  return { ...stencil, faces: order.flatMap((one) => held.get(one) ?? []) }
}

/** A face added at the end, with both its halves empty. */
export const faceAdded = (stencil: Stencil, name: string, mint: IdMaker = minting): Stencil => ({
  ...stencil,
  faces: [...stencil.faces, { id: mint(), name, lead: '', front: '', back: '' }],
})

/** A face under another name. */
export const faceNamed = (stencil: Stencil, id: string, name: string): Stencil => ({
  ...stencil,
  faces: stencil.faces.map((face) => (face.id === id ? { ...face, name } : face)),
})

/** A face taken out of the stencil. */
export const faceGone = (stencil: Stencil, id: string): Stencil => ({
  ...stencil,
  faces: stencil.faces.filter((face) => face.id !== id),
})

/** One half of one face as it now reads. */
export const faceWritten = (
  stencil: Stencil,
  id: string,
  half: 'front' | 'back',
  text: string,
): Stencil => ({
  ...stencil,
  faces: stencil.faces.map((face) => (face.id === id ? { ...face, [half]: text } : face)),
})
