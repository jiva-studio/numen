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
} from '../lib/stencil'
import { createMarks, areMarksEqual, type Marks } from '@/entities/deck'
import type { DeckProblem } from '@/entities/deck'

export function createStencilFields(
  getBody: (id: string) => string,
  onTyped: (id: string, body: string) => void,
  getProblems: (id: string) => readonly DeckProblem[],
  renameFieldOnWire: (id: string, field: string, name: string) => void,
) {
  const parsed = new Map<string, { body: string; stencil: BufferStencil }>()
  const marked = new Map<string, { problems: readonly DeckProblem[]; marks: Marks }>()

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
    const read = createMarks(
      problems,
      [],
      getStencil(id).faces.map((face) => face.id),
    )
    const marks = held && areMarksEqual(held.marks, read) ? held.marks : read
    marked.set(id, { problems, marks })
    return marks
  }

  const applyStencil = (id: string, stencil: BufferStencil): void => {
    const body = stencilBodyOf(stencil)
    parsed.set(id, { body, stencil })
    onTyped(id, body)
  }

  const forget = (id: string): void => {
    parsed.delete(id)
    marked.delete(id)
  }

  const actionsFor = (id: string) => ({
    addField: (name: string) => applyStencil(id,fieldAdded(getStencil(id), name)),
    renameField: (field: string, name: string) => void renameFieldOnWire(id, field, name),
    removeField: (field: string) => applyStencil(id,fieldGone(getStencil(id), field)),
    moveField: (field: string, at: string | null) => applyStencil(id,fieldDropped(getStencil(id), field, at)),
    addFace: (name: string) => applyStencil(id,faceAdded(getStencil(id), name)),
    renameFace: (face: string, name: string) => applyStencil(id,faceNamed(getStencil(id), face, name)),
    removeFace: (face: string) => applyStencil(id,faceGone(getStencil(id), face)),
    moveFace: (face: string, at: string | null) => applyStencil(id,faceDropped(getStencil(id), face, at)),
    writeFaceHalf: (face: string, half: Half, text: string) => applyStencil(id,faceWritten(getStencil(id), face, half, text)),
  })

  return {
    getStencil,
    getMarks,
    applyStencil,
    forget,
    actionsFor,
  }
}
