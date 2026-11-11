import process from "node:process";
import { createCruiseConfig } from "../checks/structure/ts/dependency-graph/check.mjs";
import { readLibraryMatrix } from "../checks/structure/ts/libraries/check.mjs";

export default createCruiseConfig(process.cwd(), readLibraryMatrix().packages);
