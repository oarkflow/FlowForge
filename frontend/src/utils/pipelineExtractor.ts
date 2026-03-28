import yaml from 'js-yaml';
import type { ExtractedVariable } from '../types';

export interface ExtractionResult {
  envVars: ExtractedVariable[];
  secrets: ExtractedVariable[];
}

// Shell built-in variables that should not be flagged as needing configuration.
const SHELL_BUILTINS = new Set([
  'HOME', 'PATH', 'PWD', 'USER', 'SHELL', 'TERM', 'LANG', 'HOSTNAME',
  'LOGNAME', 'OLDPWD', 'TMPDIR', 'EDITOR', 'VISUAL', 'RANDOM', 'SECONDS',
  'LINENO', 'FUNCNAME', 'BASH_SOURCE', 'BASH_LINENO', 'PIPESTATUS', 'IFS',
  'PS1', 'PS2', 'UID', 'EUID', 'PPID', 'SHLVL', 'MACHTYPE', 'OSTYPE',
  'CI', 'FLOWFORGE_ENV',
]);

// Regex patterns.
const SECRET_EXPR_RE = /\$\{\{\s*secrets\.([A-Za-z_][A-Za-z0-9_]*)\s*\}\}/g;
const ENV_EXPR_RE = /\$\{\{\s*env\.([A-Za-z_][A-Za-z0-9_]*)\s*\}\}/g;
const SHELL_DEREF_RE = /\$\{([A-Z_][A-Z0-9_]*)(?::-[^}]*)?\}/g;
const PURE_SHELL_DEREF_RE = /^\s*"\$\{([A-Z_][A-Z0-9_]*)\}"\s*$|^\s*\$\{([A-Z_][A-Z0-9_]*)\}\s*$/;
const LOCAL_DEF_RE = /^[^\S\n]*([A-Z_][A-Z0-9_]*)=/gm;

interface PipelineSpec {
  env?: Record<string, string>;
  jobs?: Record<string, JobSpec>;
}

interface JobSpec {
  env?: Record<string, string>;
  image?: string;
  when?: string;
  steps?: StepSpec[];
}

interface StepSpec {
  name?: string;
  run?: string;
  uses?: string;
  if?: string;
  with?: Record<string, string>;
  env?: Record<string, string>;
}

/**
 * Extracts environment variable and secret references from pipeline YAML.
 * Mirrors the Go backend ExtractVariables logic.
 */
export function extractPipelineVariables(yamlStr: string): ExtractionResult {
  const result: ExtractionResult = { envVars: [], secrets: [] };
  if (!yamlStr.trim()) return result;

  let spec: PipelineSpec;
  try {
    spec = yaml.load(yamlStr) as PipelineSpec;
  } catch {
    return result;
  }
  if (!spec || typeof spec !== 'object') return result;

  const vars = new Map<string, ExtractedVariable>();
  const staticEnv = new Set<string>();

  // --- Category A: Walk all env: blocks ---
  walkEnvMap(spec.env, 'pipeline env', vars, staticEnv);

  if (spec.jobs) {
    for (const [jobName, job] of Object.entries(spec.jobs)) {
      walkEnvMap(job.env, `job ${jobName} env`, vars, staticEnv);
      if (job.steps) {
        for (const step of job.steps) {
          const src = step.name ? `step ${step.name}` : 'step';
          walkEnvMap(step.env, `${src} env`, vars, staticEnv);
        }
      }
    }
  }

  // --- Category B: Scan all string fields for ${{ secrets.NAME }} and ${{ env.NAME }} ---
  const scanStrings: Array<{ value: string; source: string }> = [];

  if (spec.env) {
    for (const [k, v] of Object.entries(spec.env)) {
      scanStrings.push({ value: String(v), source: `pipeline env ${k}` });
    }
  }

  if (spec.jobs) {
    for (const [jobName, job] of Object.entries(spec.jobs)) {
      if (job.env) {
        for (const [k, v] of Object.entries(job.env)) {
          scanStrings.push({ value: String(v), source: `job ${jobName} env ${k}` });
        }
      }
      if (job.image) scanStrings.push({ value: job.image, source: `job ${jobName} image` });
      if (job.when) scanStrings.push({ value: job.when, source: `job ${jobName} when` });

      if (job.steps) {
        for (const step of job.steps) {
          const stepSrc = step.name ? `step ${step.name}` : 'step';
          if (step.run) scanStrings.push({ value: step.run, source: `${stepSrc} run` });
          if (step.if) scanStrings.push({ value: step.if, source: `${stepSrc} if` });
          if (step.uses) scanStrings.push({ value: step.uses, source: `${stepSrc} uses` });
          if (step.with) {
            for (const [wk, wv] of Object.entries(step.with)) {
              scanStrings.push({ value: String(wv), source: `${stepSrc} with.${wk}` });
            }
          }
          if (step.env) {
            for (const [ek, ev] of Object.entries(step.env)) {
              scanStrings.push({ value: String(ev), source: `${stepSrc} env ${ek}` });
            }
          }
        }
      }
    }
  }

  for (const s of scanStrings) {
    if (!s.value) continue;

    // Extract ${{ secrets.NAME }}.
    for (const m of s.value.matchAll(SECRET_EXPR_RE)) {
      promoteOrAdd(vars, m[1], 'secret', s.source, false);
    }
    // Extract ${{ env.NAME }}.
    for (const m of s.value.matchAll(ENV_EXPR_RE)) {
      addIfAbsent(vars, m[1], 'env_var', s.source, staticEnv.has(m[1]));
    }
  }

  // --- Category C: Scan step.Run for ${VAR_NAME} shell references ---
  if (spec.jobs) {
    for (const job of Object.values(spec.jobs)) {
      if (!job.steps) continue;
      for (const step of job.steps) {
        if (!step.run) continue;
        const stepSrc = step.name ? `step ${step.name}` : 'step';

        // Find locally-defined variables in this run block.
        const localDefs = new Set<string>();
        for (const m of step.run.matchAll(LOCAL_DEF_RE)) {
          localDefs.add(m[1]);
        }

        for (const m of step.run.matchAll(SHELL_DEREF_RE)) {
          const name = m[1];
          if (SHELL_BUILTINS.has(name) || localDefs.has(name)) continue;
          const existing = vars.get(name);
          if (existing?.type === 'secret') continue;
          addIfAbsent(vars, name, 'env_var', `${stepSrc} run`, staticEnv.has(name));
        }
      }
    }
  }

  // --- Build result ---
  for (const v of vars.values()) {
    if (v.type === 'secret') {
      result.secrets.push(v);
    } else {
      result.envVars.push(v);
    }
  }

  result.envVars.sort((a, b) => a.name.localeCompare(b.name));
  result.secrets.sort((a, b) => a.name.localeCompare(b.name));

  return result;
}

function walkEnvMap(
  envMap: Record<string, string> | undefined,
  source: string,
  vars: Map<string, ExtractedVariable>,
  staticEnv: Set<string>,
) {
  if (!envMap) return;
  for (const [key, rawValue] of Object.entries(envMap)) {
    const value = String(rawValue);
    const m = PURE_SHELL_DEREF_RE.exec(value);
    if (m) {
      const innerVar = m[1] || m[2];
      addIfAbsent(vars, innerVar, 'env_var', source, false);
      staticEnv.add(key);
      PURE_SHELL_DEREF_RE.lastIndex = 0;
      continue;
    }
    staticEnv.add(key);
    addIfAbsent(vars, key, 'env_var', source, true);
  }
}

function addIfAbsent(
  vars: Map<string, ExtractedVariable>,
  name: string,
  type: 'env_var' | 'secret',
  source: string,
  hasValue: boolean,
) {
  if (!vars.has(name)) {
    vars.set(name, { name, type, source, has_value: hasValue });
  }
}

function promoteOrAdd(
  vars: Map<string, ExtractedVariable>,
  name: string,
  type: 'env_var' | 'secret',
  source: string,
  hasValue: boolean,
) {
  const existing = vars.get(name);
  if (existing) {
    if (type === 'secret' && existing.type !== 'secret') {
      existing.type = 'secret';
      existing.source = source;
    }
    return;
  }
  vars.set(name, { name, type, source, has_value: hasValue });
}
