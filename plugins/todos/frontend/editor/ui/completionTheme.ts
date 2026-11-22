import type { Completion } from '@codemirror/autocomplete'
import { EditorView } from '@codemirror/view'
import { AtSign, CalendarDays, createElement, Flag, Folder, Hash } from 'lucide'

const ICONS = {
  project: Folder,
  context: AtSign,
  tag: Hash,
  date: CalendarDays,
  priority: Flag,
} as const

type IconType = keyof typeof ICONS

function isIconType(type: string | undefined): type is IconType {
  return type !== undefined && type in ICONS
}

export function renderCompletionIcon(completion: Completion): Node | null {
  if (!isIconType(completion.type)) return null
  const icon = createElement(ICONS[completion.type])
  icon.classList.add('cm-task-completion-icon')
  return icon
}

export const completionTheme = EditorView.baseTheme({
  '.cm-task-completion-icon': {
    display: 'inline-block',
    width: '1em',
    height: '1em',
    flex: 'none',
    marginRight: '0.4em',
    verticalAlign: '-0.125em',
    opacity: '0.75',
  },
  '.cm-completionIcon-project, .cm-completionIcon-context, .cm-completionIcon-tag, .cm-completionIcon-date, .cm-completionIcon-priority':
    { display: 'none' },
})
