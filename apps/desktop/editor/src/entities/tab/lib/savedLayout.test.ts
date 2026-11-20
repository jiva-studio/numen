import { expect, it } from 'vitest'
import { branch, pane } from '@numen/workbench'
import { readSavedLayout } from './savedLayout'

const tabs = ['a', 'b']
const left = pane('left', ['a'])
const right = pane('right', ['b'])
const split = branch('root', [left, right], [0.4, 0.6])
const layout = { root: split, focus: 'left', axis: 'horizontal' }

it.each([
  ['invalid JSON', '{'],
  ['invalid root', '{}'],
  ['unknown axis', JSON.stringify({ ...layout, axis: 'left' })],
  ['unknown focused pane', JSON.stringify({ ...layout, focus: 'root' })],
  ['duplicate node id', JSON.stringify({ ...layout, root: { ...split, id: left.id } })],
  ['duplicate tab', JSON.stringify({ ...layout, root: pane('left', ['a', 'b', 'a']) })],
  ['absent tab', JSON.stringify({ ...layout, root: left })],
  ['unknown active tab', JSON.stringify({ ...layout, root: pane('left', tabs, 'gone') })],
  ['invalid pane size', JSON.stringify({ ...layout, root: { ...split, sizes: [-1, 2] } })],
  ['missing pane size', JSON.stringify({ ...layout, root: { ...split, sizes: [1] } })],
  ['invalid total size', JSON.stringify({ ...layout, root: { ...split, sizes: [1, 1] } })],
])('rejects a saved workspace with %s', (_reason, json) => {
  expect(readSavedLayout(json, tabs)).toEqual({ ok: false, error: 'invalid-layout' })
})
