import fs from "node:fs";
import path from "node:path";
import ts from "typescript";
import { REPO_ROOT, failOnFindings, requireMatches } from "./helpers.mjs";

/** The file that hands the registrations of a plugin to the host. */
const REGISTRATIONS_FILE = "libs/plugin/loader/src/hosting/guards/registrations.ts";

const isFunctionLike = (node) => ts.isArrowFunction(node) || ts.isFunctionExpression(node) || ts.isFunctionDeclaration(node) || ts.isMethodDeclaration(node);

/** The parameters of a function that are typed as a registration: `ReaderRegistration`, `TabKindRegistration<State>`. */
function listRegistrationParams(fn) {
  return fn.parameters.filter(
    (param) =>
      ts.isIdentifier(param.name) &&
      param.type !== undefined &&
      ts.isTypeReferenceNode(param.type) &&
      ts.isIdentifier(param.type.typeName) &&
      param.type.typeName.text.endsWith("Registration"),
  );
}

/** What a use of the parameter `name` is: a read of one property, a copy, a destructuring, or the whole object handed on. */
function classifyUse(node) {
  const parent = node.parent;
  if (ts.isPropertyAccessExpression(parent) && parent.expression === node) return { kind: "read", props: [parent.name.text] };
  if (ts.isSpreadAssignment(parent)) return { kind: "copy" };
  if (ts.isCallExpression(parent) && ts.isIdentifier(parent.expression) && parent.expression.text === "structuredClone") return { kind: "copy" };
  if (ts.isVariableDeclaration(parent) && parent.initializer === node && ts.isObjectBindingPattern(parent.name)) {
    return { kind: "read", props: parent.name.elements.map((one) => (one.propertyName ?? one.name).getText()) };
  }
  const callee = ts.isCallExpression(parent) && parent.arguments.includes(node) ? parent.expression.getText() : undefined;
  return { kind: "whole", callee };
}

/**
 * The functions of a source that read a property of a registration more than once, or read a property and also hand
 * the registration whole to another function: the check and the host then read it separately, and a getter answers
 * each read as it likes. A function whose first use of the registration is a copy (a spread or `structuredClone`) reads
 * the plugin's object once and is not reported.
 *
 * @param {string} text The text of the source
 * @param {string} file The path reported with each finding
 * @returns {{ findings: Array<{ file: string, line: number, symbol: string }>, read: number }}
 */
export function findRegistrationsReadTwice(text, file = REGISTRATIONS_FILE) {
  const source = ts.createSourceFile(file, text, ts.ScriptTarget.Latest, true, ts.ScriptKind.TS);
  const lineOf = (node) => source.getLineAndCharacterOfPosition(node.getStart(source)).line + 1;
  const findings = [];
  let read = 0;
  const visit = (fn) => {
    for (const param of listRegistrationParams(fn)) {
      read += 1;
      const name = param.name.text;
      const uses = [];
      const collect = (node) => {
        const isMemberName = node.parent && ts.isPropertyAccessExpression(node.parent) && node.parent.name === node;
        const isKey = node.parent && ts.isPropertyAssignment(node.parent) && node.parent.name === node;
        if (ts.isIdentifier(node) && node.text === name && !isMemberName && !isKey) uses.push({ node, ...classifyUse(node) });
        ts.forEachChild(node, collect);
      };
      if (fn.body) collect(fn.body);
      uses.sort((a, b) => a.node.getStart(source) - b.node.getStart(source));
      if (uses.length === 0 || uses[0].kind === "copy") continue;
      const seen = new Set();
      const reported = new Set();
      for (const use of uses.filter((one) => one.kind === "read")) {
        for (const prop of use.props) {
          if (seen.has(prop) && !reported.has(`twice ${prop}`)) {
            reported.add(`twice ${prop}`);
            findings.push({ file, line: lineOf(use.node), symbol: `${name}.${prop} is read more than once` });
          }
          seen.add(prop);
        }
      }
      const whole = uses.find((one) => one.kind === "whole");
      if (whole) {
        for (const use of uses.filter((one) => one.kind === "read")) {
          for (const prop of use.props) {
            if (reported.has(`whole ${prop}`)) continue;
            reported.add(`whole ${prop}`);
            findings.push({ file, line: lineOf(use.node), symbol: `${name}.${prop} is read and ${name} is also handed whole${whole.callee ? ` to ${whole.callee}` : ""} (line ${lineOf(whole.node)})` });
          }
        }
      }
    }
  };
  const walk = (node) => {
    if (isFunctionLike(node)) visit(node);
    ts.forEachChild(node, walk);
  };
  walk(source);
  return { findings, read };
}

export function verifyRegistrationsReadOnce() {
  const text = fs.readFileSync(path.join(REPO_ROOT, REGISTRATIONS_FILE), "utf8");
  const { findings, read } = findRegistrationsReadTwice(text);
  requireMatches("Registration parameters", read);
  failOnFindings("A plugin registration is read more than once, or read and handed on whole", findings);
}

export const REGISTRATION_VERIFIERS = [
  {
    id: "registrations-read-once",
    scope: "libs",
    module: "boundary",
    name: "The Plugin Host Reads Each Registration Once (a copy first, or one read of each property)",
    run: verifyRegistrationsReadOnce,
  },
];
