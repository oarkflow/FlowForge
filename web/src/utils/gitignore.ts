/**
 * Lightweight .gitignore pattern matcher.
 *
 * Supports:
 *  - Comments (#) and blank lines
 *  - Negation (!)
 *  - Directory-only patterns (trailing /)
 *  - Leading / for root-relative patterns
 *  - ** for multi-directory matching
 *  - * for single-directory glob
 *  - ? for single character
 *
 * Always ignores `.git/` regardless of patterns.
 */

// Well-known directories / files that should always be ignored when uploading
// a project folder (even without a .gitignore).
const DEFAULT_IGNORE_PATTERNS = [
  '.git',
  'node_modules',
  '.DS_Store',
  'Thumbs.db',
  '__pycache__',
  '.pytest_cache',
  '.mypy_cache',
  '.tox',
  '.venv',
  'venv',
  '.env.local',
  '.idea',
  '.vscode',
  '*.pyc',
  '*.pyo',
  '*.class',
  '*.o',
  '*.so',
  '*.dylib',
];

interface Rule {
  /** The original pattern (trimmed). */
  pattern: string;
  /** Regex compiled from the pattern. */
  regex: RegExp;
  /** If true, this rule negates a previous match. */
  negation: boolean;
  /** If true, only matches directories (trailing / in pattern). */
  dirOnly: boolean;
}

function escapeRegex(s: string): string {
  return s.replace(/[.*+?^${}()|[\]\\]/g, '\\$&');
}

function patternToRegex(pattern: string): RegExp {
  let p = pattern;

  // Leading / means anchored to root — strip it but remember
  const anchored = p.startsWith('/');
  if (anchored) p = p.slice(1);

  // Split into segments on /
  const segments = p.split('/').filter(Boolean);

  const regParts: string[] = [];
  for (const seg of segments) {
    if (seg === '**') {
      regParts.push('(?:.+/)?');
    } else {
      let r = '';
      for (let i = 0; i < seg.length; i++) {
        const ch = seg[i];
        if (ch === '*') {
          r += '[^/]*';
        } else if (ch === '?') {
          r += '[^/]';
        } else {
          r += escapeRegex(ch);
        }
      }
      regParts.push(r + '/');
    }
  }

  let regStr = regParts.join('');
  // The last segment should optionally match without trailing slash
  // (so it matches both files and directories).
  if (regStr.endsWith('/')) {
    regStr = regStr.slice(0, -1) + '(?:/|$)';
  }

  // If not anchored AND pattern has no slash separators, match anywhere in path
  if (!anchored && segments.length === 1) {
    regStr = '(?:^|/)' + regStr;
  } else {
    regStr = '^' + regStr;
  }

  return new RegExp(regStr);
}

function parseRule(line: string): Rule | null {
  let trimmed = line.trim();
  if (!trimmed || trimmed.startsWith('#')) return null;

  let negation = false;
  if (trimmed.startsWith('!')) {
    negation = true;
    trimmed = trimmed.slice(1);
  }

  let dirOnly = false;
  if (trimmed.endsWith('/')) {
    dirOnly = true;
    trimmed = trimmed.slice(0, -1);
  }

  if (!trimmed) return null;

  return {
    pattern: trimmed,
    regex: patternToRegex(trimmed),
    negation,
    dirOnly,
  };
}

export interface GitignoreMatcher {
  /** Returns true if the given path should be ignored. */
  isIgnored(filePath: string): boolean;
}

/**
 * Creates a matcher from raw .gitignore content.
 *
 * @param content  The raw text of a .gitignore file.
 * @param includeDefaults  If true, prepend well-known ignore patterns (.git, node_modules, etc).
 */
export function createGitignoreMatcher(content: string, includeDefaults = true): GitignoreMatcher {
  const rules: Rule[] = [];

  // Add default patterns first (lowest priority)
  if (includeDefaults) {
    for (const p of DEFAULT_IGNORE_PATTERNS) {
      const r = parseRule(p);
      if (r) rules.push(r);
    }
  }

  // Parse .gitignore lines
  for (const line of content.split('\n')) {
    const rule = parseRule(line);
    if (rule) rules.push(rule);
  }

  return {
    isIgnored(filePath: string): boolean {
      // Normalise: remove leading /
      const normalized = filePath.startsWith('/') ? filePath.slice(1) : filePath;
      // Remove the root folder name from the path (the folder the user selected)
      // so patterns match relative to the project root.
      const slashIdx = normalized.indexOf('/');
      const relativePath = slashIdx >= 0 ? normalized.slice(slashIdx + 1) : normalized;

      if (!relativePath) return false;

      let ignored = false;
      for (const rule of rules) {
        if (rule.dirOnly) {
          // For dir-only rules, check if the path has it as a directory segment
          // by appending / to each directory component
          const withTrailingSlash = relativePath + '/';
          if (rule.regex.test(withTrailingSlash) || rule.regex.test(relativePath)) {
            ignored = !rule.negation;
          }
        } else {
          if (rule.regex.test(relativePath)) {
            ignored = !rule.negation;
          }
        }
      }
      return ignored;
    },
  };
}

/**
 * Creates a matcher using only the default well-known patterns
 * (no .gitignore content).
 */
export function createDefaultMatcher(): GitignoreMatcher {
  return createGitignoreMatcher('', true);
}
