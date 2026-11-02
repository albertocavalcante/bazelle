/**
 * Process helpers for spawning the bazelle CLI. Pure, so they are testable.
 */

/**
 * How to spawn `binaryPath`. Executables run directly (arguments are passed
 * literally); only Windows batch shims need a shell, and then the path is
 * quoted so spaces (e.g. "Program Files") survive cmd.exe parsing.
 */
export function commandFor(
  binaryPath: string,
  platform: NodeJS.Platform = process.platform,
): { command: string; shell: boolean } {
  if (platform === "win32" && /\.(cmd|bat)$/iu.test(binaryPath)) {
    return { command: `"${binaryPath}"`, shell: true };
  }
  return { command: binaryPath, shell: false };
}

/** Expand a leading `~` path segment (not `~user`) to `home`. */
export function expandHome(path: string, home: string): string {
  return /^~(?=$|[\\/])/u.test(path) ? home + path.slice(1) : path;
}
