import type { Task } from '../types'

type Placed = Pick<Task, 'line' | 'raw'>

const lineOf = (lines: readonly string[], line: number): string | undefined =>
  lines[line - 1]?.replace(/\r$/, '')

/**
 * The 1-based line of a task in the prose of its note, which the editor counts from: the note reads
 * without its frontmatter, and a task holds the line of the whole file.
 *
 * The frontmatter is the number of lines the file has above the prose. The one that puts the most
 * of the tasks of the note on the lines that say what they are is taken.
 */
export function proseLine(body: string, tasks: readonly Placed[], target: Placed): number {
  const lines = body.split('\n')
  let best = 0
  let bestScore = -1
  for (let hidden = 0; hidden < target.line; hidden += 1) {
    if (lineOf(lines, target.line - hidden) !== target.raw) continue
    const score = tasks.filter((one) => lineOf(lines, one.line - hidden) === one.raw).length
    if (score > bestScore) {
      best = hidden
      bestScore = score
    }
  }
  return target.line - best
}
