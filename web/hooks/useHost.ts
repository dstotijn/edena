import useSWR from "swr";

import { fetcher } from "../lib/fetcher";

type Host = {
  id: string;
  hostname: string;
  createdAt: string;
};

type Response = {
  data?: Host;
  error?: {
    message: string;
  };
};

export function useHost(hostId?: string): { host?: Host; error: any } {
  const { data, error } = useSWR<Response | undefined, any>(["/api/hosts", hostId], (url: string, hostId: string) => {
    if (!hostId) {
      return;
    }
    return fetcher(`${url}/${hostId}`);
  });

  if (error) {
    return { error };
  }

  return { host: data?.data, error: data?.error };
}
