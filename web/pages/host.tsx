import { DateTime } from "luxon";
import type { NextPage } from "next";
import Link from "next/link";
import { useRouter } from "next/router";
import React from "react";

import { HostsNav } from "../components/HostsNav";
import { useHost } from "../hooks/useHost";
import { useHttpLogs } from "../hooks/useHttpLogs";

const Host: NextPage = () => {
  const router = useRouter();
  const { hostId } = router.query;
  const { host, error } = useHost(hostId as string | undefined);
  const createdAt = host && DateTime.fromISO(host.createdAt);

  return (
    <>
      <HostsNav />
      <main className="p-6 clear-both">
        {error && (
          <p>
            Failed to load host: <em>{error.message}</em>
          </p>
        )}
        {!error && <h1 className="text-3xl font-bold mb-4">{host?.hostname}</h1>}
        {createdAt && <p className="mb-4">Created: {createdAt.toLocaleString(DateTime.DATETIME_MED_WITH_SECONDS)}</p>}
        <p className="mb-8">
          <Link href={{ pathname: "/http-logs", query: { hostId } }}>
            <a className="text-link">← Back to hosts</a>
          </Link>
        </p>
        <HttpLogs hostId={hostId as string | undefined} />
      </main>
    </>
  );
};

function HttpLogs({ hostId }: { hostId?: string }): JSX.Element {
  const { httpLogs, error } = useHttpLogs(hostId);
  const httpLogCount = httpLogs?.length || 0;
  const latestLog = httpLogs?.length ? httpLogs[httpLogs.length - 1] : undefined;
  const latestLogCreatedAt = latestLog ? DateTime.fromISO(latestLog.createdAt) : undefined;

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
