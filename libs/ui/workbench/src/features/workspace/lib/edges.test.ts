/** The panes at the top edge and the top corners, for each arrangement of the screen. */
import { describe, expect, it } from 'vitest'
import { NO_STRIP_EDGE, findTopPanes, getStripEdge } from './edges'
import { createWorkspace, split, stack } from '../fixtures/build'

describe('findTopPanes', () => {
  it('gives one pane the top edge and both corners', () => {
    const workspace = createWorkspace(stack('ledger', 'harvest'))

    expect(findTopPanes(workspace)).toStrictEqual({
      top: ['ledger'],
      left: 'ledger',
      right: 'ledger',
    })
  })

  it('gives every pane of a row the top edge, the first the left corner and the last the right', () => {
    const workspace = createWorkspace(
      split('row', [stack('ledger', 'harvest'), stack('map', 'orchard'), stack('chat', 'agent')]),
    )

    expect(findTopPanes(workspace)).toStrictEqual({
      top: ['ledger', 'map', 'chat'],
      left: 'ledger',
      right: 'chat',
    })
  })

  it('gives the first pane of a column the top edge and both corners', () => {
    const workspace = createWorkspace(
      split('column', [stack('ledger', 'harvest'), stack('log', 'rain')]),
      'vertical',
    )

    expect(findTopPanes(workspace)).toStrictEqual({
      top: ['ledger'],
      left: 'ledger',
      right: 'ledger',
    })
  })

  it('leaves out a pane below another in the right column of a row', () => {
    const workspace = createWorkspace(
      split('row', [
        stack('ledger', 'harvest'),
        split('aside', [stack('chat', 'agent'), stack('log', 'rain')]),
      ]),
    )

    expect(findTopPanes(workspace)).toStrictEqual({
      top: ['ledger', 'chat'],
      left: 'ledger',
      right: 'chat',
    })
  })

  it('takes the corners from the row at the top of a column', () => {
    const workspace = createWorkspace(
      split('column', [
        split('upper', [stack('ledger', 'harvest'), stack('map', 'orchard')]),
        stack('log', 'rain'),
      ]),
      'vertical',
    )

    expect(findTopPanes(workspace)).toStrictEqual({
      top: ['ledger', 'map'],
      left: 'ledger',
      right: 'map',
    })
  })

  it('gives no pane at the bottom of the window the top edge', () => {
    const workspace = createWorkspace(
      split('column', [
        split('upper', [stack('ledger', 'harvest'), stack('map', 'orchard')]),
        split('lower', [stack('log', 'rain'), stack('chat', 'agent')]),
      ]),
      'vertical',
    )

    const panes = findTopPanes(workspace)

    expect(panes.top).not.toContain('log')
    expect(panes.top).not.toContain('chat')
  })

  it('follows the left and the right column down to the panes at the top', () => {
    const workspace = createWorkspace(
      split('row', [
        split('left', [stack('ledger', 'harvest'), stack('log', 'rain')]),
        stack('map', 'orchard'),
        split('right', [stack('chat', 'agent'), stack('notes', 'pruning')]),
      ]),
    )

    expect(findTopPanes(workspace)).toStrictEqual({
      top: ['ledger', 'map', 'chat'],
      left: 'ledger',
      right: 'chat',
    })
  })

  it('reads three levels of splits', () => {
    const workspace = createWorkspace(
      split('row', [
        stack('ledger', 'harvest'),
        split('aside', [
          split('upper', [stack('map', 'orchard'), stack('chat', 'agent')]),
          stack('log', 'rain'),
        ]),
      ]),
    )

    expect(findTopPanes(workspace)).toStrictEqual({
      top: ['ledger', 'map', 'chat'],
      left: 'ledger',
      right: 'chat',
    })
  })

  it('gives the room of each corner to the strip in it, and none to a strip below the top', () => {
    const panes = findTopPanes(
      createWorkspace(
        split('row', [
          stack('ledger', 'harvest'),
          split('aside', [stack('chat', 'agent'), stack('log', 'rain')]),
        ]),
      ),
    )
    const edge = { leftInset: 80, rightInset: 12 }

    expect(getStripEdge(panes, edge, 'ledger')).toStrictEqual({
      isDraggable: true,
      leftInset: 80,
      rightInset: 0,
    })
    expect(getStripEdge(panes, edge, 'chat')).toStrictEqual({
      isDraggable: true,
      leftInset: 0,
      rightInset: 12,
    })
    expect(getStripEdge(panes, edge, 'log')).toBe(NO_STRIP_EDGE)
    expect(getStripEdge(panes, undefined, 'ledger')).toBe(NO_STRIP_EDGE)
  })

  it('names the empty pane of a window with no tabs', () => {
    const workspace = createWorkspace(stack('empty'))

    expect(findTopPanes(workspace)).toStrictEqual({ top: ['empty'], left: 'empty', right: 'empty' })
  })
})
