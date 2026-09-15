import type { Placement } from './placement'

/**
 * A stand-in for a real mind map. It sets coordinates and inherits everything
 * else: routing, limits, movement.
 */
export const ring: Placement = {
  name: 'ring',
  place(seating) {
    const around = Object.values(seating).flat()
    return around.map((node, index) => {
      const angle = (index / around.length) * Math.PI * 2 - Math.PI / 2
      return {
        ...node,
        x: Math.cos(angle) * 300,
        y: Math.sin(angle) * 190,
        width: 144,
        height: 36,
        order: index,
        opacity: 1,
      }
    })
  },
}
