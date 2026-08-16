# ADR-0028: The component library is shadcn-vue on Tailwind

- **Status:** Accepted
- **Date:** 2026-08-16
- **Applies to:** `modules/libs/ui`
- **Related:** ADR-0020

## Context

ADR-0020 settled how an interface component is built and left one question
open: which general component library to use. It said the question stays open
until a screen exists that needs a dialog, a menu or a table, and that the rule
that every value is a design token is what makes deferring free.

That screen exists. The agent panel needs a field, a button, and shortly a
dropdown, a multi-select and a dialog.

Two facts shape the answer.

The first is that **the product is not mostly made of library components**. It
is a plex, an editor, a review card and a conversation. A library is chosen
here for the chrome around those, and the chrome is the smaller half.

The second is that **the module already has a design language**. Every value a
component paints with is a custom property in one file, themed by
`color-scheme` and `light-dark()`. A library arriving with its own palette,
its own theming mechanism and its own vocabulary would be a second language in
the same module.

## Decision

### shadcn-vue, on Tailwind, with the components copied in

shadcn-vue is not a dependency. Its CLI copies a component's source into this
module, and from that point the component is this module's own code, edited
like any other file. Underneath it are Reka UI for behaviour and Tailwind for
styling.

This is chosen over a library that ships components as a package because **the
components are edited on the way in**, and the edits are not cosmetic: a
generated component arrives with a palette this module does not use, with
`dark:` variants that do not work here, and with types that do not satisfy this
module's compiler settings. Owning the source is what makes those edits
possible rather than a fork.

It is chosen over building on Reka UI alone because a dialog, a combobox and a
menu are a great deal of interaction to get right, and shadcn-vue is that work
already done, in a form that can be corrected.

### Tailwind's theme is defined from the tokens, never the reverse

`tokens.css` remains the whole styling contract. `theme.css` defines every name
Tailwind paints with in terms of a token from it, so `bg-surface` and
`var(--numen-surface)` resolve to one value.

**This is the condition the choice was made under.** ADR-0020's rule is that a
component never reaches past a token for a value, and a library with a palette
of its own would have broken it on the first component. Defining Tailwind's
names from the tokens keeps one source of truth and one vocabulary.

### A theme is chosen by `color-scheme`, and no component writes `dark:`

The tokens are `light-dark()` pairs, so setting `color-scheme` changes every
colour in the module at once. Tailwind's `dark:` variant reads a class or the
operating system's setting instead, and follows neither. A component carrying
one is fixed in whichever theme it was written in while everything around it
changes.

So no component in this module writes a `dark:` class, and generated components
have theirs removed as they are copied in.

### Where each way of styling applies

A component made of DOM is styled with utilities. The plex is styled with
scoped CSS, because SVG attributes are what it paints with and no utility
expresses them.

The line matters in one direction that is easy to get wrong: component styles
are unlayered and Tailwind's are in `@layer`, so **a scoped block silently
overrides every utility on the same element**. A scoped block on a DOM
component is for what a utility cannot say, and never for what one can.

### What this does not decide

**Which components get copied in.** One is copied when a screen needs it, and
not before. The registry is not an inventory to be stocked.

## Consequences

**Positive**

- A dialog, a menu, a combobox and a table are available at the cost of running
  a command, with their interaction already built.
- Nothing is versioned against a library. There is no upgrade that changes how
  a button looks, because the button is this module's file.
- The plex was not touched. It was already drawn from tokens alone, which is
  what ADR-0020 predicted would make this cheap.
- The design language did not change. The tokens are what they were.

**Negative**

- Every component copied in has to be read and edited before it is kept, and a
  reviewer has to know that. A generated file merged unread brings a palette
  and a theming mechanism this module rejected.
- Two ways of writing a style now live in one module, and the rule for which
  applies where has to be known rather than inferred.
- Tailwind's own reset is in the module's stylesheet. It stays out of the
  desktop window only because that window imports the tokens and not the theme,
  which is a distinction a future import could quietly undo.
- Upstream fixes do not arrive. A defect corrected in shadcn-vue is corrected
  here by hand, or not at all.

## Alternatives considered

**PrimeVue.** Rejected. It is the stronger library where an application is
mostly tables and forms, and this one is not. Its theming is a system of its
own, so the module would either adopt a second set of design tokens or spend
its life translating between two — and the translation would be the thing that
broke every time either side moved.

**Reka UI alone, with the components written here.** Rejected as the default,
though it is what shadcn-vue is underneath. The behaviour of a combobox is not
where this product's difficulty lies, and writing one from primitives spends
the effort on a solved problem. Keeping Reka directly beneath means the option
remains where a generated component turns out to be the wrong shape.

**No library — carry on as the plex does.** Rejected. It is what happened until
now and it was right until now. A dialog and a combobox carry a long tail of
interaction, and building that from nothing for every one of them is the cost
this defers.
