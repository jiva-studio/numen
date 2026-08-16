import type { VariantProps } from 'class-variance-authority'
import { cva } from 'class-variance-authority'

export { default as Button } from './Button.vue'

/**
 * Hover is a brightness filter, so a variant sets its colours once and they
 * hold in both themes. Focus wears the ring at the width the tokens name,
 * which is the treatment every focusable thing in this library uses.
 */
export const buttonVariants = cva(
  [
    'inline-flex shrink-0 items-center justify-center gap-2 whitespace-nowrap',
    'font-sans text-base font-medium',
    'cursor-pointer transition-[filter,background-color,color] duration-100 ease-numen',
    'outline-none focus-visible:ring-(length:--numen-ring-width) focus-visible:ring-ring',
    'hover:brightness-[var(--numen-hover-brightness)]',
    'disabled:pointer-events-none disabled:opacity-50',
    '[&_svg]:pointer-events-none [&_svg]:shrink-0',
  ],
  {
    variants: {
      variant: {
        solid: 'bg-accent text-accent-ink',
        outline: 'border border-rule bg-raised text-ink',
        ghost: 'text-ink hover:bg-raised',
      },
      size: {
        default: 'h-8 rounded-node px-3',
        small: 'h-7 rounded-node px-2 text-small',
        icon: 'size-action rounded-pill',
        'icon-small': 'size-7 rounded-pill',
      },
    },
    defaultVariants: {
      variant: 'solid',
      size: 'default',
    },
  },
)

export type ButtonVariants = VariantProps<typeof buttonVariants>
