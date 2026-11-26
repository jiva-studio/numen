/**
 * What the reader takes from a story source and what it refuses, and what the file check refuses.
 */
import assert from 'node:assert/strict'
import test from 'node:test'
import { getFileFaults } from './check.mjs'
import { readShotsOf } from './shots.mjs'

const source = (story) => `
import type { Meta, StoryObj } from '@storybook/vue3-vite'
const meta = { title: 'Garden/Shed' } satisfies Meta
export default meta
type Story = StoryObj<typeof meta>
export const Tools: Story = ${story}
export const Seeds: Story = {}
`

const PLUGIN = 'plugins/views-tasks/frontend/ui/Shed.stories.ts'
const WINDOW = 'libs/ui/workbench/src/screens/Shed.stories.ts'

test('a tagged story in a plugin writes into the screenshots folder of that plugin', () => {
  const { shots, faults } = readShotsOf(
    PLUGIN,
    source(`{ tags: ['screenshot'], parameters: { screenshot: [{ name: 'tools', width: 480, height: 1_200 }] } }`),
  )
  assert.deepEqual(faults, [])
  assert.deepEqual(shots, [
    {
      story: 'garden-shed--tools',
      name: 'tools',
      into: 'plugins/views-tasks/screenshots',
      width: 480,
      height: 1200,
      file: PLUGIN,
    },
  ])
})

test('a shot names the plugin or the app it is written into', () => {
  const { shots, faults } = readShotsOf(
    WINDOW,
    source(
      `{ tags: ['screenshot'], parameters: { screenshot: [{ name: 'tools', plugin: 'views-tasks' }, { name: 'shed', app: 'docs', preset: 'dracula' }] } }`,
    ),
  )
  assert.deepEqual(faults, [])
  assert.deepEqual(
    shots.map((shot) => [shot.into, shot.name, shot.width, shot.height]),
    [
      ['plugins/views-tasks/screenshots', 'tools', 1180, 740],
      ['apps/docs/src/assets', 'shed', 1180, 740],
    ],
  )
})

test('an untagged story without the parameter is no screenshot', () => {
  assert.deepEqual(readShotsOf(PLUGIN, source('{ args: {} }')), { shots: [], faults: [] })
})

for (const [what, story, said] of [
  [
    'a tag without shots',
    `{ tags: ['screenshot'] }`,
    'garden-shed--tools is tagged screenshot and declares no parameters.screenshot',
  ],
  [
    'shots without the tag',
    `{ parameters: { screenshot: [{ name: 'tools' }] } }`,
    'garden-shed--tools declares parameters.screenshot and is not tagged screenshot',
  ],
  [
    'shots that are not literals',
    `{ tags: ['screenshot'], parameters: { screenshot: [{ name: NAME }] } }`,
    'garden-shed--tools: parameters.screenshot is not written in literals',
  ],
  [
    'an empty list',
    `{ tags: ['screenshot'], parameters: { screenshot: [] } }`,
    'garden-shed--tools: parameters.screenshot is not a list of shots',
  ],
]) {
  test(`the reader refuses ${what}`, () => {
    assert.deepEqual(readShotsOf(PLUGIN, source(story)).faults, [said])
  })
}

for (const [what, file, shot, part] of [
  ['a shot outside plugins/ naming no target', WINDOW, `{ name: 'tools' }`, 'names neither a plugin nor an app'],
  ['an unknown app', PLUGIN, `{ name: 'tools', app: 'blog' }`, 'names the app blog'],
  ['a plugin that does not exist', PLUGIN, `{ name: 'tools', plugin: 'views-shed' }`, 'names the plugin views-shed'],
  ['a palette that does not exist', PLUGIN, `{ name: 'tools', preset: 'mauve' }`, 'names the palette mauve'],
  ['a name with capitals', PLUGIN, `{ name: 'Tools' }`, 'needs a name'],
  ['an unknown key', PLUGIN, `{ name: 'tools', zoom: 2 }`, 'unknown key zoom'],
  ['a size that is not whole', PLUGIN, `{ name: 'tools', width: 10.5 }`, 'width that is not a positive whole number'],
]) {
  test(`the reader refuses ${what}`, () => {
    const { shots, faults } = readShotsOf(
      file,
      source(`{ tags: ['screenshot'], parameters: { screenshot: [${shot}] } }`),
    )
    assert.deepEqual(shots, [])
    assert.equal(faults.length, 1)
    assert.match(faults[0], new RegExp(part))
  })
}

const SHOT = { story: 'garden-shed--tools', name: 'tools', into: 'plugins/views-tasks/screenshots' }
const LIGHT = 'plugins/views-tasks/screenshots/tools-light.webp'
const DARK = 'plugins/views-tasks/screenshots/tools-dark.webp'

test('a shot with both files and no other file is no fault', () => {
  assert.deepEqual(getFileFaults([SHOT], new Set([LIGHT, DARK]), [LIGHT, DARK]), [])
})

test('a shot without its dark file is a fault', () => {
  assert.deepEqual(getFileFaults([SHOT], new Set([LIGHT]), [LIGHT]), [
    `${DARK} is missing: run node tools/shoot/shoot.mjs --story garden-shed--tools`,
  ])
})

test('a file of a plugin no shot writes is a fault', () => {
  const stray = 'plugins/views-tasks/screenshots/rake-light.webp'
  assert.deepEqual(getFileFaults([SHOT], new Set([LIGHT, DARK, stray]), [LIGHT, DARK, stray]), [
    `${stray} is written by no story tagged screenshot`,
  ])
})

test('a tree that declares no shot is a fault', () => {
  assert.deepEqual(getFileFaults([], new Set(), []), ['no story declares a screenshot'])
})
