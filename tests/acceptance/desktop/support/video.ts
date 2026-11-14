/**
 * A run with video (PLAYWRIGHT_VIDEO). The window of a scenario is a page of a browser its world
 * launches (drivers/scenario.ts), which Playwright's `video` option does not reach: the world
 * records it, and the person fixture attaches the video and marks when it started.
 */
import type { Page, TestInfo, ViewportSize } from 'playwright/test'
import type { Recording } from '../drivers/scenario.js'

/** The annotation that holds the epoch milliseconds at which the video of the scenario starts. */
export const VIDEO_START = 'video-start'

const DEFAULT_SIZE: ViewportSize = { width: 1280, height: 860 }

export const isVideoOn = (): boolean => Boolean(process.env.PLAYWRIGHT_VIDEO)

export type VideoRecorder = {
  readonly recording: Recording
  readonly attachVideos: () => Promise<void>
}

export const createVideoRecorder = (
  testInfo: TestInfo,
  viewport: ViewportSize | null,
): VideoRecorder => {
  const pages: Page[] = []
  const recording: Recording = {
    contextOptions: {
      recordVideo: {
        dir: testInfo.outputPath('window-videos'),
        size: viewport ?? DEFAULT_SIZE,
      },
    },
    // A page records from its creation, so the moment it is ready is the start of its video,
    // within the time of one call to the browser.
    onPage: (page) => {
      if (pages.length === 0) markVideoStart(testInfo)
      pages.push(page)
    },
  }
  return { recording, attachVideos: () => attachVideos(testInfo, pages) }
}

const markVideoStart = (testInfo: TestInfo): void => {
  testInfo.annotations.push({ type: VIDEO_START, description: String(Date.now()) })
}

/** Attaches the video of each window, the first first; a window is written out once its browser closed. */
const attachVideos = async (testInfo: TestInfo, pages: readonly Page[]): Promise<void> => {
  for (const page of pages) {
    const video = page.video()
    if (video)
      await testInfo.attach('video', { path: await video.path(), contentType: 'video/webm' })
  }
}
