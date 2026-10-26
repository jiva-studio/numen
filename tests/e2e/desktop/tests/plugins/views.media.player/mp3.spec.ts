import { test, expect, type Locator } from 'playwright/test'
import { openVaultFile, openWorkspace } from '../../support/workspace.js'

/** What the player says where it cannot play a recording, in any of its forms. */
const FAILURE = /failed to play|unreadable|not supported|loading stopped/i

/** The length of `fixtures/talk.mp3`, a VBR mp3 with a Xing header. */
const LENGTH_MS = 187_000

/**
 * The length the player's bar runs to, which is the length the engine reads: Chromium reads 187.0 s
 * and WebKit 187.2 s, the frame count of the Xing header. A length read off the first frames is far
 * shorter.
 */
const getBarLength = async (bar: Locator): Promise<number> => Number(await bar.getAttribute('max'))

test.describe('An mp3 recording', () => {
  test('shows its whole length, seeks, and says nothing failed while it loads', async ({
    page,
  }) => {
    // Every text drawn in the page and in the shadow roots the plugins are drawn in is checked as
    // it changes, so a failure shown for a moment is kept.
    await page.addInitScript((source) => {
      const failure = new RegExp(source, 'i')
      const failures: string[] = []
      ;(window as unknown as { shownFailures: string[] }).shownFailures = failures
      const watch = (root: Node) =>
        new MutationObserver(() => {
          const found = (root.textContent ?? '').match(failure)
          if (found) failures.push(found[0])
        }).observe(root, { subtree: true, childList: true, characterData: true })
      const attachShadow = Element.prototype.attachShadow
      Element.prototype.attachShadow = function (init) {
        const root = attachShadow.call(this, init)
        watch(root)
        return root
      }
      watch(document)
    }, FAILURE.source)

    await openWorkspace(page)
    const tab = await openVaultFile(page, 'assets', 'talk.mp3')
    const player = tab.getByRole('group', { name: 'Recording' })
    const bar = player.getByRole('slider')

    await expect(player).toContainText('3:07')
    await expect.poll(() => getBarLength(bar)).toBeCloseTo(LENGTH_MS, -3)

    await bar.fill('120000')
    await player.getByRole('button', { name: /^Play/ }).click()
    await expect(player).toContainText(/2:0[0-9]/)
    await expect.poll(() => getBarLength(bar)).toBeCloseTo(LENGTH_MS, -3)

    const failures = await page.evaluate(
      () => (window as unknown as { shownFailures: string[] }).shownFailures,
    )
    expect(failures).toEqual([])
  })

  test('shows its own length while another recording plays', async ({ page }) => {
    await openWorkspace(page)
    const talk = await openVaultFile(page, 'assets', 'talk.mp3')
    const talkPlayer = talk.getByRole('group', { name: 'Recording' })
    await talkPlayer.getByRole('button', { name: /^Play/ }).click()
    await expect(talkPlayer.getByRole('button', { name: /^Pause/ })).toBeVisible()

    const lecture = await openVaultFile(page, 'assets', 'lecture.wav')
    const lecturePlayer = lecture.getByRole('group', { name: 'Recording' })

    await expect(lecturePlayer.getByRole('slider')).toHaveAttribute('max', '6000')
    await expect(lecturePlayer).toContainText(/0:00\s*0:06/)
    await expect(lecturePlayer.getByRole('button', { name: /^Play/ })).toBeVisible()
  })
})
