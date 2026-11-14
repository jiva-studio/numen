/** In a run with video, each step shows its text over the page before it runs, so the video says which step it shows. */
import { BeforeStep } from './bdd.js'
import { isVideoOn } from './video-start.js'

/** How long the text stays on the page; Playwright adds a fade of 300 ms and waits for both. */
const CHAPTER_MS = 700

BeforeStep(async ({ page, $bddContext, $step }) => {
  if (!isVideoOn()) return
  const title = $bddContext.bddTestData.steps[$bddContext.stepIndex]?.textWithKeyword ?? $step.title
  await page.screencast.showChapter(title, { duration: CHAPTER_MS }).catch(() => {
    // The text is decoration of the video: a page that navigates while it shows must not fail the scenario.
  })
})
