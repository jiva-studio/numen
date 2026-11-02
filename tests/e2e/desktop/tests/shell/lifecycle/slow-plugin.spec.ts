import { test, expect, type Page } from '../../support/test.js'
import { startOwnServer, type InstalledPlugin, type OwnServer } from '../../support/server.js'

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

/** The window is usable within these seconds of loading, whatever a plugin does. */
const USABLE_WITHIN_MS = 5_000

/** The workspace is drawn: a tab bar, the panes, the plex graph of the opening note and the files. */
const expectWorkspace = async (page: Page): Promise<void> => {
  const within = { timeout: USABLE_WITHIN_MS }
  await expect(page.getByRole('tablist').first()).toBeVisible(within)
  await expect(page.locator('section[data-workspace-pane]')).toHaveCount(2, within)
  for (const pane of await page.locator('section[data-workspace-pane]').all()) {
    const box = await pane.boundingBox()
    expect(box?.width).toBeGreaterThan(0)
    expect(box?.height).toBeGreaterThan(0)
  }
  await expect(page.getByRole('group', { name: 'Neighbourhood' })).toBeVisible(within)
  await expect(page.getByRole('tab', { name: /^Files/ })).toBeVisible(within)
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
  test('draws the tab bar and the panes with the status bar items of both', async ({ page }) => {
    const errors: string[] = []
    page.on('pageerror', (error) => errors.push(error.message))
    await page.goto('/')
    await expectWorkspace(page)
    const bar = page.getByRole('region', { name: 'Status Bar' })
    await expect(bar).toBeVisible()
    expect(errors).toStrictEqual([])
  })
})
