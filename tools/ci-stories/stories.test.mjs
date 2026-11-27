/**
 * The walk over the story files reads the tree, and names each story as Storybook does.
 */
import assert from 'node:assert/strict'
import test from 'node:test'
import { getStoryId, stories } from './stories.mjs'

test('a story id is the title and the export name in lower case and dashes', () => {
  assert.equal(getStoryId('Flash Cards/Window', 'CardsDue'), 'flash-cards-window--cards-due')
  assert.equal(getStoryId('Plex Headings Dropdown', 'Nesting'), 'plex-headings-dropdown--nesting')
})

/**
 * A walk that read the wrong folders finds nothing, so the stories it must have found are named:
 * one from the library, one from the shell, one from a window and one from a plugin, with the
 * Storybook each is drawn in.
 */
test('every story the tree declares is found where it stands', async () => {
  const declared = await stories()
  for (const [one, book] of [
    ['controls-menu--grouped', 'ui'],
    ['workspace--crowded', 'shell'],
    ['desktop-window--settings', 'editor'],
    ['flash-cards-window--cards-due', 'plugins'],
  ]) {
    assert.equal(declared.get(one), book, `${one} is declared in the ${book} Storybook`)
  }
})
