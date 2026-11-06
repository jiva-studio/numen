/**
 * The copy of the spread in front that a turn into another document carries out.
 */
import { describe, expect, it } from 'vitest'

import { copyPaper } from './leaving'

const createPaper = () => {
  const paper = document.createElement('div')
  paper.className = 'book__paper'
  paper.style.translate = '-600px 0'
  paper.innerHTML = '<h2 id="title" data-offset="10">Title</h2><p data-offset="16">Text</p>'
  return paper
}

describe('copying the paper', () => {
  it('puts a copy of the text and its place into the holder', () => {
    const holder = document.createElement('div')

    const copy = copyPaper(createPaper(), holder)

    expect(holder.children).toHaveLength(1)
    expect(holder.firstElementChild).toBe(copy)
    expect(copy.className).toBe('book__paper')
    expect(copy.style.translate).toBe('-600px 0')
    expect(copy.textContent).toBe('TitleText')
  })

  it('leaves no ids and no run offsets in the copy', () => {
    const copy = copyPaper(createPaper(), document.createElement('div'))

    expect(copy.querySelectorAll('[id], [data-offset]')).toHaveLength(0)
  })

  it('replaces the copy the holder already holds', () => {
    const holder = document.createElement('div')
    const first = copyPaper(createPaper(), holder)

    copyPaper(createPaper(), holder)

    expect(holder.children).toHaveLength(1)
    expect(holder.firstElementChild).not.toBe(first)
  })
})
