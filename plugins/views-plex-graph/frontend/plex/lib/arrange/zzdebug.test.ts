import { it } from 'vitest'
import { arrangePlex } from './arrange'
import { packOptions } from './layout/crowding'
import { resolveOptions } from './options'
import { build } from '../../fixtures/build'
it('dbg', () => {
  const out: string[] = []
  for (const width of [440, 480, 500, 520, 540, 560, 600]) {
    const viewport = { width, height: 1000 }
    const packed = packOptions(resolveOptions({ viewport }), 0)
    const frame = arrangePlex(build('A note', { parent: 4, child: 12, jump: 4, sibling: 4 }), { options: { ...packed, viewport: undefined } })
    const out2 = frame.nodes.filter((n) => Math.abs(n.x) + n.width / 2 > width / 2 - 16).map((n) => n.id.replace('focus/', ''))
    out.push(`${width}: ${out2.join(',')}`)
  }
  console.log('RESULT ' + out.join(' | '))
})
