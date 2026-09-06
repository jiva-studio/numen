import assert from 'node:assert/strict'
import test from 'node:test'
import { composables, declaration } from './composables.mjs'
import { blocks, sources } from './source.mjs'

/** The parts of a file that are code: a component's is in its script blocks. */
const codeOf = ({ at, text }) => (at.endsWith('.vue') ? blocks(text, 'script').join('\n') : text)

const declared = () => {
  const found = []
  for (const one of sources(['.ts', '.vue'])) {
    for (const said of composables(codeOf(one))) found.push({ at: one.at, ...said })
  }
  return found
}

/**
 * A `use` on a function that deals in no reactive state promises a composable
 * and hands back a plain value. The prefix is the only thing a caller reads
 * before deciding where to call it from.
 */
test('every use* of the interface modules deals in reactive state', () => {
  const found = declared()
  const wrong = found
    .filter((one) => !one.reactive)
    .map((one) => `${one.at} declares ${one.name}, which deals in no reactive state`)
  assert.deepEqual(wrong, [])

  // A walk that read no composable is a rule checked against nothing, and it
  // passes. A count alone cannot say which modules it read, so a file outside
  // the component library is named: the windows are where this walk stops
  // first if it stops at a package border.
  assert.ok(
    found.some(
      (one) => one.at.endsWith('apps/desktop/flashcards/src/window.ts') && one.name === 'useWindow',
    ),
    "the walk did not read useWindow in the review window, so the rule stops at the library's border",
  )
})

/**
 * What the rule refuses, read against declarations written to be refused. A
 * composable deals in reactive state in one of four ways and a machine can see
 * all four; a factory holding a ref and named for what it returns is nobody's
 * business here, because this rule runs one way only.
 */
test('what the composable rule refuses', () => {
  const cases = [
    { says: 'a use* over plain values', allowed: false, text: 'export function useSum(a, b) {\n  return a + b\n}' },
    { says: 'a use* handing back a frozen table', allowed: false, text: 'export const useIcons = () => {\n  return ICONS\n}' },
    { says: 'a use* making a ref', allowed: true, text: 'export function useTally() {\n  const n = ref(0)\n}' },
    { says: 'a use* taking a ref', allowed: true, text: 'export function useWidth(held: Ref<Element>) {\n  return 1\n}' },
    { says: 'a use* reading a value', allowed: true, text: 'export function useSaid(held) {\n  return held.value\n}' },
    { says: 'a use* holding another composable', allowed: true, text: 'export function useBoth(el) {\n  return useWidth(el)\n}' },
    { says: 'a use* registering a hook', allowed: true, text: 'export function useShut() {\n  onScopeDispose(stop)\n}' },
  ]

  const wrong = cases.filter((one) => composables(one.text).some((said) => !said.reactive))
  assert.deepEqual(
    wrong.map((one) => one.says),
    cases.filter((one) => !one.allowed).map((one) => one.says),
  )
})

/** A bare name is not this rule's business, however much state it holds. */
test('a factory that is not named use is never read', () => {
  assert.deepEqual(composables('export function raising() {\n  const told = shallowRef([])\n}'), [])
})

/**
 * A signature written over several lines closes its parameters at the margin,
 * and a walk reading to the first close at the margin stops before the body.
 */
test('a declaration is read past a signature closed at the margin', () => {
  const text = [
    'export function useDrag<At>(',
    '  drag: DragDeps<At>,',
    '): DragState<At> {',
    '  const dragged = shallowRef(null)',
    '}',
  ].join('\n')
  assert.ok(declaration(text, text.indexOf('<At>')).includes('shallowRef'))
  assert.deepEqual(composables(text), [{ name: 'useDrag', reactive: true }])
})
