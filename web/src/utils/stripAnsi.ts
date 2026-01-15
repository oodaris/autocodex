/* eslint-disable no-control-regex */
const csiRegex = new RegExp('\\x1b\\[[0-9;:?><=]*[ -/]*[@-~]', 'g')
const oscRegex = new RegExp('\\x1b\\][^\\x07\\x1b]*(?:\\x07|\\x1b\\\\)', 'g')
const otherRegex = new RegExp('\\x1b[PX^_][\\s\\S]*?\\x1b\\\\', 'g')
const singleEscRegex = new RegExp('\\x1b.', 'g')
/* eslint-enable no-control-regex */

export function stripAnsi(input: string): string {
  let output = input
  output = output.replace(oscRegex, '')
  output = output.replace(otherRegex, '')
  output = output.replace(csiRegex, '')
  output = output.replace(singleEscRegex, '')
  return output
}

export function normalizeTerminalOutput(input: string): string {
  let output = input.replace(/\r/g, '')
  output = output.replace(/\n{3,}/g, '\n\n')

  const lines = output.split('\n')
  const nonEmpty = lines.filter((line) => line !== '')
  if (nonEmpty.length === 0) {
    return output
  }
  const singleCharLines = nonEmpty.filter((line) => line.length === 1)
  if (singleCharLines.length / nonEmpty.length >= 0.6) {
    return nonEmpty.join('')
  }
  return output
}
