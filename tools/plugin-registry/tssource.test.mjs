/**
 * Tests of the TypeScript reader: what the syntax trees of TypeScript and Vue files show.
 *
 * Run: node --test tools/plugin-registry/tssource.test.mjs
 */
import assert from 'node:assert/strict'
import test from 'node:test'
import {
  countStories,
  countTestCases,
  listImportedModules,
  listNetworkCalls,
  listStringTexts,
  parseSourceFile,
  parseTypeScript,
  readVueScripts,
} from './tssource.mjs'

test('commented-out code has no calls, literals, test cases or stories', () => {
  const tree = parseTypeScript(`
// fetch('https://disabled.numen.md')
/*
new WebSocket('wss://removed.numen.md')
it('was removed', () => {})
export const Removed = {}
*/
export const kept = 1
`)

  assert.deepEqual(listNetworkCalls(tree), [])
  assert.deepEqual(listStringTexts(tree), [])
  assert.equal(countTestCases(tree), 0)
  assert.equal(countStories(tree), 0)
})

test('a call written inside a string is a literal and not a call', () => {
  const tree = parseTypeScript(`const sample = "fetch('https://quoted.numen.md')"\nconst other = "it('is quoted', () => {})"\n`)

  assert.deepEqual(listNetworkCalls(tree), [])
  assert.equal(countTestCases(tree), 0)
  assert.deepEqual(listStringTexts(tree), ["fetch('https://quoted.numen.md')", "it('is quoted', () => {})"])
})

test('a network call carries the target its first argument starts with', () => {
  const tree = parseTypeScript(`
import { getAssetUrl } from '@numen/plugin-sdk'
fetch('https://api.numen.md/a')
fetch(\`https://api.numen.md/\${id}\`)
fetch(\`\${base}/b\`)
fetch(getAssetUrl(path))
new EventSource(url)
new XMLHttpRequest()
cache.fetch('https://cache.numen.md')
window.fetch('https://window.numen.md')
`)

  assert.deepEqual(listNetworkCalls(tree), [
    { name: 'fetch', target: { kind: 'literal', text: 'https://api.numen.md/a' } },
    { name: 'fetch', target: { kind: 'literal', text: 'https://api.numen.md/' } },
    { name: 'fetch', target: { kind: 'unknown' } },
    { name: 'fetch', target: { kind: 'helper' } },
    { name: 'EventSource', target: { kind: 'unknown' } },
    { name: 'XMLHttpRequest', target: { kind: 'unknown' } },
  ])
})

test('a story imports a component under an alias, as a namespace, by re-export or dynamically', () => {
  const tree = parseTypeScript(`
import { default as Row } from './parts/Row.vue'
import * as Cell from './parts/Cell.vue'
import type { Meta } from '@storybook/vue3'
export { default as Tab } from '../Tab.vue'
const Lazy = () => import('./Lazy.vue')
// import Removed from './Removed.vue'
`)

  assert.deepEqual(listImportedModules(tree), [
    './parts/Row.vue',
    './parts/Cell.vue',
    '@storybook/vue3',
    '../Tab.vue',
    './Lazy.vue',
  ])
})

test('a Vue file is read from its script blocks, its template left out', () => {
  const text = `<template><a href="https://docs.numen.md">fetch('https://template.numen.md')</a></template>
<script lang="ts">
export default { name: 'Book' }
</script>
<script setup lang="ts">
const load = () => fetch('https://api.numen.md/books')
</script>
<style scoped>.book { color: red }</style>
`

  assert.equal(readVueScripts(text).length, 2)
  const calls = parseSourceFile('Book.vue', text).flatMap(listNetworkCalls)
  assert.deepEqual(calls, [{ name: 'fetch', target: { kind: 'literal', text: 'https://api.numen.md/books' } }])
})

test('a Vue file without a script block has no tree', () => {
  assert.deepEqual(parseSourceFile('Icon.vue', '<template><svg /></template>\n'), [])
})
