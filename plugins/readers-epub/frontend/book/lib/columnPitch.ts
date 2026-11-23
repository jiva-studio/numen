import type { Flow } from './spread'

export function getColumnPitch(flow: Flow): number {
  if (flow.columns <= 0) return 0
  return flow.width / flow.columns
}
