import { afterEach, describe, expect, it } from 'vitest'
import { branch, pane, type Workspace } from '../lib/node'
import { getTabPaneRect } from './paneRect'

const WORKSPACE: Workspace = {
  root: branch('root', [pane('left', ['reef', 'tide']), pane('aside', ['atlas'])], [0.6, 0.4]),
  axis: 'horizontal',
  focus: 'left',
}

const drawPane = (id: string, box: { x: number; y: number; width: number; height: number }) => {
  const element = document.createElement('div')
  element.setAttribute('data-workspace-pane', id)
  element.getBoundingClientRect = () =>
    ({ left: box.x, top: box.y, width: box.width, height: box.height }) as DOMRect
  document.body.append(element)
}

afterEach(() => {
  document.body.replaceChildren()
})

describe('the box of the pane a tab is in', () => {
  it('is the box of the drawn pane that holds the tab, in viewport coordinates', () => {
    drawPane('left', { x: 0, y: 40, width: 600, height: 800 })
    drawPane('aside', { x: 600, y: 40, width: 400, height: 800 })

    expect(getTabPaneRect(WORKSPACE, 'atlas')).toStrictEqual({
      x: 600,
      y: 40,
      width: 400,
      height: 800,
    })
  })

  it('is null for a tab no pane holds, or a pane that is not drawn', () => {
    drawPane('left', { x: 0, y: 40, width: 600, height: 800 })

    expect(getTabPaneRect(WORKSPACE, 'chart')).toBeNull()
    expect(getTabPaneRect(WORKSPACE, 'atlas')).toBeNull()
  })
})
