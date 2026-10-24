/**
 * Makes `@stryker-mutator/vitest-runner` 10.0.0 name tests the way Vitest 5
 * matches them: the suite chain joined with ` > `. Unpatched, the per-mutant
 * name filter selects no test and every covered mutant survives
 * (stryker-mutator/stryker-js#6210). The patch is applied in place and is a
 * no-op once applied.
 */
import { readFileSync, writeFileSync } from 'node:fs'
import { createRequire } from 'node:module'
import { dirname, join } from 'node:path'
import { root } from './targets.mjs'

export const PATCHED_VERSION = '10.0.0'

const FILES = ['dist/src/stryker-setup.js', 'dist/src/test-helpers.js']

const UNPATCHED = "return nameParts.join(' ').trim();"
const PATCHED =
  "return nameParts.map((part) => part.trim()).filter((part) => part !== '').join(' > ');"

export const patchSource = (source) => {
  if (source.includes(PATCHED)) return source
  if (!source.includes(UNPATCHED))
    throw new Error('the vitest runner no longer names tests as this patch expects')
  return source.replace(UNPATCHED, PATCHED)
}

export const patchVitestRunner = () => {
  const require = createRequire(join(root, 'modules/package.json'))
  const manifest = require.resolve('@stryker-mutator/vitest-runner/package.json')
  const { version } = JSON.parse(readFileSync(manifest, 'utf8'))
  if (version !== PATCHED_VERSION) {
    throw new Error(
      `@stryker-mutator/vitest-runner is ${version}; patch.mjs is written for ${PATCHED_VERSION}. Check whether stryker-js#6210 is fixed in it and drop or rewrite the patch.`,
    )
  }
  for (const file of FILES) {
    const path = join(dirname(manifest), file)
    const source = readFileSync(path, 'utf8')
    const patched = patchSource(source)
    if (patched !== source) writeFileSync(path, patched)
  }
}
