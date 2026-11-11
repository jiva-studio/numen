import process from "node:process";
import { createCruiseConfig } from "./checks/dependencies.mjs";
import { readLibraryMatrix } from "./checks/libraries.mjs";

export default createCruiseConfig(process.cwd(), readLibraryMatrix().packages);
