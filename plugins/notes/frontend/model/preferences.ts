/**
 * The editor settings of the application as the notes plugin follows them, undefined until the window has loaded them.
 */
import { shallowRef } from 'vue'
import type { EditorSettings } from '@numen/plugin-sdk'

export const editorPreferences = shallowRef<EditorSettings | undefined>(undefined)
