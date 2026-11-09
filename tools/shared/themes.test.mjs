import assert from 'node:assert/strict'
import fs from 'node:fs'
import path from 'node:path'
import { describe, it } from 'node:test'
import postcss from 'postcss'
import { fileURLToPath } from 'node:url'

const REPO_ROOT = path.resolve(path.dirname(fileURLToPath(import.meta.url)), '../..')

const TOKEN = '--numen-titlebar'

/** The names of the custom properties a stylesheet declares. */
const readDeclaredTokens = (file) => {
  const names = new Set()
  postcss.parse(fs.readFileSync(file, 'utf8'), { from: file }).walkDecls((declaration) => {
    if (declaration.prop.startsWith('--')) names.add(declaration.prop)
  })
  return names
}

const themesFolder = path.join(REPO_ROOT, 'plugins', 'themes-presets', 'themes')
const themes = fs.readdirSync(themesFolder).filter((name) => name.endsWith('.css'))

describe('the title bar colour', () => {
  it('is declared by the default tokens', () => {
    const tokens = path.join(
      REPO_ROOT,
      'libs',
      'ui',
      'components',
      'src',
      'shared',
      'tokens',
      'tokens.css',
    )
    assert.ok(readDeclaredTokens(tokens).has(TOKEN))
  })

  for (const theme of themes) {
    it(`is declared by the ${theme} theme`, () => {
      assert.ok(readDeclaredTokens(path.join(themesFolder, theme)).has(TOKEN))
    })
  }

  it('is checked against every theme the plugin ships', () => {
    assert.equal(themes.length, 12)
  })
})
