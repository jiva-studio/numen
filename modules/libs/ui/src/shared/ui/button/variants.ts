import type { VariantProps } from 'class-variance-authority'
import { cva } from 'class-variance-authority'

/**
 * Hover mixes a little of a variant's own text into its ground, so one
 * expression darkens a light button and lightens a dark one. Focus wears the
 * ring at the width the tokens name, which is the treatment every focusable
 * thing in this library uses.
 */
export const buttonVariants = cva(
  [
    'inline-flex shrink-0 items-center justify-center gap-2 whitespace-nowrap',
    'font-sans text-base font-medium',
    'cursor-pointer transition-[background-color,color] duration-hover ease-numen',
    'outline-none ring-numen',
    'disabled:pointer-events-none disabled:opacity-50',
    '[&_svg]:pointer-events-none [&_svg]:shrink-0',
  ],
  {
    variants: {
      variant: {
        solid: [
          'bg-accent text-accent-ink',
          'hover:bg-[color-mix(in_oklab,var(--numen-accent),var(--numen-accent-ink)_8%)]',
        ],
        outline: [
          'border border-rule bg-raised text-ink',
          'hover:bg-[color-mix(in_oklab,var(--numen-raised),var(--numen-ink)_8%)]',
        ],
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
