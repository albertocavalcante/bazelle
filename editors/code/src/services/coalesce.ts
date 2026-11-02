/**
 * Per-key single-flight: while a run for a key is in progress, further
 * triggers for that key collapse into one follow-up run.
 */
export function coalesce<K>(run: (key: K) => Promise<void>): (key: K) => void {
  const state = new Map<K, "running" | "rerun">();

  const start = (key: K): void => {
    state.set(key, "running");
    run(key)
      .catch(() => {
        // Callers report their own errors; a failure must not wedge the key.
      })
      .finally(() => {
        if (state.get(key) === "rerun") start(key);
        else state.delete(key);
      });
  };

  return (key) => {
    if (state.has(key)) state.set(key, "rerun");
    else start(key);
  };
}
