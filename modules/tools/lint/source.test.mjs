import assert from 'node:assert/strict'
import { existsSync, readdirSync } from 'node:fs'
import { join } from 'node:path'
import test from 'node:test'
import { elsewhere, modules, root } from '../modules.mjs'
import { sources } from './source.mjs'

/** Every package standing under `modules/libs` and `modules/apps`. */
const packages = () => {
  const found = []
  for (const under of ['modules/libs', 'modules/apps']) {
    for (const one of readdirSync(join(root, under), { withFileTypes: true })) {
      if (!one.isDirectory()) continue
      const at = `${under}/${one.name}`
      if (existsSync(join(root, at, 'package.json'))) found.push(at)
      for (const inner of readdirSync(join(root, at), { withFileTypes: true })) {
        if (!inner.isDirectory()) continue
        const deeper = `${at}/${inner.name}`
        if (existsSync(join(root, deeper, 'package.json'))) found.push(deeper)
      }
    }
  }
  return found
}

/**
 * A rule holds for the modules the walk reaches and nowhere else, so a package
 * added to the repository and left off the list is a border the rules stop at
 * without anybody being told. The list was two lists once, a module apart.
 */
test('every package of the repository is read or is written off by name', () => {
  const named = new Set([...modules.map((one) => one.at), ...Object.keys(elsewhere)])
  assert.deepEqual(
    packages().filter((at) => !named.has(at)),
    [],
  )
})

/** A module named and not walked is the same border, reached the other way. */
test('every module the list names is walked, and has source under it', () => {
  const read = sources(['.ts', '.vue', '.mjs'])
  for (const one of modules) {
    assert.ok(
      read.some(({ at }) => at.startsWith(`${one.written}/`) || at === one.written),
      `${one.name}: nothing was read under ${one.written}`,
    )
    assert.ok(
      existsSync(join(root, one.at, one.reads)),
      `${one.name}: ${one.reads} is named as what a cruise must read and is not there`,
    )
  }
})
