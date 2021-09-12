import type { NextPage } from "next";
import { useRouter } from "next/router";
import useSWR from "swr";
import { HttpLogDetail } from "../components/http-logs/HttpLogDetail";
import { HttpLogListItem } from "../components/http-logs/HttpLogListItem";

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
  const { data, error } = useSWR<HttpLogsData, any>(
    ["/api/http-logs/", hostId],
    (url, hostId) => {
      if (!hostId) {
        return { data: [] };
      }
      return fetcher(`${url}?${new URLSearchParams({ hostId })}`);
    }
  );

  if (error) {
    return <div>Failed to load: {error.message}</div>;
  }
  if (!data) {
    return <div>Loading ...</div>;
  }

  const logEntry = data.data.find((logEntry) => logEntry.id === id);
  if (!logEntry) {
    return <div>Log entry not found.</div>;
  }

  return (
    <div className="flex divide-x">
      <div className="flex-shrink-0 w-1/4">
        <ul className="divide-y">
          {data.data.map((logEntry) => (
            <HttpLogListItem key={logEntry.id} httpLogEntry={logEntry} />
          ))}
        </ul>
      </div>
      <div className="flex-auto p-8">
        <HttpLogDetail httpLogEntry={logEntry} />
      </div>
    </div>
  );
};

export default HttpLogs;
