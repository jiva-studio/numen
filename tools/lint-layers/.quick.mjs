import { Linter } from 'eslint'
import tseslint from 'typescript-eslint'
import vueParser from 'vue-eslint-parser'
import numen from '/home/akd/Projects/jiva-studio/numen/resources/worktrees/pkg-C2/tools/lint-layers/numen-rules.mjs'
import fs from 'node:fs'; import {execSync} from 'node:child_process'
const files=execSync('git ls-files -- "apps/**/*.vue" "libs/**/*.vue" "plugins/**/*.vue" "apps/**/*.ts" "libs/**/*.ts" "plugins/**/*.ts"',{encoding:'utf8',maxBuffer:1<<28}).split('\n').filter(f=>f&&!f.includes('node_modules')&&!/_pb\.ts$/.test(f))
const rules={'numen/vue-script-sections':'error','numen/handler-naming':'error','numen/no-mock-of-subject-slice':'error','numen/max-composable-return-properties':['error',{max:8}],'numen/no-style-declaration-read':'error'}
const counts={}
const linter=new Linter({cwd:process.cwd()})
for(const f of files){
  const isVue=f.endsWith('.vue'); const isTest=/\.test\.ts$/.test(f); const isStory=/\.(stories|spec)\.ts$/.test(f)||/\/(testing|fixtures)\//.test(f)
  const r={}
  if(isVue){r['numen/vue-script-sections']='error';r['numen/handler-naming']='error'}
  if(isTest){r['numen/no-mock-of-subject-slice']='error';r['numen/no-style-declaration-read']='error'}
  if(!isTest&&!isStory){r['numen/max-composable-return-properties']=['error',{max:8}]}
  const cfg=[{files:['**/*.ts','**/*.vue'],plugins:{numen},languageOptions:{parser:isVue?vueParser:tseslint.parser,parserOptions:{parser:tseslint.parser,sourceType:'module'}},rules:r}]
  const msgs=linter.verify(fs.readFileSync(f,'utf8'),cfg,{filename:f})
  for(const m of msgs){ if(m.fatal){console.log('FATAL',f,m.message);continue}; (counts[m.ruleId]??=[]).push(`${f}:${m.line} ${m.messageId}`)}
}
for(const [k,v] of Object.entries(counts)){console.log('##',k,v.length); if(process.argv[2]===k) console.log(v.join('\n'))}
