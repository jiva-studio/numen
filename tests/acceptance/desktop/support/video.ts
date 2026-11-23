import fs from 'node:fs'
import type { Page, TestInfo, ViewportSize } from 'playwright/test'
import type { Recording } from '../drivers/scenario.js'
import { attachInputs, recordInputs, type RecordedInput } from './inputs.js'

export const VIDEO_START = 'video-start'
export const PLATFORM = 'platform'

const DEFAULT_SIZE: ViewportSize = { width: 1280, height: 860 }

export const isVideoOn = (): boolean => Boolean(process.env.PLAYWRIGHT_VIDEO)

export type VideoRecorder = {
  readonly videoRecording: Recording
  readonly attachVideos: () => Promise<void>
}

export const createVideoRecorder = (
  testInfo: TestInfo,
  viewport: ViewportSize | null,
): VideoRecorder => {
  const pages: Page[] = []
  const inputs: RecordedInput[] = []
  const videoRecording: Recording = {
    contextOptions: {
      recordVideo: {
        dir: testInfo.outputPath('window-videos'),
        size: viewport ?? DEFAULT_SIZE,
      },
    },
    onPage: async (page) => {
      if (pages.length === 0) markVideoStart(testInfo)
      await recordInputs(page, pages.length, inputs)
      pages.push(page)
    },
  }
  return {
    videoRecording,
    attachVideos: async () => {
      await attachInputs(testInfo, inputs)
      await attachVideos(testInfo, pages)
    },
  }
}

const markVideoStart = (testInfo: TestInfo): void => {
  testInfo.annotations.push(
    { type: VIDEO_START, description: String(Date.now()) },
    { type: PLATFORM, description: process.platform },
  )
}

const attachVideos = async (testInfo: TestInfo, pages: readonly Page[]): Promise<void> => {
  for (const page of pages) {
    const path = await page
      .video()
      ?.path()
      .catch(() => undefined)
    if (path && fs.existsSync(path))
      await testInfo.attach('video', { path, contentType: 'video/webm' })
  }
}
