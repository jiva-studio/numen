/**
 * A window-sized area that contains a panel, so a story shows it at its real
 * size. Storybook test fixture only; it is not shipped.
 *
 * The area is fixed in pixels and does not change with the interface
 * multiplier. A window has a fixed pixel size and a pane is a share of it, so a
 * larger interface is drawn in the same area, and the crowding a story shows at
 * 2× matches the crowding in the window.
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
