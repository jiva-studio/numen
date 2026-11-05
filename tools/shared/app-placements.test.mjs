/**
 * The documentation for plugin authors says the application places commands of its own in the
 * same places, in the same way a plugin does.
 */
import assert from 'node:assert/strict'
import { readFileSync } from 'node:fs'
import { join } from 'node:path'
import { fileURLToPath } from 'node:url'
import test from 'node:test'

const ROOT = fileURLToPath(new URL('../../', import.meta.url))

test('the docs say the application places commands of its own in the places of the menus', () => {
  const readme = readFileSync(join(ROOT, 'docs/plugins/README.md'), 'utf8')
  const menus = readme.slice(readme.indexOf('\n## Menus'))
  const section = menus.slice(0, menus.indexOf('\n## ', 4))
  assert.match(
    section,
    /the application places (commands|a command) of its own/i,
    'the Menus section does not say the application places commands of its own',
  )
  assert.match(section, /app:settings\.open/, 'the section does not name the settings button')
  assert.match(section, /`settings`/, 'the section does not name the group of the application')
})
