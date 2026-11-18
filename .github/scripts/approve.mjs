import { execFileSync } from "node:child_process";
import { readFileSync } from "node:fs";
import { getOwners, parseCodeowners } from "./codeowners.mjs";

const { REPO, PR_NUMBER, COMMENTER, CODEOWNERS_FILE = ".github/CODEOWNERS" } = process.env;
const [owner, name] = REPO.split("/");

function gh(...args) {
  return execFileSync("gh", args, { encoding: "utf8" });
}

function readJson(...args) {
  return JSON.parse(gh(...args));
}

function readChecks() {
  try {
    return JSON.parse(gh("pr", "checks", PR_NUMBER, "--repo", REPO, "--required", "--json", "name,bucket"));
  } catch (error) {
    if (error.stdout) return JSON.parse(error.stdout);
    throw error;
  }
}

const pr = readJson("pr", "view", PR_NUMBER, "--repo", REPO, "--json", "headRefOid,state,isDraft,reviews");
const problems = [];

if (pr.state !== "OPEN") problems.push("The pull request is not open.");
if (pr.isDraft) problems.push("The pull request is a draft.");

const files = gh("api", "--paginate", `repos/${REPO}/pulls/${PR_NUMBER}/files`, "--jq", ".[] | .filename, (.previous_filename // empty)")
  .split("\n")
  .filter(Boolean);
const rules = parseCodeowners(readFileSync(CODEOWNERS_FILE, "utf8"));
const commenter = COMMENTER.toLowerCase();
const notOwned = files.filter((file) => !getOwners(rules, file).includes(commenter));
if (notOwned.length > 0) {
  problems.push(`@${COMMENTER} is not an owner of: ${notOwned.slice(0, 10).join(", ")}${notOwned.length > 10 ? ", ..." : ""}.`);
}

const checks = readChecks();
const failing = checks.filter((check) => check.bucket !== "pass" && check.bucket !== "skipping");
if (checks.length === 0) problems.push("No required check is reported.");
if (failing.length > 0) problems.push(`Required checks are not green: ${failing.map((check) => check.name).join(", ")}.`);

const threads = readJson(
  "api", "graphql",
  "-f", "query=query($owner:String!,$name:String!,$number:Int!){repository(owner:$owner,name:$name){pullRequest(number:$number){reviewThreads(first:100){pageInfo{hasNextPage} nodes{isResolved}}}}}",
  "-f", `owner=${owner}`, "-f", `name=${name}`, "-F", `number=${PR_NUMBER}`,
).data.repository.pullRequest.reviewThreads;
const open = threads.nodes.filter((thread) => !thread.isResolved).length;
if (open > 0) problems.push(`${open} review thread(s) are not resolved.`);
if (threads.pageInfo.hasNextPage) problems.push("The pull request has more than 100 review threads.");

const reviewsOfClaude = pr.reviews.filter((review) => review.author.login.replace(/\[bot\]$/, "") === "claude");
const lastReview = reviewsOfClaude.at(-1);
if (!lastReview || lastReview.state !== "APPROVED" || lastReview.commit?.oid !== pr.headRefOid) {
  problems.push("The last commit has no approval from claude[bot].");
}

if (problems.length > 0) {
  gh("pr", "comment", PR_NUMBER, "--repo", REPO, "--body", `Not approved:\n\n${problems.map((problem) => `- ${problem}`).join("\n")}`);
  process.exit(0);
}

gh(
  "api", "-X", "POST", `repos/${REPO}/pulls/${PR_NUMBER}/reviews`,
  "-f", "event=APPROVE", "-f", `commit_id=${pr.headRefOid}`, "-f", `body=Approved on the comment of @${COMMENTER}.`,
);
