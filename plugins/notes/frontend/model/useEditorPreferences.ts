/**
 * The preferences an editor runs with: what its props set, else what the settings say.
 */
import { computed } from 'vue'
import { getFontFamilyValue } from './fontFamily'
import { editorPreferences, type MarkdownEditorPreferences } from './preferences'

export function useEditorPreferences(props: MarkdownEditorPreferences) {
  return computed(() => ({
    softWrap: props.softWrap ?? editorPreferences.softWrap,
    lineNumbers: props.lineNumbers ?? editorPreferences.lineNumbers,
    tabSize: props.tabSize ?? editorPreferences.tabSize,
    lineHeight: props.lineHeight ?? editorPreferences.lineHeight,
    highlightActiveLine: props.highlightActiveLine ?? editorPreferences.highlightActiveLine,
    fontFamily: getFontFamilyValue(props.fontFamily ?? editorPreferences.fontFamily) ?? '',
  }))
}
