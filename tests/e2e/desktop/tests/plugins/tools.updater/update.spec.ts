import { test, expect, type APIRequestContext } from '../../support/test.js'
import { NEW_RELEASE } from '../../../scripts/stand-ins.mjs'
import { openWorkspace } from '../../support/workspace.js'
import { getStandInsOrigin, writePluginSetting } from '../../support/media.js'

const UPDATER = 'md.numen.tools.updater'

const publishNewRelease = async (request: APIRequestContext, origin: string): Promise<void> => {
  const answer = await request.put(`${origin}/stand-in/release`, { data: NEW_RELEASE })
  expect(answer.ok()).toBe(true)
}

const withdrawNewRelease = async (request: APIRequestContext, origin: string): Promise<void> => {
  const answer = await request.delete(`${origin}/stand-in/release`)
  expect(answer.ok()).toBe(true)
}

const countManifestRequests = async (
  request: APIRequestContext,
  origin: string,
): Promise<number> => {
  const counts = (await (await request.get(`${origin}/stand-in/requests`)).json()) as Record<
    string,
    number
  >
  return counts['/latest/latest.json'] ?? 0
}

test.describe('the update indicator', () => {
  let origin: string
  let endpoint: string

  test.beforeEach(async ({ request }) => {
    origin = await getStandInsOrigin(request)
    endpoint = `${origin}/latest/latest.json`
  })

  test.afterEach(async ({ request }) => {
    await withdrawNewRelease(request, origin)
    await writePluginSetting(request, UPDATER, 'channel', 'custom')
    await writePluginSetting(request, UPDATER, 'endpoint', endpoint)
  })

  test('announces a newer release the feed names when the window opens', async ({
    page,
    request,
  }) => {
    await publishNewRelease(request, origin)

    await openWorkspace(page)

    const indicator = page.getByRole('button', { name: /^Update available: v/ })
    await expect(indicator).toHaveText(`v${NEW_RELEASE.version}`)
    await indicator.click()
    await expect(page.getByText('Update available')).toBeVisible()
    await expect(
      page
        .getByRole('dialog', { name: 'Update available' })
        .getByRole('button', { name: 'Download' }),
    ).toBeVisible()
  })

  test('checks the feed again when the release feed address changes, and announces what it finds', async ({
    page,
    request,
  }) => {
    await openWorkspace(page)
    await expect.poll(() => countManifestRequests(request, origin)).toBeGreaterThan(0)
    const indicator = page.getByRole('button', { name: /^Update available: v/ })
    await expect(indicator).toHaveCount(0)
    const requestCount = await countManifestRequests(request, origin)
    await publishNewRelease(request, origin)

    await writePluginSetting(request, UPDATER, 'endpoint', `${endpoint}?again=1`)

    await expect(indicator).toHaveText(`v${NEW_RELEASE.version}`)
    expect(await countManifestRequests(request, origin)).toBeGreaterThan(requestCount)

    await withdrawNewRelease(request, origin)
    await writePluginSetting(request, UPDATER, 'endpoint', endpoint)

    await expect(indicator).toHaveCount(0)
  })
})
