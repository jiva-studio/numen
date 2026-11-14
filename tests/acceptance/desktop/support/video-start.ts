/** The switch of a run with video, and the mark of the moment the video of a scenario starts. */
import type { TestInfo } from 'playwright/test'

/** The annotation that holds the epoch milliseconds at which the page, and its video, started. */
export const VIDEO_START = 'video-start'

export const isVideoOn = (): boolean => Boolean(process.env.PLAYWRIGHT_VIDEO)

export const markVideoStart = (testInfo: TestInfo): void => {
  testInfo.annotations.push({ type: VIDEO_START, description: String(Date.now()) })
}
