/**
 * The styles of every stylesheet and every `<style>` block of a component. A colour is a design token:
 * a colour literal and a declaration of a `--numen-` token belong to the token files (a `tokens`
 * directory) and the theme files (a `theme` or `themes` directory) alone.
 *
 * The window chrome keeps one size at every interface scale: the title bar, the status bar and the
 * frontend of every plugin that draws a view in one of their slots.
 */
import fs from "node:fs";

/** The slots of the window chrome. */
const CHROME_SLOT = /^(titleBar|statusBar)(\.|$)/;

/** The frontends of the bars and of every plugin whose manifest draws a view in a slot of a bar. */
function listChromeFiles() {
  const bars = ["views-title-bar", "views-status-bar"];
  const widgets = fs.readdirSync("plugins").filter((name) => {
    const manifest = `plugins/${name}/manifest.json`;
    if (!fs.existsSync(manifest)) return false;
    const slots = JSON.parse(fs.readFileSync(manifest, "utf8")).contributes?.slots ?? [];
    return slots.some((one) => one.view && CHROME_SLOT.test(one.slot ?? ""));
  });
  return [...new Set([...bars, ...widgets])].sort().map((name) => `plugins/${name}/frontend/**/*.{css,vue}`);
}

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
    { files: listChromeFiles(), rules: { "numen/chrome-fixed-size": true } },
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
