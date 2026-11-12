import type { VariantProps } from 'class-variance-authority'
import { cva } from 'class-variance-authority'

/**
 * Hover moves a variant's background without lowering the text contrast: a ground apart from its text
 * (the accent, the alarm tint) moves further from it; the outline variant moves a little toward its text.
 * Focus draws the ring at the token width, the treatment of every focusable element in this library.
 */
export const buttonVariants = cva(
  [
    'inline-flex shrink-0 items-center justify-center gap-2 whitespace-nowrap',
    'font-sans text-base font-medium',
    'duration-hover ease-numen cursor-pointer transition-[background-color,color]',
    'ring-numen outline-none',
    'disabled:pointer-events-none disabled:opacity-50',
    '[&_svg]:pointer-events-none [&_svg]:shrink-0',
  ],
  {
    variants: {
      variant: {
        solid: [
          'bg-accent text-accent-ink',
          'hover:bg-[color-mix(in_oklab,var(--numen-accent),var(--numen-ink)_8%)]',
        ],
        outline: [
          'border-field-rule bg-field text-ink border',
          'hover:bg-[color-mix(in_oklab,var(--numen-field-bg),var(--numen-ink)_8%)]',
        ],
        danger: [
          'border-alarm text-alarm border bg-[var(--numen-alarm-bg)]',
          'hover:bg-[color-mix(in_oklab,var(--numen-alarm-bg),var(--numen-surface)_40%)]',
        ],
        ghost: 'text-ink hover:bg-raised',
      },
      size: {
        default: 'rounded-node h-8 px-3',
        small: 'rounded-node text-small h-7 px-2',
        icon: 'size-action rounded-pill',
        'icon-small': 'rounded-pill size-7',
      },
    },
    defaultVariants: {
      variant: 'solid',
      size: 'default',
    },
  },
)

export type ButtonVariants = VariantProps<typeof buttonVariants>
