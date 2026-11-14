/**
 * The view of the `note-editor` capability: one note of the vault, edited in the live editor.
 */
import { defineComponent, h, type PropType } from 'vue'
import {
  NOTE_EDITOR,
  type NoteEditorContext,
  type VaultNotes,
  type ViewRegistration,
} from '@numen/plugin-sdk'
import MarkdownNoteEditor from './MarkdownNoteEditor.vue'
import { editorPreferences, type MarkdownEditorPreferences } from '../model/preferences'
import { useNoteFile } from '../model/useNoteFile'

export function createNoteEditorView(
  notes: Pick<VaultNotes, 'read' | 'write'>,
  preferences: MarkdownEditorPreferences = editorPreferences,
): ViewRegistration {
  return {
    capability: NOTE_EDITOR,
    component: defineComponent({
      name: 'NoteEditorView',
      props: { context: { type: Object as PropType<NoteEditorContext>, required: true } },
      setup(props) {
        const state = useNoteFile(props.context.path, notes)
        return () =>
          h(MarkdownNoteEditor, {
            state,
            softWrap: preferences.softWrap,
            lineNumbers: preferences.lineNumbers,
            tabSize: preferences.tabSize,
            fontFamily: preferences.fontFamily,
            lineHeight: preferences.lineHeight,
            highlightActiveLine: preferences.highlightActiveLine,
          })
      },
    }),
  }
}
