import { test, expect } from '../../support/test.js'
import { startOwnServer, type InstalledPlugin, type OwnServer } from '../../support/server.js'
import { expectWorkspaceDrawn as expectWorkspace } from '../../support/workspace.js'

/**
 * An agent plugin whose process never says it is ready. The core asks every agent plugin which
 * programs it runs when the window reads the settings, and waits for the answer.
 */
const stuckAgent: InstalledPlugin = {
  id: 'md.numen.test.stuck',
  files: {
    'manifest.json': JSON.stringify({
      id: 'md.numen.test.stuck',
      name: 'Stuck agent',
      version: '2026.10.1',
      protocol: 1,
      description: 'An agent plugin that never answers',
      capabilities: ['ai.agent.provider'],
      targets: ['desktop.editor'],
      contributes: { category: 'agents' },
      permissions: [],
      backend: { entrypoint: 'bin/stuck' },
      services: ['numen.v1.AgentService'],
    }),
    'bin/stuck': '#!/bin/sh\nexec sleep 3600\n',
  },
}

test.describe('Start-up while a plugin does not start', () => {
  let server: OwnServer

  test.beforeAll(async () => {
    server = await startOwnServer([stuckAgent])
  })

  test.afterAll(async () => {
    await server.stop()
  })

  test('draws the workspace with the files and the plex while an agent plugin never becomes ready', async ({
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
    await expect(bar.locator('button:has(.index-stats__figure)')).toBeVisible()
    expect(errors).toStrictEqual([])
  })
})
