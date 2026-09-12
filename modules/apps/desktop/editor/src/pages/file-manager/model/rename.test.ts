/**
 * The name a row is given, read without a screen.
 *
 * A name is punctuated the way a person writes one, and the rule has to tell a
 * sentence's dot from a file's ending. The negatives are the ones worth having:
 * a name that moves nothing asks the vault for nothing.
 */
import { describe, expect, it } from 'vitest'
import { resolveRenamePath } from './rename'

describe('a name typed over a row', () => {
  it('files it under that name, in the folder it is already in', () => {
    expect(resolveRenamePath('physics/Kelvin.md', 'Celsius.md')).toBe('physics/Celsius.md')
  })

  it('files it at the top of the vault where that is where it is', () => {
    expect(resolveRenamePath('Entropy.md', 'Order.md')).toBe('Order.md')
  })

  it('moves nothing where the name is the one it carries', () => {
    expect(resolveRenamePath('physics/Kelvin.md', 'Kelvin.md')).toBe('')
  })

  it('moves nothing where nothing was typed', () => {
    expect(resolveRenamePath('physics/Kelvin.md', '   ')).toBe('')
  })

  /** A name is a name and not a path: the field renames, and dragging moves. */
  it('moves nothing where the name names a folder of its own', () => {
    expect(resolveRenamePath('physics/Kelvin.md', 'heat/Kelvin.md')).toBe('')
  })

  it('keeps the ending the file carries, where the name carries none', () => {
    expect(resolveRenamePath('physics/Kelvin.md', 'Celsius')).toBe('physics/Celsius.md')
    expect(resolveRenamePath('Cover.png', 'Jacket')).toBe('Jacket.png')
  })

  it('takes the ending the name carries, where it carries one', () => {
    expect(resolveRenamePath('Cover.png', 'Jacket.jpg')).toBe('Jacket.jpg')
  })

  it('moves nothing where the name is the one it carries, ending and all', () => {
    expect(resolveRenamePath('physics/Kelvin.md', 'Kelvin')).toBe('')
  })

  /** A name beginning with a dot is a name, and the whole of it. */
  it('keeps nothing for a file whose name is an ending', () => {
    expect(resolveRenamePath('.keep', 'ignored')).toBe('ignored')
  })

  /**
   * A dot is punctuation more often than it is an ending. A note called after a
   * chapter, a figure or a person keeps the ending its file carries, or it
   * stops being a note the moment it is named.
   */
  it('reads a dot inside a sentence as punctuation, not as an ending', () => {
    expect(resolveRenamePath('physics/Kelvin.md', 'Ch. 2 heat')).toBe('physics/Ch. 2 heat.md')
    expect(resolveRenamePath('Cover.png', 'Fig. 3')).toBe('Fig. 3.png')
    expect(resolveRenamePath('Kelvin.md', 'Mr. Smith')).toBe('Mr. Smith.md')
  })

  it('reads a dot with no space after it as an ending, in any script', () => {
    expect(resolveRenamePath('Kelvin.md', 'Notes.txt')).toBe('Notes.txt')
    expect(resolveRenamePath('Cover.png', 'Jacket.jpeg')).toBe('Jacket.jpeg')
    expect(resolveRenamePath('Kelvin.md', 'Записка.текст')).toBe('Записка.текст')
    expect(resolveRenamePath('Kelvin.md', 'Заметка')).toBe('Заметка.md')
  })

  /**
   * What the rule gives up: a dot between digits reads as an ending, so a note
   * called after a version keeps the name it was given and takes no other.
   */
  it('reads a version as an ending, and leaves it alone', () => {
    expect(resolveRenamePath('Kelvin.md', 'v1.2')).toBe('v1.2')
  })

  /** A name that is a dot and an ending is the whole name, and takes no other. */
  it('adds nothing to a name that is an ending', () => {
    expect(resolveRenamePath('physics/Kelvin.md', '.gitignore')).toBe('physics/.gitignore')
  })

  it('takes what was typed for a folder, which carries no ending at all', () => {
    expect(resolveRenamePath('physics/v1.0', 'v2', true)).toBe('physics/v2')
  })
})
