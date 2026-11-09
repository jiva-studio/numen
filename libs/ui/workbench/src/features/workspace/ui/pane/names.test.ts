/** The names a tab and its panel give each other. */
import { describe, expect, it } from 'vitest'
import { getPanelName, getTabName } from './names'

describe('the name of a tab and of its panel', () => {
  it('carry the pane and the place in its strip', () => {
    expect(getTabName('main', 2)).toBe('main-tab-2')
    expect(getPanelName('main', 2)).toBe('main-panel-2')
  })

  it('differ between panes, so they are unique to the document', () => {
    expect(getTabName('main', 0)).not.toBe(getTabName('aside', 0))
  })
})
