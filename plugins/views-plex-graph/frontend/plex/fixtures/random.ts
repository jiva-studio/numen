/** A seeded source of numbers in [0, 1), so a test that draws random cases draws the same ones every run. */
export function createRandom(seed: number): () => number {
  let state = seed
  return () => {
    state = (state * 1664525 + 1013904223) >>> 0
    return state / 2 ** 32
  }
}
