import { test, expect } from '../../support/test.js'
import { startOwnServer, type InstalledPlugin, type OwnServer } from '../../support/server.js'
import { expectWorkspaceDrawn as expectWorkspace } from '../../support/workspace.js'

/** A plugin with a backend whose process never says it is ready. */
const stuckPlugin: InstalledPlugin = {
  id: 'md.numen.test.stuck',
  files: {
    'manifest.json': JSON.stringify({
      id: 'md.numen.test.stuck',
      name: 'Stuck plugin',
      version: '2026.10.1',
      protocol: 1,
      description: 'A plugin that never answers',
      targets: ['desktop.editor'],
      permissions: [],
      backend: { entrypoint: 'bin/stuck' },
      services: ['md.numen.test.stuck'],
    }),
    'bin/stuck': '#!/bin/sh\nexec sleep 3600\n',
  },
}

test.describe('Start-up while a plugin does not start', () => {
  let server: OwnServer

  test.beforeAll(async () => {
    server = await startOwnServer([stuckPlugin])
  })

  test.afterAll(async () => {
    await server.stop()
  })

  test('draws the workspace with the files and the plex while a plugin never becomes ready', async ({
    page,
  }) => {
    const errors: string[] = []
    page.on('pageerror', (error) => errors.push(error.message))
    await page.goto(server.url)
    await expectWorkspace(page)
    expect(errors).toStrictEqual([])
  })
})

test.describe('Start-up with the updater and the index stats', () => {
  test('draws the tab bar and the panes with the title bar items of both', async ({ page }) => {
    const errors: string[] = []
    page.on('pageerror', (error) => errors.push(error.message))
    await page.goto('/')
    await expectWorkspace(page)
    const bar = page.locator('[data-title-bar]')
    await expect(bar.locator('[data-tasks-status]')).toBeVisible()
    await expect(bar.locator('.index-stats__trigger')).toBeVisible()
    expect(errors).toStrictEqual([])
  })
})
