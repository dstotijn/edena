import useSWR from "swr";

import { fetcher } from "../lib/fetcher";
import { HttpLogEntry } from "../types/HttpLogEntry";

type Response = {
  data?: HttpLogEntry[];
  error?: {
    message: string;
  };
};

export function useHttpLogs(hostId?: string): { httpLogs?: HttpLogEntry[]; error: any } {
  const { data, error } = useSWR<Response | undefined, any>(
    ["/api/http-logs/", hostId],
    (url: string, hostId: string) => {
      if (!hostId) {
        return;
      }
      return fetcher(`${url}?${new URLSearchParams({ hostId })}`);
    }
  );

  if (error) {
    return { error };
  }

  return { httpLogs: data?.data, error: data?.error };
}
