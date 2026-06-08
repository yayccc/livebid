const longIDFieldPattern =
  /("(?:id|[A-Za-z0-9_]+_id|[A-Za-z0-9_]+(?:Id|ID))"\s*:\s*)(-?\d{16,})(?=\s*[,}\]])/g

export function parseJsonPreservingIDs<T>(text: string): T {
  return JSON.parse(quoteLongIDNumbers(text)) as T
}

function quoteLongIDNumbers(json: string) {
  return json.replace(longIDFieldPattern, '$1"$2"')
}
