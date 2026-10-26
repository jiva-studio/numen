/**
 * A window-sized area for a panel to sit in, so a story shows it at the size
 * it is used at. Storybook's furniture; it ships to nobody.
 *
 * The room is in pixels and stays there while the interface multiplier moves.
 * A window is a fixed number of pixels and a pane is a share of one, so a
 * larger interface is drawn in the same room and the crowding a story shows at
 * 2× is the crowding the window has.
 */
import type { Decorator } from '@storybook/vue3-vite'

export const frameStory: Decorator = (story) => ({
  components: { story },
  template: `
    <div class="numen h-screen bg-surface p-6 font-sans text-base text-ink">
      <div class="mx-auto flex h-full w-[420px] max-w-full flex-col">
        <story />
      </div>
    </div>
  `,
})

/**
 * The classes of the boxes the stories set a component in. They stand here, outside the stories,
 * because the utilities the stories are drawn with are built from the plugin's frontend files and
 * not from its stories.
 */
export const STORY_BOXES = {
  thread:
    'numen h-[460px] w-[420px] max-w-[calc(100vw-2rem)] rounded-panel border border-rule bg-surface px-4 py-3',
  shortThread: 'numen h-[300px] w-[420px] rounded-panel border border-rule bg-surface px-4 py-3',
  code: 'rounded-node bg-raised px-1.5 py-0.5',
  composers: 'numen flex w-[420px] max-w-[calc(100vw-2rem)] flex-col gap-4',
  narrowComposers: 'numen flex w-[180px] flex-col gap-4',
  toolCalls: 'numen flex w-[320px] flex-col gap-1 text-ink',
  indicators: 'numen flex items-center gap-4 text-ink',
  bubble: 'flex size-action items-center justify-center rounded-pill bg-bubble text-bubble-ink',
  accent: 'flex size-action items-center justify-center rounded-pill bg-accent text-accent-ink',
} as const
