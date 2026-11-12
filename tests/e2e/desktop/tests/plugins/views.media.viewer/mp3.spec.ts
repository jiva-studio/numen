import { test, expect, type Locator } from '../../support/test.js'
import { openVaultFile, openWorkspace } from '../../support/workspace.js'

declare global {
  interface Window {
    shownFailures: string[]
  }
}

/** The message the player shows when it cannot play a recording, in any of its forms. */
const FAILURE = /failed to play|unreadable|not supported|loading stopped/i

/** The length of `fixtures/talk.mp3`, a VBR mp3 with a Xing header. */
const LENGTH_MS = 187_000

const getBarLength = async (bar: Locator): Promise<number> => Number(await bar.getAttribute('max'))

test.describe('An mp3 recording', () => {
  test('shows its whole length, seeks, and says nothing failed while it loads', async ({
    page,
  }) => {
    // Every text in the page and in the shadow roots of the plugins is checked as
    // it changes, so a failure message shown briefly is still recorded.
    await page.addInitScript((source) => {
      const failure = new RegExp(source, 'i')
      const failures: string[] = []
      window.shownFailures = failures
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

    const failures = await page.evaluate(() => window.shownFailures)
    expect(failures).toEqual([])
  })
})
