import { runTool } from "../../../../shared/tools.mjs";
import { REPO_ROOT, failOnFindings, requireMatches } from "../../../lib/graph.mjs";

/** The segments of a slice. */
const SEGMENTS = ["lib", "api", "model", "ui"];

/** The layers of a window, from the lowest. */
const WINDOW_LAYERS = ["shared", "entities", "features", "pages", "app"];

/** The layers of a shared UI library. */
const LIBRARY_LAYERS = ["shared", "features", "screens"];

/** The layers whose children are slices. */
const SLICED_LAYERS = ["entities", "features", "pages", "screens"];

/** Folders that hold test support, never shipped code. */
const SUPPORT = ["testing", "fixtures", "testdata"];

/** What a shared UI library keeps in `shared` besides segments: the design tokens and the fixtures its stories draw. */
const LIBRARY_SHARED = ["tokens", "fixtures"];

/** The roots whose top level is cut into layers, and the layers each has. */
export const LAYERED_ROOTS = [
  { root: "apps/desktop/editor/src", layers: [...WINDOW_LAYERS, "testing"], scope: "apps" },
  { root: "apps/desktop/flashcards/src", layers: WINDOW_LAYERS, scope: "apps" },
  { root: "apps/mobile/src", layers: WINDOW_LAYERS, scope: "apps" },
  { root: "libs/ui/components/src", layers: LIBRARY_LAYERS, shared: LIBRARY_SHARED, scope: "libs" },
  { root: "libs/ui/workbench/src", layers: LIBRARY_LAYERS, shared: LIBRARY_SHARED, scope: "libs" },
];

/**
 * The roots without layers, whose top-level folders are slices or segments: a plugin's `frontend` and the other
 * libraries. A folder of a library is an area when it has an `index.ts`, the entry the rest of the library imports it by.
 */
export const FLAT_ROOTS = [
  { root: "libs/core/client/src", scope: "libs" },
  { root: "libs/plugin/loader/src", scope: "libs" },
  { root: "libs/plugin/manager/src", scope: "libs" },
  { root: "libs/plugin/sdk-ts/src", scope: "libs" },
];

/** The folder and file names directly below a directory, from the tracked paths. */
function listChildren(files, dir) {
  const folders = new Map();
  const loose = [];
  for (const file of files) {
    if (!file.startsWith(`${dir}/`)) continue;
    const [first, ...rest] = file.slice(dir.length + 1).split("/");
    if (rest.length === 0) loose.push(first);
    else folders.set(first, [...(folders.get(first) ?? []), rest.join("/")]);
  }
  return { folders, loose };
}

/** A folder is a slice when it has a public `index.ts` or a segment folder of its own. */
function isSlice(inside) {
  return inside.some((file) => file === "index.ts" || SEGMENTS.includes(file.split("/")[0]));
}

const isSource = (file) => /\.(ts|vue|css)$/.test(file) && !/\.(d|test|spec|stories)\.ts$/.test(file);

/**
 * The folders of a root with layers that belong to no layer, and inside a layer the files and folders that are
 * neither a slice nor a segment: a sliced layer holds slices only, `shared` holds segments only.
 */
export function findUnplacedInLayers(files, { root, layers, shared = [] }) {
  const found = [];
  const top = listChildren(files, root);
  for (const name of top.folders.keys()) {
    if (!layers.includes(name)) found.push({ file: `${root}/${name}`, line: 1, symbol: `folder ${name}/ belongs to no layer (${layers.join(", ")})` });
  }
  for (const layer of layers.filter((one) => top.folders.has(one))) {
    const inside = listChildren(files, `${root}/${layer}`);
    if (layer === "shared") {
      for (const file of inside.loose.filter((name) => isSource(name))) {
        found.push({ file: `${root}/shared/${file}`, line: 1, symbol: "a file directly in shared/ belongs to no segment" });
      }
      for (const [name] of inside.folders) {
        if (!SEGMENTS.includes(name) && !SUPPORT.includes(name) && !shared.includes(name)) {
          found.push({ file: `${root}/shared/${name}`, line: 1, symbol: `folder shared/${name}/ is no segment (${SEGMENTS.join(", ")})` });
        }
      }
    }
    if (SLICED_LAYERS.includes(layer)) {
      for (const file of inside.loose.filter((name) => isSource(name))) {
        found.push({ file: `${root}/${layer}/${file}`, line: 1, symbol: `a file directly in ${layer}/ belongs to no slice` });
      }
      for (const [name, files2] of inside.folders) {
        if (!SUPPORT.includes(name) && !isSlice(files2)) {
          found.push({ file: `${root}/${layer}/${name}`, line: 1, symbol: `folder ${layer}/${name}/ is no slice: no index.ts and no segment` });
        }
      }
    }
  }
  return found;
}

/** The top-level folders of a root without layers that are neither a segment, test support nor a slice. */
export function findUnplacedFolders(files, { root }) {
  const found = [];
  for (const [name, inside] of listChildren(files, root).folders) {
    if (SEGMENTS.includes(name) || SUPPORT.includes(name) || isSlice(inside)) continue;
    found.push({ file: `${root}/${name}`, line: 1, symbol: `folder ${name}/ is no segment and no slice: no index.ts and no segment of its own` });
  }
  return found;
}

function listTrackedFiles(prefixes) {
  const listed = runTool("git", ["ls-files", "-z", "--", ...prefixes], { cwd: REPO_ROOT, encoding: "utf8", maxBuffer: 1 << 28 });
  return (listed.stdout || "").split("\0").filter(Boolean);
}

/** The folder of each plugin frontend, from the tracked paths. */
function listPluginRoots(files) {
  return [...new Set(files.map((file) => /^(plugins\/[^/.][^/]*)\/frontend\//.exec(file)?.[1]).filter(Boolean))].map((plugin) => ({ root: `${plugin}/frontend` }));
}

export function verifyLayeredFrontendRoots(scope) {
  const files = listTrackedFiles(["apps", "libs"]);
  const roots = LAYERED_ROOTS.filter((one) => one.scope === scope);
  requireMatches("Layered frontend roots", roots.length);
  failOnFindings("A frontend folder belongs to no layer, slice or segment", roots.flatMap((one) => findUnplacedInLayers(files, one)));
}

export function verifyFlatFrontendRoots() {
  const files = listTrackedFiles(["libs"]);
  failOnFindings("A library folder is neither a slice nor a segment", FLAT_ROOTS.flatMap((one) => findUnplacedFolders(files, one)));
}

export function verifyPluginFrontendRoots() {
  const files = listTrackedFiles(["plugins"]);
  const roots = listPluginRoots(files);
  requireMatches("Plugin frontends", roots.length);
  failOnFindings("A plugin frontend folder is neither a slice nor a segment", roots.flatMap((one) => findUnplacedFolders(files, one)));
}

export const FRONTEND_LAYOUT_VERIFIERS = [
  {
    id: "libs-layout-complete",
    scope: "libs",
    module: "boundary",
    name: "Every Top-Level Folder of a UI Library Is a Known Layer; Shared Holds Segments; the Other Libraries Hold Slices and Segments",
    run: () => {
      verifyLayeredFrontendRoots("libs");
      verifyFlatFrontendRoots();
    },
  },
  {
    id: "apps-layout-complete",
    scope: "apps",
    module: "boundary",
    name: "Every Top-Level Folder of a Window Is a Known Layer; Shared Holds Segments; a Sliced Layer Holds Slices",
    run: () => verifyLayeredFrontendRoots("apps"),
  },
  {
    id: "plugins-frontend-layout-complete",
    scope: "plugins",
    module: "boundary",
    name: "Every Folder of a Plugin Frontend Is a Slice or a Segment",
    run: verifyPluginFrontendRoots,
  },
];
