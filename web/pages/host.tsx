import { DateTime } from "luxon";
import type { NextPage } from "next";
import Link from "next/link";
import { useRouter } from "next/router";
import React from "react";

import { HostsNav } from "../components/HostsNav";
import { useHttpLogs } from "../hooks/useHttpLogs";

const Host: NextPage = () => {
  const router = useRouter();
  const { hostId } = router.query;

  return (
    <>
      <HostsNav />
      <main className="p-6 clear-both">
        <HttpLogs hostId={hostId as string} />
      </main>
    </>
  );
};

function HttpLogs({ hostId }: { hostId: string }): JSX.Element {
  const { httpLogs, error } = useHttpLogs(hostId);
  const httpLogCount = httpLogs?.length || 0;
  const latestLog = httpLogs?.length ? httpLogs[httpLogs.length - 1] : undefined;
  const latestLogCreatedAt = latestLog ? DateTime.fromISO(latestLog.createdAt) : undefined;

  <svg xmlns="http://www.w3.org/2000/svg" className="h-6 w-6" fill="none" viewBox="0 0 24 24" stroke="currentColor">
    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M14 5l7 7m0 0l-7 7m7-7H3" />
  </svg>;

  return (
    <>
      <h2 className="text-2xl font-bold mb-4">HTTP logs</h2>
      {error && (
        <p>
          Failed to load HTTP logs: <em>{error.message}</em>
        </p>
      )}
      {!(httpLogs || error) && <p>Loading ...</p>}
      {httpLogs && (
        <>
          <p className="mb-4">
            {httpLogCount} log {httpLogCount === 1 ? "entry" : "entries"}.{" "}
            {latestLogCreatedAt && (
              <>
                Last log: <span className="font-semibold">{latestLogCreatedAt.toRelative()}</span>
              </>
            )}
          </p>
          <p>
            <Link href={{ pathname: "/http-logs", query: { hostId } }}>
              <a className="text-link">→ Browse logs</a>
            </Link>
          </p>
        </>
      )}
    </>
  );
}

export default Host;
