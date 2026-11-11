/**
 * Where the link inspector popover is placed in its container.
 */
export interface PopoverPlacementOptions {
  readonly at: { readonly x: number; readonly y: number }
  readonly popoverSize: { readonly width: number; readonly height: number }
  readonly containerSize: { readonly width: number; readonly height: number }
  readonly windowSize?: { readonly width: number; readonly height: number }
  readonly margin?: number
  readonly gap?: number
}

export interface PopoverPlacementResult {
  readonly left: number
  readonly top: number
}

function getLeftPosition(
  atX: number,
  width: number,
  ...[containerW, margin]: [containerW: number, margin: number]
): number {
  const idealLeft = atX - width / 2
  const maxLeft = Math.max(margin, containerW - width - margin)
  return Math.max(margin, Math.min(idealLeft, maxLeft))
}

function getTopPosition(
  atY: number,
  height: number,
  ...[containerH, margin, gap]: [containerH: number, margin: number, gap: number]
): number {
  const aboveTop = atY - height - gap
  if (aboveTop >= margin) return aboveTop

  const belowTop = atY + gap
  const maxTop = Math.max(margin, containerH - height - margin)
  if (belowTop + height <= containerH - margin) return belowTop

  const clampedAbove = Math.max(margin, Math.min(aboveTop, maxTop))
  const clampedBelow = Math.max(margin, Math.min(belowTop, maxTop))
  return atY < containerH / 2 ? clampedBelow : clampedAbove
}

export function resolvePopoverPosition({
  at: point,
  popoverSize,
  containerSize,
  windowSize = { width: 800, height: 600 },
  margin = 12,
  gap = 12,
}: PopoverPlacementOptions): PopoverPlacementResult {
  const width = popoverSize.width || 320
  const height = popoverSize.height || 160
  const containerW = containerSize.width || windowSize.width
  const containerH = containerSize.height || windowSize.height

  return {
    left: Math.round(getLeftPosition(point.x, width, containerW, margin)),
    top: Math.round(getTopPosition(point.y, height, containerH, margin, gap)),
  }
}
