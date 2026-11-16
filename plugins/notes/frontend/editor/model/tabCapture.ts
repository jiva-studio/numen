import { indentLess, indentMore } from '@codemirror/commands'
import { StateEffect, StateField } from '@codemirror/state'
import { type KeyBinding, EditorView } from '@codemirror/view'

/**
 * Whether Tab belongs to the editor or to the page. It starts with the editor,
 * Escape releases it to the page, and keyboard input into the editor restores it, so
 * every visit starts the same way.
 */
const setTabCapture = StateEffect.define<boolean>()

export const tabCapture = StateField.define<boolean>({
  create: () => true,
  update: (target, change) =>
    change.effects.reduce((now, effect) => (effect.is(setTabCapture) ? effect.value : now), target),
})

export const restoreTabCapture = EditorView.domEventHandlers({
  focus: (_press, view) => {
    if (!view.state.field(tabCapture)) view.dispatch({ effects: setTabCapture.of(true) })
    return false
  },
})

/** Tab indents while the editor holds it, and walks on once it does not. */
export const tabKeyBinding: KeyBinding = {
  key: 'Tab',
  run: (view) => view.state.field(tabCapture) && indentMore(view),
  shift: (view) => view.state.field(tabCapture) && indentLess(view),
}

/**
 * Escape hands Tab to the page and is passed on, so what else answers Escape
 * around the editor still answers it.
 */
export const escapeKeyBinding: KeyBinding = {
  key: 'Escape',
  run: (view) => {
    if (view.state.field(tabCapture)) view.dispatch({ effects: setTabCapture.of(false) })
    return false
  },
}
