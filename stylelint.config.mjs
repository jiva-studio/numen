/**
 * The styles of every stylesheet and every `<style>` block of a component. A colour is a design token:
 * a colour literal and a declaration of a `--numen-` token belong to the token files (a `tokens`
 * directory) and the theme files (a `theme` or `themes` directory) alone.
 */
export default {
  // The website is being rewritten by its owner and keeps the styles it had. Remove this entry when
  // that is done.
  ignoreFiles: ["apps/landing/**"],
  plugins: ["./tools/lint-layers/stylelint-plugin.mjs"],
  rules: {
    "color-no-hex": true,
    "color-named": "never",
    "function-disallowed-list": ["rgb", "rgba", "hsl", "hsla", "hwb", "lab", "lch", "oklab", "oklch", "color"],
    "numen/no-var-fallback": true,
    "numen/no-token-declaration": true,
    "numen/known-token": true,
    "numen/no-rounded-side-border": true,
  },
  overrides: [
    { files: ["**/*.vue"], customSyntax: "postcss-html" },
    {
      files: ["**/tokens/**/*.css", "**/theme/**/*.css", "**/themes/**/*.css"],
      rules: {
        "color-no-hex": null,
        "color-named": null,
        "function-disallowed-list": null,
        "numen/no-token-declaration": null,
      },
    },
  ],
};
