/**
 * What is rendered in place of a mark.
 *
 * A widget holds only what it renders. Its current position is read from the
 * editor when it acts, so a widget kept from an earlier render still writes at
 * the correct position.
 */
import { doc } from '@numen/ui'
import { EditorView, WidgetType } from '@codemirror/view'
import { addressResolver } from '../../lib/outside'

/** The disc a bullet list is marked by. */
export class Bullet extends WidgetType {
  override eq() {
    return true
  }

  toDOM() {
    const dot = doc.createElement('span')
    dot.className = 'cm-bullet'
    dot.textContent = '•'
    return dot
  }
}

/** The horizontal rule that a `---` renders. */
export class Rule extends WidgetType {
  override eq() {
    return true
  }

  toDOM() {
    const rule = doc.createElement('div')
    rule.className = 'cm-rule'
    rule.appendChild(doc.createElement('hr'))
    return rule
  }
}

/** A picture, drawn where its address was written. */
export class Picture extends WidgetType {
  constructor(readonly address: string) {
    super()
  }

  override eq(other: Picture) {
    return other.address === this.address
  }

  toDOM(view: EditorView) {
    const frame = doc.createElement('div')
    frame.className = 'cm-picture'
    const picture = doc.createElement('img')
    picture.src = view.state.facet(addressResolver)(this.address)
    picture.loading = 'lazy'
    picture.addEventListener('error', () => frame.classList.add('cm-picture-lost'))
    frame.appendChild(picture)
    return frame
  }
}
