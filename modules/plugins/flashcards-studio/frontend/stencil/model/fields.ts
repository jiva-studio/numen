/**
 * Buffer parsing, marks caching, and field/face editing actions for stencil tabs.
 */
import type { Half } from '@numen/ui'
import {
  addFace,
  addField,
  moveFace,
  moveField,
  removeFace,
  removeField,
  renameFace,
  areStencilsEqual,
  serializeStencil,
  parseStencil,
  writeFace,
  type BufferStencil,
} from '../lib/stencil'
import { createMarks, areMarksEqual, type Marks, type DeckProblem } from '../../cards'

export interface StencilFieldsDeps {
  getBody: (id: string) => string
  onTyped: (id: string, body: string) => void
  getProblems: (id: string) => readonly DeckProblem[]
  renameFieldOnWire: (id: string, field: string, name: string) => void
}

function createStencilActions(
  id: string,
  deps: {
    getStencil: (id: string) => BufferStencil
    applyStencil: (id: string, stencil: BufferStencil) => void
    renameFieldOnWire: (id: string, field: string, name: string) => void
  },
) {
  const { getStencil, applyStencil, renameFieldOnWire } = deps
  return {
    addField: (name: string) => applyStencil(id, addField(getStencil(id), name)),
    renameField: (field: string, name: string) => void renameFieldOnWire(id, field, name),
    removeField: (field: string) => applyStencil(id, removeField(getStencil(id), field)),
    moveField: (field: string, at: string | null) =>
      applyStencil(id, moveField(getStencil(id), field, at)),
    addFace: (name: string) => applyStencil(id, addFace(getStencil(id), name)),
    renameFace: (face: string, name: string) =>
      applyStencil(id, renameFace(getStencil(id), face, name)),
    removeFace: (face: string) => applyStencil(id, removeFace(getStencil(id), face)),
    moveFace: (face: string, at: string | null) =>
      applyStencil(id, moveFace(getStencil(id), face, at)),
    writeFaceHalf: (face: string, half: Half, text: string) =>
      applyStencil(id, writeFace(getStencil(id), face, half, text)),
  }
}

function resolveStencilFieldsDeps(
  args:
    | [deps: StencilFieldsDeps]
    | [
        getBody: (id: string) => string,
        onTyped: (id: string, body: string) => void,
        getProblems: (id: string) => readonly DeckProblem[],
        renameFieldOnWire: (id: string, field: string, name: string) => void,
      ],
): StencilFieldsDeps {
  return args.length === 1
    ? args[0]
    : {
        getBody: args[0],
        onTyped: args[1],
        getProblems: args[2],
        renameFieldOnWire: args[3],
      }
}

export function createStencilFields(
  ...args:
    | [deps: StencilFieldsDeps]
    | [
        getBody: (id: string) => string,
        onTyped: (id: string, body: string) => void,
        getProblems: (id: string) => readonly DeckProblem[],
        renameFieldOnWire: (id: string, field: string, name: string) => void,
      ]
) {
  const { getBody, onTyped, getProblems, renameFieldOnWire } = resolveStencilFieldsDeps(args)
  const parsed = new Map<string, { body: string; stencil: BufferStencil }>()
  const marked = new Map<string, { problems: readonly DeckProblem[]; marks: Marks }>()

  const getStencil = (id: string): BufferStencil => {
    const body = getBody(id)
    const held = parsed.get(id)
    const read = parseStencil(body)
    const stencil = held && areStencilsEqual(held.stencil, read) ? held.stencil : read
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
    const body = serializeStencil(stencil)
    parsed.set(id, { body, stencil })
    onTyped(id, body)
  }

  const forget = (id: string): void => {
    parsed.delete(id)
    marked.delete(id)
  }

  return {
    getStencil,
    getMarks,
    applyStencil,
    forget,
    createActions: (id: string) =>
      createStencilActions(id, { getStencil, applyStencil, renameFieldOnWire }),
  }
}
