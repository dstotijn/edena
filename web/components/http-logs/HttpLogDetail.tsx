import { DateTime } from "luxon";

import { HttpLogEntry } from "../../types/HttpLogEntry";

type Props = {
  httpLogEntry: HttpLogEntry;
};

export function HttpLogDetail({ httpLogEntry }: Props): JSX.Element {
  const createdAt = DateTime.fromISO(httpLogEntry.createdAt);

  return (
    <div>
      <h1 className="text-3xl font-bold mb-4 float-left">
        {httpLogEntry.request.method} {httpLogEntry.request.url}
      </h1>

      <p className="mb-4 clear-left">
        {createdAt.toLocaleString(DateTime.DATETIME_MED_WITH_SECONDS)}
        <span className="text-gray-400 ml-4">{createdAt.toRelative()}</span>
      </p>

      <h2 className="text-2xl font-bold mb-4">Request</h2>
      <pre className="text-sm text-indigo-200 bg-primary rounded-xl p-4 mb-4">{atob(httpLogEntry.request.raw)}</pre>

      <h2 className="text-2xl font-bold mb-4">Response</h2>
      <pre className="text-sm text-indigo-200 bg-primary rounded-xl p-4 mb-4">{atob(httpLogEntry.response.raw)}</pre>
    </div>
  );
}

export default HttpLogDetail;
