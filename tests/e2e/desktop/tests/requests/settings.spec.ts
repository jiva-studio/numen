import { mkdir, writeFile } from 'node:fs/promises'
import { dirname } from 'node:path'
import { test, expect } from '../support/test.js'
import { readOwner, readSettings, writeSetting } from '../support/vault.js'

const PLUGIN_ID = 'md.numen.agents.antigravity'

const createExecutable = async (): Promise<string> => {
  const path = test.info().outputPath('antigravity', 'bin', 'gemini')
  await mkdir(dirname(path), { recursive: true })
  await writeFile(path, '#!/bin/sh\n', { mode: 0o755 })
  return path
}

test.describe('The settings file', () => {
  // Every test of a shard shares the settings: leave the two values as a new vault has them.
  test.afterEach(async ({ request }) => {
    await writeSetting(request, [PLUGIN_ID, 'executablePath'], '')
    await writeSetting(request, ['app', 'appearance', 'textScale'], 1)
  })

  test('stores a path under the id of the plugin and keeps no plugins object', async ({
    request,
  }) => {
    const path = await createExecutable()
    await writeSetting(request, [PLUGIN_ID, 'executablePath'], path)

    const file = await readSettings(request)

    expect(file[PLUGIN_ID]).toMatchObject({ executablePath: path })
    expect(file).not.toHaveProperty('plugins')
    expect(file).toHaveProperty('app')
  })

  test('refuses a size outside the bounds of its setting and keeps the size in force', async ({
    request,
  }) => {
    await writeSetting(request, ['app', 'appearance', 'textScale'], 1.25)

    const answer = await request.post('/numen.v1.SettingsService/WriteSettings', {
      data: { settings: [{ at: ['app', 'appearance', 'textScale'], value: '9' }] },
      headers: { 'Content-Type': 'application/json' },
    })

    expect(answer.ok()).toBe(false)
    expect((await readOwner(request, 'app'))['appearance']).toMatchObject({ textScale: 1.25 })
  })
})
