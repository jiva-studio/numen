/**
 * What is drawn in place of a mark.
 *
 * A widget is told nothing but what it draws. at its current position is read back
 * from the editor when it acts, so a widget kept from an earlier draw still
 * writes in the right place.
 */
import { doc } from '../../../dom'
import { EditorView, WidgetType } from '@codemirror/view'
import { addressResolver } from '../outside'

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

/** A task's box, which is ticked by clicking it. */
export class Box extends WidgetType {
  constructor(
    readonly isTicked: boolean,
    readonly isWritable: boolean,
  ) {
    super()
  }

  override eq(other: Box) {
    return other.isTicked === this.isTicked && other.isWritable === this.isWritable
  }

  toDOM(view: EditorView) {
    const box = doc.createElement('input')
    box.type = 'checkbox'
    box.className = 'cm-box'
    box.checked = this.isTicked
    box.disabled = !this.isWritable
    box.addEventListener('mousedown', (event) => event.preventDefault())
    box.addEventListener('click', () => {
      if (!this.isWritable) return
      const pos = view.posAtDOM(box)
      const marker = view.state.doc.sliceString(pos, pos + 3)
      if (marker.length !== 3) return
      view.dispatch({
        changes: { from: pos, to: pos + 3, insert: this.isTicked ? '[ ]' : '[x]' },
      })
    })
    return box
  }
}

/** The line a `---` stands for. */
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
