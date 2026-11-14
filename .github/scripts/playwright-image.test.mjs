// A browser in the Playwright image starts only under the Playwright release that baked it.
import assert from 'node:assert/strict'
import { readdirSync, readFileSync } from 'node:fs'
import { describe, it } from 'node:test'

const ROOT = new URL('../../', import.meta.url)
const WORKFLOWS = new URL('.github/workflows/', ROOT)

const readLockedVersion = (name) =>
  JSON.parse(readFileSync(new URL('package-lock.json', ROOT), 'utf8')).packages[`node_modules/${name}`].version

const listImageVersions = () =>
  readdirSync(WORKFLOWS)
    .filter((file) => file.endsWith('.yml'))
    .flatMap((file) =>
      [...readFileSync(new URL(file, WORKFLOWS), 'utf8').matchAll(/mcr\.microsoft\.com\/playwright:v([^-\s]+)-/g)].map(
        (match) => ({ file, version: match[1] }),
      ),
    )

describe('the Playwright image', () => {
  it('has the version of playwright-core in package-lock.json in every workflow', () => {
    const images = listImageVersions()
    assert.ok(images.length > 0, 'no workflow names the Playwright image')
    const locked = readLockedVersion('playwright-core')
    for (const { file, version } of images) assert.equal(version, locked, file)
  })
})
