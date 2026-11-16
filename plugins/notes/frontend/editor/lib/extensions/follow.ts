import { EditorView } from '@codemirror/view'
import { getLinkAddressAt } from './address'
import { addressOpener } from '../outside'

/** Handles ctrl/cmd click on links to open their destination addresses. */
export const follow = EditorView.domEventHandlers({
  mousedown(event, view) {
    if (!event.metaKey && !event.ctrlKey) return false
    const pos = view.posAtCoords({ x: event.clientX, y: event.clientY })
    if (pos === null) return false

    const targetAddress = getLinkAddressAt(view.state, pos)
    if (targetAddress === null) return false
    view.state.facet(addressOpener)(targetAddress)
    return true
  },
})
