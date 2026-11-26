/**
 * The `test` of the acceptance suite: the `test` of playwright-bdd that `bddgen` requires, and the
 * person driver, which ends the scenario by removing its world, the world's server and its window,
 * also when it failed; a worker that is killed leaves that to the helper of drivers/reaper.ts.
 */
import { test as bdd } from 'playwright-bdd'
import { Person } from '../drivers/person.js'
import { MODEL_TAG, readSkipReason } from '../drivers/live.js'
import { createVideoRecorder, isVideoOn } from './video.js'

export const test = bdd.extend<{ person: Person }>({
  person: async ({ page, viewport }, use, testInfo) => {
    const needsModel = testInfo.tags.includes(MODEL_TAG)
    const skipReason = readSkipReason(testInfo.tags)
    if (skipReason !== undefined) testInfo.skip(true, skipReason)
    const recorder = isVideoOn() ? createVideoRecorder(testInfo, viewport) : undefined
    const person = new Person(page, viewport, recorder?.videoRecording, needsModel)
    try {
      await use(person)
      const log = person.readServerLog()
      if (log !== undefined && testInfo.status !== testInfo.expectedStatus) {
        await testInfo.attach('server.log', { body: log, contentType: 'text/plain' })
      }
    } finally {
      try {
        await person.removeNotes()
      } finally {
        await recorder?.attachVideos()
      }
    }
  },
})
