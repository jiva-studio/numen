import { EditorView } from '@codemirror/view'

const ICONS = {
  project:
    '<path d="M20 20a2 2 0 0 0 2-2V8a2 2 0 0 0-2-2h-7.9a2 2 0 0 1-1.69-.9L9.6 3.9A2 2 0 0 0 7.93 3H4a2 2 0 0 0-2 2v13a2 2 0 0 0 2 2Z"/>',
  context: '<circle cx="12" cy="12" r="4"/><path d="M16 8v5a3 3 0 0 0 6 0v-1a10 10 0 1 0-4 8"/>',
  tag: '<path d="M4 9h16"/><path d="M4 15h16"/><path d="M10 3 8 21"/><path d="m16 3-2 18"/>',
  date: '<path d="M8 2v4"/><path d="M16 2v4"/><rect width="18" height="18" x="3" y="4" rx="2"/><path d="M3 10h18"/>',
  priority:
    '<path d="M4 22V4"/><path d="M4 15s1-1 4-1 5 2 8 2 4-1 4-1V4s-1 1-4 1-5-2-8-2-4 1-4 1"/>',
} as const

function toMask(paths: string): string {
  const svg = `<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="none" stroke="black" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">${paths}</svg>`
  return `url("data:image/svg+xml,${encodeURIComponent(svg)}")`
}

function toIconRule(type: keyof typeof ICONS): Record<string, Record<string, string>> {
  const mask = toMask(ICONS[type])
  return {
    [`.cm-completionIcon-${type}`]: {
      width: '1em',
      opacity: '1',
    },
    [`.cm-completionIcon-${type}::after`]: {
      content: '""',
      display: 'block',
      width: '1em',
      height: '1em',
      backgroundColor: 'currentColor',
      maskImage: mask,
      webkitMaskImage: mask,
      maskSize: 'contain',
      webkitMaskSize: 'contain',
      maskRepeat: 'no-repeat',
      webkitMaskRepeat: 'no-repeat',
      maskPosition: 'center',
      webkitMaskPosition: 'center',
    },
  }
}

export const completionTheme = EditorView.baseTheme(
  Object.assign({}, ...(Object.keys(ICONS) as (keyof typeof ICONS)[]).map(toIconRule)),
)
