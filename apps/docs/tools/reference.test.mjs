/**
 * The settings reference of the manual is built from the declarations: the JSON of the owner `app`
 * and the manifest of every plugin. The page is the file the command `manual.mjs` writes, and
 * `manual.mjs --check` fails while the page differs from what the declarations say.
 *
 * Run with `node --test apps/docs/tools/reference.test.mjs` from the root of the repository.
 */
import assert from 'node:assert/strict'
import { spawnSync } from 'node:child_process'
import { readFileSync, readdirSync, writeFileSync } from 'node:fs'
import { join } from 'node:path'
import test from 'node:test'
import { fileURLToPath } from 'node:url'

const ROOT = fileURLToPath(new URL('../../../', import.meta.url))
const MANUAL = join(ROOT, 'apps/docs/tools/manual.mjs')
const REFERENCE = join(ROOT, 'apps/docs/src/content/docs/reference.md')
const APP_JSON = join(ROOT, 'libs/core/kernel/adapter/settings/app.json')
const PLUGINS = join(ROOT, 'plugins')

const readJson = (file) => JSON.parse(readFileSync(file, 'utf8'))

const readPlugins = () =>
  readdirSync(PLUGINS, { withFileTypes: true })
    .filter((entry) => entry.isDirectory())
    .map((entry) => join(PLUGINS, entry.name, 'manifest.json'))
    .flatMap((file) => {
      try {
        return [{ file, document: readJson(file) }]
      } catch {
        return []
      }
    })

const plugins = readPlugins()
const withSettings = plugins.filter(({ document }) => (document.settings ?? []).length > 0)

/** The lines under the first heading that names `name`, down to the next heading of its level or above. */
const getSection = (page, name) => {
  const lines = page.split('\n')
  const start = lines.findIndex((line) => /^#{1,6} /.test(line) && line.includes(name))
  if (start < 0) return null
  const level = lines[start].match(/^#+/)[0].length
  const rest = lines.slice(start + 1)
  const end = rest.findIndex((line) => {
    const heading = line.match(/^(#{1,6}) /)
    return heading !== null && heading[1].length <= level
  })
  return (end < 0 ? rest : rest.slice(0, end)).join('\n')
}

const runCheck = () =>
  spawnSync(process.execPath, [MANUAL, '--check'], { cwd: ROOT, encoding: 'utf8' })

/** Runs `check` while `file` holds `change(text)`, and puts the original bytes back after. */
const withChangedFile = (file, change, check) => {
  const original = readFileSync(file, 'utf8')
  try {
    writeFileSync(file, change(original))
    check()
  } finally {
    writeFileSync(file, original)
  }
}

test('the plugins that declare settings exist to be documented', () => {
  assert.ok(withSettings.length > 0)
})

test('the reference names every setting of the owner app', () => {
  const page = readFileSync(REFERENCE, 'utf8')
  for (const one of readJson(APP_JSON).settings) {
    const [section, ...rest] = one.key.split('.')
    const body = getSection(page, `app.${section}`)
    assert.notEqual(body, null, `no section for app.${section}`)
    assert.ok(body.includes(`\`${rest.join('.')}\``), `app.${one.key} is not in its section`)
  }
})

for (const { document } of withSettings) {
  test(`the reference has a section for ${document.id} holding every setting its manifest declares`, () => {
    const body = getSection(readFileSync(REFERENCE, 'utf8'), document.id)
    assert.notEqual(body, null, `no section for ${document.id}`)
    for (const one of document.settings) {
      assert.ok(
        body.includes(`\`${one.key}\``),
        `${document.id}: \`${one.key}\` is not in its section`,
      )
    }
  })
}

test('the reference describes a plugin setting in the full words of its manifest, or by its title without them', () => {
  const page = readFileSync(REFERENCE, 'utf8')
  for (const { document } of withSettings) {
    const body = getSection(page, document.id) ?? ''
    for (const one of document.settings) {
      const row = body.split('\n').find((line) => line.includes(`\`${one.key}\``)) ?? ''
      const said = one.details ?? one.description ?? one.title
      const words = said.replace(/\.$/, '').toLowerCase()
      assert.ok(
        row.toLowerCase().includes(words),
        `${document.id}.${one.key}: the row does not say "${said}"`,
      )
    }
  }
})

test('the check passes while the reference equals the declarations', () => {
  const result = runCheck()
  assert.equal(result.status, 0, result.stderr)
})

test('the check fails when a plugin manifest declares a setting the reference lacks', () => {
  const { file } = withSettings[0]
  withChangedFile(
    file,
    (text) => {
      const document = JSON.parse(text)
      document.settings.push({
        key: 'referenceProbe',
        type: 'boolean',
        title: 'Probe',
        description: 'A probe setting.',
        default: false,
      })
      return `${JSON.stringify(document, null, 2)}\n`
    },
    () => {
      const result = runCheck()
      assert.notEqual(result.status, 0, 'the check passed')
      assert.match(result.stderr, /reference\.md/)
    },
  )
})

test('the check fails when the core declaration holds a setting the reference lacks', () => {
  withChangedFile(
    APP_JSON,
    (text) => {
      const document = JSON.parse(text)
      document.settings.push({
        key: 'naming.referenceProbe',
        type: 'boolean',
        title: 'Probe',
        description: 'A probe setting.',
        default: false,
      })
      return `${JSON.stringify(document, null, 2)}\n`
    },
    () => {
      const result = runCheck()
      assert.notEqual(result.status, 0, 'the check passed')
      assert.match(result.stderr, /reference\.md/)
    },
  )
})

test('the check fails when a plugin setting is removed from its manifest', () => {
  const { file } = withSettings[0]
  withChangedFile(
    file,
    (text) => {
      const document = JSON.parse(text)
      document.settings.pop()
      return `${JSON.stringify(document, null, 2)}\n`
    },
    () => {
      const result = runCheck()
      assert.notEqual(result.status, 0, 'the check passed')
      assert.match(result.stderr, /reference\.md/)
    },
  )
})

test('the reference names a list setting a list of text', () => {
  const { file, document } = withSettings[0]
  withChangedFile(REFERENCE, (text) => text, () => {
    withChangedFile(
      file,
      (text) => {
        const changed = JSON.parse(text)
        changed.settings.push({
          key: 'referenceListProbe',
          type: 'list',
          title: 'Probe folders',
          description: 'Folders a probe skips.',
          default: [],
        })
        return `${JSON.stringify(changed, null, 2)}\n`
      },
      () => {
        const result = spawnSync(process.execPath, [MANUAL], { cwd: ROOT, encoding: 'utf8' })
        assert.equal(result.status, 0, result.stderr)
        const body = getSection(readFileSync(REFERENCE, 'utf8'), document.id) ?? ''
        const row = body.split('\n').find((line) => line.includes('`referenceListProbe`')) ?? ''
        assert.match(row, /\| a list of text \|/)
      },
    )
  })
})
