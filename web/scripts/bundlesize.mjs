import { readdirSync, statSync } from 'node:fs'
import { join } from 'node:path'
import { fileURLToPath } from 'node:url'

const distDir = fileURLToPath(new URL('../dist', import.meta.url))

const maxJsKb = Number.parseInt(process.env.BUNDLE_MAX_JS_KB ?? '800', 10)
const maxCssKb = Number.parseInt(process.env.BUNDLE_MAX_CSS_KB ?? '80', 10)
const maxTotalKb = Number.parseInt(process.env.BUNDLE_MAX_TOTAL_KB ?? '900', 10)

function walk(dir) {
  const entries = readdirSync(dir, { withFileTypes: true })
  return entries.flatMap((entry) => {
    const full = join(dir, entry.name)
    if (entry.isDirectory()) return walk(full)
    return [full]
  })
}

let files
try {
  files = walk(distDir)
} catch {
  console.error('dist/ not found. Run `npm run build` before bundlesize.')
  process.exit(1)
}

const totals = { js: 0, css: 0 }

for (const file of files) {
  if (file.endsWith('.map')) continue
  const size = statSync(file).size
  if (file.endsWith('.js')) totals.js += size
  if (file.endsWith('.css')) totals.css += size
}

const jsKb = totals.js / 1024
const cssKb = totals.css / 1024
const totalKb = (totals.js + totals.css) / 1024

console.log(`Bundle size (raw JS+CSS): JS ${jsKb.toFixed(1)} KB, CSS ${cssKb.toFixed(1)} KB, total ${totalKb.toFixed(1)} KB`)

const failures = []
if (jsKb > maxJsKb) failures.push(`JS ${jsKb.toFixed(1)} KB > ${maxJsKb} KB`)
if (cssKb > maxCssKb) failures.push(`CSS ${cssKb.toFixed(1)} KB > ${maxCssKb} KB`)
if (totalKb > maxTotalKb) failures.push(`Total ${totalKb.toFixed(1)} KB > ${maxTotalKb} KB`)

if (failures.length > 0) {
  console.error('Bundle size limits exceeded:')
  for (const failure of failures) console.error(`- ${failure}`)
  process.exit(1)
}
