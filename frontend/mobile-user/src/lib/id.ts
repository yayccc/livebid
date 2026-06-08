export function idToJsonNumberLiteral(value: string | number) {
  const text = String(value).trim()
  return /^[1-9]\d*$/.test(text) ? text : null
}
