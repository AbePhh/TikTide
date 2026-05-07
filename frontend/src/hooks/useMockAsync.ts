import { useEffect, useState } from "react";

export function useMockAsync<T>(loader: () => Promise<T>, deps: unknown[] = []) {
  const [data, setData] = useState<T | null>(null);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    let active = true;
    setLoading(true);

    void loader().then((result) => {
      if (!active) {
        return;
      }
      setData(result);
      setLoading(false);
    });

    return () => {
      active = false;
    };
  }, deps);

  return { data, loading };
}
