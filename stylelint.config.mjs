/**
 * The styles of every stylesheet and every `<style>` block of a component. A colour is a design token:
 * a colour literal and a declaration of a `--numen-` token belong to the token files (a `tokens`
 * directory) and the theme files (a `theme` or `themes` directory) alone.
 */
export default {
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
      // Lengths: the shared libraries, the plugin manager, the desktop windows and the plugin frontends.
      files: ["libs/ui/**/*.{css,vue}", "libs/plugin/manager/**/*.{css,vue}", "apps/desktop/**/*.{css,vue}", "plugins/*/frontend/**/*.{css,vue}"],
      rules: { "numen/no-bare-length": true },
    },
    {
      files: ["**/tokens/**/*.css", "**/theme/**/*.css", "**/themes/**/*.css"],
      rules: {
        "color-no-hex": null,
        "color-named": null,
        "function-disallowed-list": null,
        "numen/no-token-declaration": null,
        "numen/no-bare-length": null,
      },
    },
  ],
};
