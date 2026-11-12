import { it } from 'vitest'
import { arrangePlex } from './arrange'
import { packOptions } from './layout/crowding'
import { resolveOptions } from './options'
import { build } from '../../fixtures/build'
it('dbg', () => {
  const viewport = { width: 440, height: 1000 }
  const options = resolveOptions({ viewport })
  const packed = packOptions(options, 0)
  const frame = arrangePlex(build('A note', { parent: 4, child: 12, jump: 4, sibling: 4 }), { options: { ...packed, viewport: undefined } })
  console.log(JSON.stringify(frame.nodes.map((n) => [n.id.replace('focus/', ''), Math.round(n.x), Math.round(n.y), Math.round(n.width)])))
  console.log(viewport.width / 2 - 16)
})
