<script setup lang="ts">
/**
 * A box whose name sits on its own outline.
 *
 * The outline is a `fieldset` and the name is its `legend`: the browser draws
 * the line through the legend's middle and leaves the gap the name stands in,
 * so the two are aligned by construction and there is nothing to fit by hand.
 *
 * The line runs half a name's height below the top of this, so the room the
 * name needs is kept here. A box put inside begins at the line and never under
 * it, whoever puts it there.
 */
const props = defineProps<{
  /** The name the box stands under. */
  label: string
  /**
   * The box it names. Where a box is handed in, the name is that box's own
   * label; a region carrying no box is named by whatever announces the region.
   */
  box?: string
}>()
</script>

<template>
  <div class="notched">
    <slot />

    <fieldset class="notched__outline">
      <legend class="notched__notch">
        <label v-if="props.box" :for="props.box">{{ label }}</label>
        <span v-else>{{ label }}</span>
      </legend>
    </fieldset>
  </div>
</template>

<style scoped>
.notched {
  --notch-line: 0.875rem;
  /* How far along the line the gap is cut, and the air either side of the name. */
  --notch-inset: 0.5rem;
  --notch-air: 0.25rem;
  /* The air a box inside keeps, which is what a box inherits rather than sets. */
  --box-air: 0.5rem;
  --box-pad-inline: 0.625rem;

  position: relative;
  display: grid;
  min-inline-size: 0;
  /* The room the name takes on the line above the box. */
  padding-block-start: calc(var(--notch-line) / 2);
}

.notched__outline {
  position: absolute;
  inset: 0;
  margin: 0;
  padding: 0 var(--notch-inset);
  border: var(--numen-stroke) solid var(--numen-field-border);
  border-radius: var(--numen-radius);
  pointer-events: none;
}

/* The legend is the name, and its own layout is what cuts the line. Nothing
   here may take that layout away from it. */
.notched__notch {
  max-inline-size: 100%;
  padding-inline: var(--notch-air);
  color: var(--numen-edge-label);
  font-size: var(--numen-edge-label-size);
  line-height: var(--notch-line);
}

.notched__notch > * {
  display: block;
  max-inline-size: 100%;
  overflow: hidden;
  white-space: nowrap;
  text-overflow: ellipsis;
  pointer-events: auto;
}

/* What stands in the outline is filled, and begins where the line is drawn. */
.notched > :deep(:not(fieldset)) {
  border-radius: var(--numen-radius);
  background: var(--numen-field-bg);
}

/* Where the keyboard is, the name says so. */
.notched:focus-within .notched__notch {
  color: var(--numen-node-fg);
}
</style>
