/** The line the pointer is on stays raised through the frames of a move, as long as the line is there. */
import { effectScope, nextTick, reactive } from 'vue'
import { describe, expect, it } from 'vitest'
import { useEdgeLines } from './lines'
import { arrangePlex } from '../../lib/arrange'
import { build } from '../../fixtures/build'

const wide = arrangePlex(build('Hub', { child: 3 }))
const narrow = arrangePlex(build('Hub', { child: 1 }))

const setUp = () => {
  const props = reactive({ frame: wide, highlightLinks: false })
  const lines = effectScope().run(() => useEdgeLines(props, 'u', { node: null, names: null }))!
  return { props, lines }
}

describe('the line the pointer is on', () => {
  it('stays raised when the frame changes and the line is still there', async () => {
    const { props, lines } = setUp()
    const first = lines.lines.value[0]!
    lines.setOver(first.pair)

    props.frame = { ...wide }
    await nextTick()

    expect(lines.isRaised(lines.lines.value[0]!)).toBe(true)
  })

  it('is lowered when the frame no longer has the line', async () => {
    const { props, lines } = setUp()
    const gone = lines.lines.value.at(-1)!
    lines.setOver(gone.pair)

    props.frame = narrow
    await nextTick()

    expect(lines.lines.value.every((line) => !lines.isRaised(line))).toBe(true)
  })
})
