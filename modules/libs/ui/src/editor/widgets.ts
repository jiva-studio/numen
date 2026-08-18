/**
 * What is drawn in place of a mark.
 *
 * A widget is told nothing but what it draws. Where it stands is read back
 * from the editor when it acts, so a widget kept from an earlier draw still
 * writes in the right place.
 */
import { EditorView, WidgetType } from '@codemirror/view'
import { resolving } from './outside'

/** The disc a bullet list is marked by. */
export class Bullet extends WidgetType {
  override eq() {
    return true
  }

  toDOM() {
    const dot = document.createElement('span')
    dot.className = 'cm-bullet'
    dot.textContent = '•'
    return dot
  }
}

/** A task's box, which is ticked by clicking it. */
export class Box extends WidgetType {
  constructor(
    readonly done: boolean,
    readonly writable: boolean,
  ) {
    super()
  }

  override eq(other: Box) {
    return other.done === this.done && other.writable === this.writable
  }

  toDOM(view: EditorView) {
    const box = document.createElement('input')
    box.type = 'checkbox'
    box.className = 'cm-box'
    box.checked = this.done
    box.disabled = !this.writable
    box.addEventListener('mousedown', (event) => event.preventDefault())
    box.addEventListener('click', () => {
      if (!this.writable) return
      const at = view.posAtDOM(box)
      const marker = view.state.doc.sliceString(at, at + 3)
      if (marker.length !== 3) return
      view.dispatch({
        changes: { from: at, to: at + 3, insert: this.done ? '[ ]' : '[x]' },
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
    const rule = document.createElement('div')
    rule.className = 'cm-rule'
    rule.appendChild(document.createElement('hr'))
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
    const frame = document.createElement('div')
    frame.className = 'cm-picture'
    const picture = document.createElement('img')
    picture.src = view.state.facet(resolving)(this.address)
    picture.loading = 'lazy'
    picture.addEventListener('error', () => frame.classList.add('cm-picture-lost'))
    frame.appendChild(picture)
    return frame
  }
}
