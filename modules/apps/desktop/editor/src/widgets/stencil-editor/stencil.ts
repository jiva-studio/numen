/**
 * What a stencil is while the window holds it, and what each gesture makes of
 * it.
 *
 * The components take names and text and hand them back under identities they
 * were given. Minting those identities, and turning what a person did into the
 * fields and the faces a file is written from, is here.
 */
import { ordered, reordered, type InsertionPoint } from '@numen/ui'
import type { VaultFace, VaultStencil } from '@/entities/deck/cards'
import { generateId, type IdMaker } from '@/entities/deck/identity'
import type { Surrounds } from '@/entities/deck/surrounds'

/** One face as the window holds it: what the file says, under an identity of its own. */
export interface BufferFace {
  /** The identity the window addresses it by, minted at every reading. */
  readonly id: string
  readonly name: string
  /** The prose between the face's heading and its first side. */
  readonly preamble: string
  readonly front: string
  readonly back: string
}

/** A stencil as the window holds it: its fields, and its faces under identities. */
export interface BufferStencil extends Surrounds {
  readonly fields: readonly string[]
  readonly faces: readonly BufferFace[]
}

/** A stencil that names nothing and shows nothing. */
export const NO_STENCIL: BufferStencil = { fields: [], preamble: '', faces: [], tail: '' }

/** A stencil as the vault read it, each face under an identity this window mints. */
export const stencilOf = (read: VaultStencil, mint: IdMaker = generateId): BufferStencil => ({
  fields: read.fields,
  preamble: read.preamble,
  faces: read.faces.map((face) => ({
    id: mint(),
    name: face.name,
    preamble: face.preamble,
    front: face.front,
    back: face.back,
  })),
  tail: read.tail,
})

/**
 * A stencil as one string, which is what the tab holding it is dirty against.
 * The parts are written in a settled order, so a stencil that came back
 * unchanged reads as the string it went in as.
 */
export const stencilBodyOf = (stencil: BufferStencil): string =>
  JSON.stringify({
    fields: stencil.fields,
    preamble: stencil.preamble,
    tail: stencil.tail,
    faces: stencil.faces.map((face) => ({
      id: face.id,
      name: face.name,
      preamble: face.preamble,
      front: face.front,
      back: face.back,
    })),
  })

/** The stencil a string stands for. A string holding nothing names no field. */
export const stencilIn = (body: string): BufferStencil => (body ? (JSON.parse(body) as BufferStencil) : NO_STENCIL)

/** The faces of a stencil, in the shape the vault takes them. */
export const facesOf = (stencil: BufferStencil): readonly VaultFace[] =>
  stencil.faces.map(({ name, preamble, front, back }) => ({ name, preamble, front, back }))

/** Whether two stencils read the same, the identities left out the same way. */
export const sameStencil = (one: BufferStencil, other: BufferStencil): boolean =>
  one.preamble === other.preamble &&
  one.tail === other.tail &&
  JSON.stringify(one.fields) === JSON.stringify(other.fields) &&
  JSON.stringify(facesOf(one)) === JSON.stringify(facesOf(other))

/** A field named at the end of the order. */
export const fieldAdded = (stencil: BufferStencil, name: string): BufferStencil => ({
  ...stencil,
  fields: [...stencil.fields, name],
})

/** A field the stencil no longer names. What the faces stand in its braces stays. */
export const fieldGone = (stencil: BufferStencil, field: string): BufferStencil => ({
  ...stencil,
  fields: stencil.fields.filter((one) => one !== field),
})

/**
 * A field let go somewhere in the order. The first field names every card the
 * stencil cuts, so it stays first and nothing lands above it.
 */
export const fieldDropped = (stencil: BufferStencil, field: string, at: InsertionPoint): BufferStencil => ({
  ...stencil,
  fields: reordered(stencil.fields, field, at),
})

/**
 * A face let go somewhere in the order: before another, or at the end. The
 * order of the faces is the order a card's repetitions are taken from it, and
 * nothing among them is fixed.
 */
export const faceDropped = (stencil: BufferStencil, id: string, at: InsertionPoint): BufferStencil => {
  const order = ordered(
    stencil.faces.map((face) => face.id),
    id,
    at,
  )
  const held = new Map(stencil.faces.map((face) => [face.id, face]))
  return { ...stencil, faces: order.flatMap((one) => held.get(one) ?? []) }
}

/** A face added at the end, with both its halves empty. */
export const faceAdded = (stencil: BufferStencil, name: string, mint: IdMaker = generateId): BufferStencil => ({
  ...stencil,
  faces: [...stencil.faces, { id: mint(), name, preamble: '', front: '', back: '' }],
})

/** A face under another name. */
export const faceNamed = (stencil: BufferStencil, id: string, name: string): BufferStencil => ({
  ...stencil,
  faces: stencil.faces.map((face) => (face.id === id ? { ...face, name } : face)),
})

/** A face taken out of the stencil. */
export const faceGone = (stencil: BufferStencil, id: string): BufferStencil => ({
  ...stencil,
  faces: stencil.faces.filter((face) => face.id !== id),
})

/** One half of one face as it now reads. */
export const faceWritten = (
  stencil: BufferStencil,
  id: string,
  half: 'front' | 'back',
  text: string,
): BufferStencil => ({
  ...stencil,
  faces: stencil.faces.map((face) => (face.id === id ? { ...face, [half]: text } : face)),
})
