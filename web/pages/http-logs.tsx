import type { NextPage } from "next";
import { useRouter } from "next/router";
import useSWR from "swr";

import { HttpLogDetail } from "../components/http-logs/HttpLogDetail";
import { HttpLogListItem } from "../components/http-logs/HttpLogListItem";
import { HostsNav } from "../components/HostsNav";

async function fetcher(...args: Parameters<typeof fetch>): Promise<any> {
  const res = await fetch(...args);
  return res.json();
}

type HttpHeaders = Record<string, string[]>;

export type HttpLogEntry = {
  id: string;
  hostId: string;
  request: {
    host: string;
    url: string;
    method: string;
    headers: HttpHeaders;
    body: string;
    raw: string;
  };
  response: {
    statusCode: number;
    status: string;
    headers: HttpHeaders;
    body: string;
    raw: string;
  };
  createdAt: string;
};

type HttpLogsData = {
  data: HttpLogEntry[];
};

const HttpLogs: NextPage = () => {
  const router = useRouter();
  const { hostId, id } = router.query;
  const { data: d, error } = useSWR<HttpLogsData, any>(
    ["/api/http-logs/", hostId],
    (url, hostId) => {
      if (!hostId) {
        return { data: [] };
      }
      return fetcher(`${url}?${new URLSearchParams({ hostId })}`);
    }
  );
  const data = d?.data;

  const logEntry = data?.find((logEntry) => logEntry.id === id);
  const logs = (
    <div className="flex divide-x">
      <div className="flex-shrink-0 w-1/4">
        <ul className="divide-y">
          {data?.map((logEntry) => (
            <HttpLogListItem key={logEntry.id} httpLogEntry={logEntry} />
          ))}
        </ul>
      </div>
      <div className="flex-auto p-8">
        {logEntry && <HttpLogDetail httpLogEntry={logEntry} />}
        {data && data.length > 0 && !id && <p>Select a log entry...</p>}
        {id && !logEntry && <p>Log entry not found.</p>}
      </div>
    </div>
  );

  return (
    <>
      <HostsNav />
      <main className="clear-both">
        {error && <div className="p-6">Failed to load: {error.message}</div>}
        {!(data || error) && <div className="p-6">Loading ...</div>}
        {data && logs}
      </main>
    </>
  );
};

export default HttpLogs;
