/**
 * Buffer parsing, marks caching, and field/face editing actions for stencil tabs.
 */
import type { Half } from '@numen/ui'
import {
  faceAdded,
  faceDropped,
  faceGone,
  faceNamed,
  faceWritten,
  fieldAdded,
  fieldDropped,
  fieldGone,
  sameStencil,
  stencilBodyOf,
  stencilIn,
  type BufferStencil,
} from './stencil'
import { marksOf, sameMarks, type Marks } from '../shared/flashcards/marks'
import type { Problem } from '../shared/flashcards/cards'

export function createStencilFields(
  getBody: (id: string) => string,
  onTyped: (id: string, body: string) => void,
  getProblems: (id: string) => readonly Problem[],
  renameFieldOnWire: (id: string, field: string, name: string) => void,
) {
  const parsed = new Map<string, { body: string; stencil: BufferStencil }>()
  const marked = new Map<string, { problems: readonly Problem[]; marks: Marks }>()

  const getStencil = (id: string): BufferStencil => {
    const body = getBody(id)
    const held = parsed.get(id)
    if (held && held.body === body) return held.stencil
    const read = stencilIn(body)
    const stencil = held && sameStencil(held.stencil, read) ? held.stencil : read
    parsed.set(id, { body, stencil })
    return stencil
  }

  const getMarks = (id: string): Marks => {
    const problems = getProblems(id)
    const held = marked.get(id)
    if (held && held.problems === problems) return held.marks
    const read = marksOf(
      problems,
      [],
      getStencil(id).faces.map((face) => face.id),
    )
    const marks = held && sameMarks(held.marks, read) ? held.marks : read
    marked.set(id, { problems, marks })
    return marks
  }

  const turns = (id: string, stencil: BufferStencil): void => {
    const body = stencilBodyOf(stencil)
    parsed.set(id, { body, stencil })
    onTyped(id, body)
  }

  const forget = (id: string): void => {
    parsed.delete(id)
    marked.delete(id)
  }

  const actionsFor = (id: string) => ({
    addField: (name: string) => turns(id, fieldAdded(getStencil(id), name)),
    renameField: (field: string, name: string) => void renameFieldOnWire(id, field, name),
    removeField: (field: string) => turns(id, fieldGone(getStencil(id), field)),
    moveField: (field: string, at: string | null) => turns(id, fieldDropped(getStencil(id), field, at)),
    addFace: (name: string) => turns(id, faceAdded(getStencil(id), name)),
    renameFace: (face: string, name: string) => turns(id, faceNamed(getStencil(id), face, name)),
    removeFace: (face: string) => turns(id, faceGone(getStencil(id), face)),
    moveFace: (face: string, at: string | null) => turns(id, faceDropped(getStencil(id), face, at)),
    writeFaceHalf: (face: string, half: Half, text: string) => turns(id, faceWritten(getStencil(id), face, half, text)),
  })

  return {
    getStencil,
    getMarks,
    turns,
    forget,
    actionsFor,
  }
}
