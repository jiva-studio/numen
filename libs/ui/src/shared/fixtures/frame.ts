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
