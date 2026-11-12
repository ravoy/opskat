type Result<T> = T extends (infer Item)[] ? NonNullable<Item>[] : NonNullable<T>;
type Client<T, Nullable extends keyof T> = {
  [Key in keyof T]: T[Key] extends (...args: infer Args) => PromiseLike<infer Value>
    ? (
        ...args: { [Index in keyof Args]: NonNullable<Args[Index]> }
      ) => Promise<Key extends Nullable ? Value : Value extends void ? void : Result<Value>>
    : never;
};

// Domain clients expose ordinary promises and enforce their successful-result
// contract once at the IPC boundary. Only explicitly nullable operations (for
// example cancelling a file picker) accept a successful null result.
export function createIPCClient<T extends object, Key extends keyof T = never>(
  native: T,
  nullable: readonly Key[] = []
): Client<T, Key> {
  const optional = new Set<keyof T>(nullable);
  return Object.fromEntries(
    Object.entries(native).map(([name, method]) => [
      name,
      async (...args: unknown[]) => {
        const result: unknown = await method(...args);
        if (result === null && !optional.has(name as keyof T)) {
          throw new Error(`${name}: backend returned no result`);
        }
        if (Array.isArray(result) && result.some((item) => item === null)) {
          throw new Error(`${name}: backend returned a null list item`);
        }
        return result;
      },
    ])
  ) as Client<T, Key>;
}
