import assert from "node:assert/strict";
import fs from "node:fs";
import path from "node:path";
import { describe, it } from "node:test";
import { REPO_ROOT } from "../../../lib/graph.mjs";
import { findRegistrationsReadTwice } from "./check.mjs";

const symbols = (text) => findRegistrationsReadTwice(text, "r.ts").findings.map((one) => `${one.line} ${one.symbol}`);

const wrap = (body, type = "CommandRegistration") => `const make = {\n  command: (reg: ${type}) => {\n${body}\n  },\n}\n`;

describe("a registration is read once", () => {
  it("fails a property read twice", () => {
    assert.deepEqual(symbols(wrap("    check(reg.id, isShared(reg.id))")), ["3 reg.id is read more than once"]);
  });

  it("fails a property read and the registration handed on whole, in either order", () => {
    assert.deepEqual(symbols(wrap("    check(reg.id)\n    return add(register(reg))")), ["3 reg.id is read and reg is also handed whole to register (line 4)"]);
    assert.deepEqual(symbols(wrap("    register(reg)\n    check(reg.id)")), ["4 reg.id is read and reg is also handed whole to register (line 3)"]);
  });

  it("fails a destructured property that is also handed on whole", () => {
    assert.deepEqual(symbols(wrap("    const { id } = reg\n    return register(reg)")), ["3 reg.id is read and reg is also handed whole to register (line 4)"]);
  });

  it("passes a registration copied first, by a spread or by structuredClone", () => {
    assert.deepEqual(symbols(wrap("    const snapshot = { ...reg }\n    check(snapshot.id)\n    return register(snapshot)")), []);
    assert.deepEqual(symbols(wrap("    const copy = structuredClone(reg)\n    check(copy.id)\n    return register(copy)")), []);
  });

  it("passes one destructuring and a registration read only through distinct properties", () => {
    assert.deepEqual(symbols(wrap("    const { id, handler } = reg\n    check(id)\n    return register({ id, handler })")), []);
    assert.deepEqual(symbols(wrap("    check(reg.id)\n    return register(reg.handler)")), []);
  });

  it("reads only parameters typed as a registration and counts them", () => {
    const plain = "const f = (value: Options) => { check(value.id, value.id); return register(value) }";
    const found = findRegistrationsReadTwice(plain, "r.ts");
    assert.deepEqual(found.findings, []);
    assert.equal(found.read, 0);
    assert.equal(findRegistrationsReadTwice(wrap("    return 1"), "r.ts").read, 1);
  });

  it("reads the registration functions of the plugin host", () => {
    const file = "libs/plugin/loader/src/hosting/guards/registrations.ts";
    const found = findRegistrationsReadTwice(fs.readFileSync(path.join(REPO_ROOT, file), "utf8"), file);
    assert.ok(found.read >= 1);
  });
});
