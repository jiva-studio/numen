/**
 * The fixture generator, which plays the application's part in the stories.
 * A walk through it has to keep producing neighbourhoods a plex will accept.
 */
import { describe, expect, it } from 'vitest'
import { around, build, type Named } from './build'
import { arrangePlex } from '../arrange'

const COUNTS = { parent: 3, child: 6, jump: 3, sibling: 3 }

/** Choose a node and hand back the neighbourhood around it, as a story does. */
function step(current: ReturnType<typeof build>, id: string) {
  const chosen = current.nodes.find((node) => node.id === id)!
  const was = current.nodes.find((node) => node.seat === 'focus')!
  const from: Named = { id: was.id, title: was.title }
  return around({ id: chosen.id, title: chosen.title }, from, COUNTS)
}

describe('walking', () => {
  it('never seats the same node twice, however far it goes', () => {
    let here = build('A node', COUNTS)

    // Down, then back up, then down again: the node you came from is already
    // seated when you arrive back at it, and must not be seated a second time.
    for (let i = 0; i < 8; i++) {
      const child = here.nodes.find((node) => node.seat === 'child')!
      here = step(here, child.id)
      expect(() => arrangePlex(here)).not.toThrow()

      const parent = here.nodes.find((node) => node.seat === 'parent')!
      here = step(here, parent.id)
      expect(() => arrangePlex(here)).not.toThrow()
    }
  })

  it('keeps the identifier of the node it came from', () => {
    const start = build('A node', COUNTS)
    const child = start.nodes.find((node) => node.seat === 'child')!
    const next = step(start, child.id)

    expect(next.nodes.find((node) => node.seat === 'focus')?.id).toBe(child.id)
    expect(next.nodes.some((node) => node.id === 'focus' && node.seat === 'parent')).toBe(
      true,
    )
  })

  it('gives every node a name from the pool rather than a number', () => {
    const built = build('A node', COUNTS)
    for (const node of built.nodes) {
      expect(node.title).not.toMatch(/^(Parent|Child|Jump|Sibling) \d+$/)
    }
  })
})
