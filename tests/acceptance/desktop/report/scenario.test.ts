import { describe, expect, it } from 'vitest'
import { getFeatureKey } from './scenario.js'

describe('getFeatureKey', () => {
  it('keeps the folders of a feature apart from the dashes of its name', () => {
    expect(getFeatureKey('features/a-b/c.feature')).not.toBe(
      getFeatureKey('features/a/b-c.feature'),
    )
  })

  it('gives a key that is safe in a link and a folder name', () => {
    expect(getFeatureKey('features/editor/plex/plex drag.feature')).toBe(
      'features.editor.plex.plex-drag',
    )
  })
})
