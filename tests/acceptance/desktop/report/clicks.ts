import fs from 'node:fs'
import type { TestResult } from 'playwright/types/testReporter'
import { CLICKS, type RecordedClick } from '../support/clicks.js'

export type VideoClick = {
  readonly t: number
  readonly x: number
  readonly y: number
}

const round = (value: number, digits: number): number => Number(value.toFixed(digits))

export const mapClicks = (
  clicks: readonly RecordedClick[],
  videoStart: number,
  page = 0,
): VideoClick[] =>
  clicks
    .filter((click) => click.page === page && click.width > 0 && click.height > 0)
    .map((click) => ({
      t: round(Math.max(click.at - videoStart, 0) / 1000, 3),
      x: round(click.x / click.width, 4),
      y: round(click.y / click.height, 4),
    }))
    .sort((a, b) => a.t - b.t)

const readBody = (attachment: TestResult['attachments'][number]): string | undefined => {
  if (attachment.body) return attachment.body.toString('utf8')
  return attachment.path && fs.existsSync(attachment.path)
    ? fs.readFileSync(attachment.path, 'utf8')
    : undefined
}

export const readClicks = (result: TestResult): RecordedClick[] => {
  const attachment = result.attachments.find((one) => one.name === CLICKS)
  const body = attachment && readBody(attachment)
  if (!body) return []
  return JSON.parse(body) as RecordedClick[]
}
