const EASING = { x1: 0.32, y1: 0.72, x2: 0, y2: 1 }

/** How far along the curve a turn is when that share of its time has passed. */
export function getEasedProgress(time: number): number {
  const { x1, y1, x2, y2 } = EASING
  const getBezier = (a: number, b: number, u: number) =>
    3 * a * (1 - u) ** 2 * u + 3 * b * (1 - u) * u ** 2 + u ** 3
  let low = 0
  let high = 1
  for (let step = 0; step < 24; step += 1) {
    const middle = (low + high) / 2
    if (getBezier(x1, x2, middle) < time) low = middle
    else high = middle
  }
  return getBezier(y1, y2, (low + high) / 2)
}
