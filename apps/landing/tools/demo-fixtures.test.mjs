/** The plex the demo draws is fixed data: every curve joins two notes it draws, and every note fits the drawing. */
import assert from 'node:assert/strict'
import { readFileSync } from 'node:fs'
import test from 'node:test'

const layout = JSON.parse(
  readFileSync(new URL('../src/components/demo/lib/plex.json', import.meta.url), 'utf8'),
)
const WIDTH = 630
const HEIGHT = 338

test('the plex draws one focus note', () => {
  assert.equal(layout.nodes.filter((one) => one.relation === 'focus').length, 1)
})

test('each curve of the plex joins two notes it draws', () => {
  const paths = new Set(layout.nodes.map((one) => one.path))
  for (const edge of layout.edges) {
    const [from, to] = edge.id.split('->')
    assert.ok(paths.has(from) && paths.has(to), `${edge.id} joins a note that is not drawn`)
  }
})

test('each note of the plex fits the drawing', () => {
  for (const node of layout.nodes) {
    assert.ok(
      node.x - node.width / 2 >= 0 && node.x + node.width / 2 <= WIDTH,
      `${node.path} is cut at the sides`,
    )
    assert.ok(
      node.y - node.height / 2 >= 0 && node.y + node.height / 2 <= HEIGHT,
      `${node.path} is cut at the top or bottom`,
    )
  }
})
