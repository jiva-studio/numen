import fs from 'node:fs'
import type { TestResult } from 'playwright/types/testReporter'
import { INPUTS, type InputKind, type RecordedInput } from '../support/inputs.js'

export type VideoInput = {
  readonly kind: InputKind
  readonly t: number
  readonly x: number
  readonly y: number
  readonly w?: number
  readonly h?: number
  readonly key?: string
  readonly mods?: readonly string[]
  readonly isField?: boolean
  readonly text?: string
  readonly isAppend?: boolean
}

const round = (value: number, digits: number): number => Number(value.toFixed(digits))

const getShare = (value: number, size: number): number => round(value / size, 4)

const mapInput = (input: RecordedInput, videoStart: number): VideoInput => {
  const { page: _page, at, x, y, w, h, width, height, ...rest } = input
  const size =
    w === undefined || h === undefined ? {} : { w: getShare(w, width), h: getShare(h, height) }
  return {
    ...rest,
    t: round(Math.max(at - videoStart, 0) / 1000, 3),
    x: getShare(x, width),
    y: getShare(y, height),
    ...size,
  }
}

export const mapInputs = (
  inputs: readonly RecordedInput[],
  videoStart: number,
  page = 0,
): VideoInput[] =>
  inputs
    .filter((input) => input.page === page && input.width > 0 && input.height > 0)
    .map((input) => mapInput(input, videoStart))
    .sort((a, b) => a.t - b.t)

const readBody = (attachment: TestResult['attachments'][number]): string | undefined => {
  if (attachment.body) return attachment.body.toString('utf8')
  return attachment.path && fs.existsSync(attachment.path)
    ? fs.readFileSync(attachment.path, 'utf8')
    : undefined
}

export const readInputs = (result: TestResult): RecordedInput[] => {
  const attachment = result.attachments.find((one) => one.name === INPUTS)
  const body = attachment && readBody(attachment)
  if (!body) return []
  return JSON.parse(body) as RecordedInput[]
}
