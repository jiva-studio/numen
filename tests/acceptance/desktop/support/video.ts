import type { Page, TestInfo, ViewportSize } from 'playwright/test'
import type { Recording } from '../drivers/scenario.js'
import { attachClicks, recordClicks, type RecordedClick } from './clicks.js'

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
  const clicks: RecordedClick[] = []
  const recording: Recording = {
    contextOptions: {
      recordVideo: {
        dir: testInfo.outputPath('window-videos'),
        size: viewport ?? DEFAULT_SIZE,
      },
    },
    onPage: async (page) => {
      if (pages.length === 0) markVideoStart(testInfo)
      await recordClicks(page, pages.length, clicks)
      pages.push(page)
    },
  }
  return {
    recording,
    attachVideos: async () => {
      await attachClicks(testInfo, clicks)
      await attachVideos(testInfo, pages)
    },
  }
}

const markVideoStart = (testInfo: TestInfo): void => {
  testInfo.annotations.push({ type: VIDEO_START, description: String(Date.now()) })
}

const attachVideos = async (testInfo: TestInfo, pages: readonly Page[]): Promise<void> => {
  for (const page of pages) {
    const video = page.video()
    if (video)
      await testInfo.attach('video', { path: await video.path(), contentType: 'video/webm' })
  }
}
