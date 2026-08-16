/**
 * A window-sized area for a panel to sit in, so a story shows it at the size
 * it is used at. Storybook's furniture; it ships to nobody.
 */
import type { Decorator } from '@storybook/vue3-vite'

export const framed: Decorator = (story) => ({
  components: { story },
  template: `
    <div class="numen h-screen bg-surface p-6 font-sans text-base text-ink">
      <div class="mx-auto flex h-full w-[420px] max-w-full flex-col">
        <story />
      </div>
    </div>
  `,
})
