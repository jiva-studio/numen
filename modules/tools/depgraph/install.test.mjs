/**
 * A package is named, and a path into the folder it was installed in is not a
 * name. The two edges stand side by side in one fixture: the same package,
 * reached once by its name and once by the path the install happens to have
 * put it at.
 *
 * The fixture is written in a temporary folder because the whole of it is a
 * node_modules, which is the one folder this repository does not commit.
 */
import assert from 'node:assert/strict'
import { spawnSync } from 'node:child_process'
import { mkdirSync, mkdtempSync, rmSync, writeFileSync } from 'node:fs'
import { tmpdir } from 'node:os'
import { join } from 'node:path'
import test from 'node:test'
import { config, depcruise } from './modules.mjs'

/** A module with the two edges, and the package the install hoisted above it. */
function fixture(under) {
  const wrote = (at, text) => {
    mkdirSync(join(under, at, '..'), { recursive: true })
    writeFileSync(join(under, at), text)
  }
  wrote('node_modules/@fixture/held/package.json', '{"name":"@fixture/held","version":"1.0.0","main":"index.js"}\n')
  wrote('node_modules/@fixture/held/index.js', 'export const held = 1\n')
  wrote('mod/package.json', '{"name":"mod","private":true,"dependencies":{"@fixture/held":"1.0.0"}}\n')
  wrote(
    'mod/tsconfig.json',
    '{"compilerOptions":{"target":"esnext","module":"esnext","moduleResolution":"bundler","strict":true,"noEmit":true},"include":["src"]}\n',
  )
  wrote(
    'mod/src/named.ts',
    ["import { held } from '@fixture/held'", 'export const named = held', ''].join('\n'),
  )
  wrote(
    'mod/src/pathed.ts',
    [
      "import { held } from '../../node_modules/@fixture/held/index.js'",
      'export const pathed = held',
      '',
    ].join('\n'),
  )
  return join(under, 'mod')
}

test('a package is reached by its name, and never by the path it was installed at', () => {
  const under = mkdtempSync(join(tmpdir(), 'install-'))
  try {
    const run = spawnSync(depcruise, ['--config', config, '--output-type', 'json', 'src'], {
      cwd: fixture(under),
      encoding: 'utf8',
      maxBuffer: 64 * 1024 * 1024,
    })
    assert.ok(run.stdout, `the fixture cruise did not run: ${run.stderr?.trim() || run.error}`)
    const out = JSON.parse(run.stdout)

    // A cruise that read neither file refuses nothing and says so in the same
    // words as one that read both.
    const read = out.modules.map((one) => one.source)
    for (const at of ['src/named.ts', 'src/pathed.ts']) {
      assert.ok(read.includes(at), `the fixture cruise did not read ${at}`)
    }

    const refused = out.summary.violations.map((one) => `${one.rule.name}: ${one.from} → ${one.to}`)
    assert.deepEqual(refused, [
      'no-path-into-the-install: src/pathed.ts → ../node_modules/@fixture/held/index.js',
    ])
  } finally {
    rmSync(under, { recursive: true, force: true })
  }
})
