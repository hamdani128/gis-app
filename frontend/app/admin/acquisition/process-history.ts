export function dedupeHistory<T>(history: T[]): T[] {
  return [...new Map(history.map((record) => [JSON.stringify(record), record])).values()];
}
