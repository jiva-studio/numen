export function parseCodeowners(text) {
  const rules = [];
  for (const raw of text.split("\n")) {
    const line = raw.replace(/\s+#.*$/, "").trim();
    if (line === "" || line.startsWith("#")) continue;
    const [pattern, ...owners] = line.split(/\s+/);
    rules.push({ pattern, owners: owners.map((owner) => owner.replace(/^@/, "").toLowerCase()) });
  }
  return rules;
}

function toRegExp(pattern) {
  const anchored = pattern.startsWith("/") || pattern.slice(0, -1).includes("/");
  const directory = pattern.endsWith("/");
  const body = pattern.replace(/^\//, "").replace(/\/$/, "");
  let source = "";
  for (let i = 0; i < body.length; i++) {
    const char = body[i];
    if (char === "*" && body[i + 1] === "*") {
      const slash = body[i + 2] === "/";
      source += slash ? "(?:.*/)?" : ".*";
      i += slash ? 2 : 1;
    } else if (char === "*") {
      source += "[^/]*";
    } else if (char === "?") {
      source += "[^/]";
    } else {
      source += char.replace(/[.+^${}()|[\]\\]/g, "\\$&");
    }
  }
  const prefix = anchored ? "^" : "^(?:.*/)?";
  const suffix = directory ? "/.*$" : "(?:/.*)?$";
  return new RegExp(prefix + source + suffix);
}

export function getOwners(rules, path) {
  let owners = [];
  for (const rule of rules) {
    if (toRegExp(rule.pattern).test(path)) owners = rule.owners;
  }
  return owners;
}
