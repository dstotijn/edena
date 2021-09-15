import type { NextPage } from "next";
import { useRouter } from "next/router";

import { HttpLogDetail } from "../components/http-logs/HttpLogDetail";
import { HttpLogListItem } from "../components/http-logs/HttpLogListItem";
import { HostsNav } from "../components/HostsNav";
import { useHttpLogs } from "../hooks/useHttpLogs";

const HttpLogs: NextPage = () => {
  const router = useRouter();
  const { hostId, id } = router.query;
  const { httpLogs, error } = useHttpLogs(hostId as string);

  const logEntry = httpLogs?.find((logEntry) => logEntry.id === id);
  const logs = (
    <div className="flex divide-x">
      <div className="flex-shrink-0 w-1/4">
        <ul className="divide-y">
          {httpLogs?.map((logEntry) => (
            <HttpLogListItem key={logEntry.id} httpLogEntry={logEntry} />
          ))}
        </ul>
      </div>
      <div className="flex-auto p-8">
        {logEntry && <HttpLogDetail httpLogEntry={logEntry} />}
        {httpLogs && httpLogs.length > 0 && !id && <p>Select a log entry...</p>}
        {id && !logEntry && <p>Log entry not found.</p>}
      </div>
    </div>
  );

  return (
    <>
      <HostsNav />
      <main className="clear-both">
        {error && (
          <div className="p-6">
            Failed to load HTTP logs: <em>{error.message}</em>
          </div>
        )}
        {!(httpLogs || error) && <div className="p-6">Loading ...</div>}
        {httpLogs && logs}
      </main>
    </>
  );
};

export default HttpLogs;
