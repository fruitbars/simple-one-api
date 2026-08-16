export function takeDisplayChunk(text: string): [string, string] {
  if (text.length <= 4) return [text, ""];

  const newlineIndex = text.indexOf("\n");
  if (newlineIndex >= 0 && newlineIndex < 24) {
    return [text.slice(0, newlineIndex + 1), text.slice(newlineIndex + 1)];
  }

  const chunkSize = Math.min(24, Math.max(4, Math.ceil(text.length / 28)));
  return [text.slice(0, chunkSize), text.slice(chunkSize)];
}
