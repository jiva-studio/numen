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
import MarkdownNoteEditor from '../MarkdownNoteEditor.vue'
import { useNoteFile } from './useNoteFile'

export function createNoteEditorView(notes: Pick<VaultNotes, 'read' | 'write'>): ViewRegistration {
  return {
    capability: NOTE_EDITOR,
    component: defineComponent({
      name: 'NoteEditorView',
      props: { context: { type: Object as PropType<NoteEditorContext>, required: true } },
      setup(props) {
        const state = useNoteFile(props.context.path, notes)
        return () => h(MarkdownNoteEditor, { state })
      },
    }),
  }
}
