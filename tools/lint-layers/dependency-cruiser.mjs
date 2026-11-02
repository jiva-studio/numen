import process from "node:process";
import { createCruiseConfig } from "./checks/dependencies.mjs";

export default createCruiseConfig(process.cwd());
