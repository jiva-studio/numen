import { test, expect, type APIRequestContext } from '../../support/test.js'
import { NEW_RELEASE } from '../../../scripts/stand-ins.mjs'
import { openWorkspace } from '../../support/workspace.js'
import { getStandInsOrigin, writePluginSetting } from '../../support/media.js'

const UPDATER = 'md.numen.tools.updater'

/** Has the stand-in release feed name a release newer than any build. */
const publishNewRelease = async (request: APIRequestContext, origin: string): Promise<void> => {
  const answer = await request.put(`${origin}/stand-in/release`, { data: NEW_RELEASE })
  expect(answer.ok()).toBe(true)
}

/** Has the stand-in release feed name the old release again. */
const withdrawNewRelease = async (request: APIRequestContext, origin: string): Promise<void> => {
  const answer = await request.delete(`${origin}/stand-in/release`)
  expect(answer.ok()).toBe(true)
}

/** How many times the updater asked the stand-in for the release manifest. */
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

  test.beforeEach(async ({ request }) => {
    origin = await getStandInsOrigin(request)
  })

  test.afterEach(async ({ request }) => {
    await withdrawNewRelease(request, origin)
    await writePluginSetting(request, UPDATER, 'channel', 'stable')
  })

  test('announces a newer release the feed names when the window opens', async ({
    page,
    request,
  }) => {
    await publishNewRelease(request, origin)

    await openWorkspace(page)

    const indicator = page.getByRole('button', { name: 'Application update available' })
    await expect(indicator).toHaveText(`v${NEW_RELEASE.version}`)
    await indicator.click()
    await expect(page.getByText('Update available')).toBeVisible()
    await expect(page.getByRole('button', { name: 'Download' })).toBeVisible()
  })

  test('checks the feed again when the channel changes, and announces what it finds', async ({
    page,
    request,
  }) => {
    await openWorkspace(page)
    await expect.poll(() => countManifestRequests(request, origin)).toBeGreaterThan(0)
    const indicator = page.getByRole('button', { name: 'Application update available' })
    await expect(indicator).toHaveCount(0)
    const requestCount = await countManifestRequests(request, origin)
    await publishNewRelease(request, origin)

    await writePluginSetting(request, UPDATER, 'channel', 'beta')

    await expect(indicator).toHaveText(`v${NEW_RELEASE.version}`)
    expect(await countManifestRequests(request, origin)).toBeGreaterThan(requestCount)

    await withdrawNewRelease(request, origin)
    await writePluginSetting(request, UPDATER, 'channel', 'stable')

    await expect(indicator).toHaveCount(0)
  })
})
