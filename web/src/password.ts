// Password rules shown live while typing. Keep in sync with
// internal/auth/password.go, which enforces the same rules on the server.

export const MIN_PASSWORD_LENGTH = 10
/** bcrypt's limit, in UTF-8 bytes (accented letters and emoji take 2–4 bytes) */
export const MAX_PASSWORD_BYTES = 72

const COMMON_WORDS = new Set([
  'password', 'passwrd', 'pass', 'admin', 'administrator',
  'qwerty', 'qwertyuiop', 'asdf', 'asdfgh', 'zxcvbn',
  'letmein', 'welcome', 'iloveyou', 'monkey', 'dragon',
  'sunshine', 'princess', 'football', 'master', 'secret',
  'rahasia', 'sandi', 'katasandi', 'indonesia', 'manado',
  'jemaat', 'kolom', 'gereja', 'siloam', 'tuhan',
  'yesus', 'jesus', 'haleluya', 'amin',
])

const UPPER = /\p{Lu}/u
const LOWER = /\p{Ll}/u
const DIGIT = /\p{Nd}/u
const SYMBOL = /[^\p{L}\p{Nd}\s]/u

const lettersOnly = (p: string) => p.replace(/[^\p{L}]/gu, '').toLowerCase()

const containsUsername = (p: string, username: string) => {
  const u = username.trim().toLowerCase()
  return u.length >= 3 && p.toLowerCase().includes(u)
}

export type RuleResult = { id: string; label: string; ok: boolean }

export function checkRules(password: string, username = ''): RuleResult[] {
  return [
    { id: 'length', label: `Minimal ${MIN_PASSWORD_LENGTH} karakter`, ok: [...password].length >= MIN_PASSWORD_LENGTH },
    { id: 'case', label: 'Huruf besar dan huruf kecil', ok: UPPER.test(password) && LOWER.test(password) },
    { id: 'digit', label: 'Minimal satu angka', ok: DIGIT.test(password) },
    { id: 'symbol', label: 'Minimal satu simbol (mis. ! @ # ?)', ok: SYMBOL.test(password) },
    { id: 'username', label: 'Tidak mengandung username', ok: !containsUsername(password, username) },
    { id: 'common', label: 'Bukan password yang umum dipakai', ok: !COMMON_WORDS.has(lettersOnly(password)) },
    // only shown when broken (see PasswordStrength)
    {
      id: 'max',
      label: `Maksimal ${MAX_PASSWORD_BYTES} byte (huruf beraksen dan emoji dihitung lebih dari satu)`,
      ok: new TextEncoder().encode(password).length <= MAX_PASSWORD_BYTES,
    },
  ]
}

export function passwordValid(password: string, username = '') {
  return checkRules(password, username).every((r) => r.ok)
}

export type Strength = { score: 0 | 1 | 2 | 3 | 4; label: string }

const LABELS = ['Sangat lemah', 'Lemah', 'Sedang', 'Kuat', 'Sangat kuat']
const SEQUENCES = ['abcdefghijklmnopqrstuvwxyz', '01234567890', 'qwertyuiop', 'asdfghjkl', 'zxcvbnm']

/** Count runs of 3+ characters that follow a keyboard/alphabet sequence (either direction). */
function sequenceRuns(p: string) {
  const s = p.toLowerCase()
  let runs = 0
  for (let i = 0; i + 3 <= s.length; i++) {
    const tri = s.slice(i, i + 3)
    const rev = [...tri].toReversed().join('')
    if (SEQUENCES.some((seq) => seq.includes(tri) || seq.includes(rev))) runs++
  }
  return runs
}

/**
 * Rough strength estimate: entropy from length × character pool, minus
 * penalties for repeats, sequences, common words and the username. Capped at
 * "Lemah" until every rule passes.
 */
export function passwordStrength(password: string, username = ''): Strength {
  if (!password) return { score: 0, label: '' }
  const chars = [...password]
  let pool = 0
  if (LOWER.test(password)) pool += 26
  if (UPPER.test(password)) pool += 26
  if (DIGIT.test(password)) pool += 10
  if (SYMBOL.test(password)) pool += 33
  if (/\s/.test(password)) pool += 1
  if (/[^\p{ASCII}]/u.test(password)) pool += 20

  // count distinct characters so "aaaaaaaa" is not as strong as its length
  let bits = new Set(chars).size * Math.log2(Math.max(pool, 2)) + (chars.length - new Set(chars).size) * 1.5
  bits -= (password.match(/(.)\1{2,}/gu)?.length ?? 0) * 8
  bits -= sequenceRuns(password) * 4
  if (COMMON_WORDS.has(lettersOnly(password))) bits = Math.min(bits, 20)
  if (containsUsername(password, username)) bits -= 15

  let score: Strength['score'] = bits < 28 ? 0 : bits < 40 ? 1 : bits < 60 ? 2 : bits < 80 ? 3 : 4
  if (!checkRules(password, username).every((r) => r.ok)) score = Math.min(score, 1) as Strength['score']
  return { score, label: LABELS[score] }
}

const rand = (n: number) => crypto.getRandomValues(new Uint32Array(1))[0] % n

/** Random password that always satisfies the rules (no look-alike characters). */
export function generatePassword(length = 14) {
  const sets = ['ABCDEFGHJKLMNPQRSTUVWXYZ', 'abcdefghijkmnpqrstuvwxyz', '23456789', '!@#$%&*?-_+=']
  const all = sets.join('')
  const out = sets.map((s) => s[rand(s.length)])
  while (out.length < length) out.push(all[rand(all.length)])
  for (let i = out.length - 1; i > 0; i--) {
    const j = rand(i + 1)
    ;[out[i], out[j]] = [out[j], out[i]]
  }
  return out.join('')
}
