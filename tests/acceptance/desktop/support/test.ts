/**
 * The `test` of the acceptance suite: the `test` of playwright-bdd that `bddgen` requires, and the
 * person driver, which ends the scenario by removing its world, the world's server and its window,
 * also when it failed; a worker that is killed leaves that to the helper of drivers/reaper.ts.
 */
import { test as bdd } from 'playwright-bdd'
import { Person } from '../drivers/person.js'
import { isVideoOn, markVideoStart } from './video-start.js'

export const test = bdd.extend<{ person: Person }>({
  // Playwright starts the video of a page while it creates the page, so the moment the page is
  // ready is the start of the video, within the time of one call to the browser.
  page: async ({ page }, use, testInfo) => {
    if (isVideoOn()) markVideoStart(testInfo)
    await use(page)
  },
  person: async ({ page, viewport }, use, testInfo) => {
    const person = new Person(page, viewport)
    try {
      await use(person)
      const log = person.readServerLog()
      if (log !== undefined && testInfo.status !== testInfo.expectedStatus) {
        await testInfo.attach('server.log', { body: log, contentType: 'text/plain' })
      }
    } finally {
      await person.removeNotes()
    }
  },
})
