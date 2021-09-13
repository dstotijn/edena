import { DateTime } from "luxon";
import Link from "next/link";
import { useRouter } from "next/router";

import { HttpLogEntry } from "../../pages/http-logs";

type Props = {
  httpLogEntry: HttpLogEntry;
};

export function HttpLogListItem({ httpLogEntry }: Props): JSX.Element {
  const router = useRouter();
  const url = {
    pathname: "/http-logs",
    query: {
      hostId: httpLogEntry.hostId,
      id: httpLogEntry.id,
    },
  };
  const selectedItem = router.query.id === httpLogEntry.id;

  const createdAt = DateTime.fromISO(httpLogEntry.createdAt);

  return (
    <li className={selectedItem ? "bg-indigo-50" : ""}>
      <Link href={url}>
        <a className="block px-6 py-4">
          <h3 className="font-bold">
            {httpLogEntry.request.method} {httpLogEntry.request.url}
          </h3>
          <p>{createdAt.toLocaleString(DateTime.DATETIME_MED_WITH_SECONDS)}</p>
          <p className="text-gray-400">{createdAt.toRelative()}</p>
        </a>
      </Link>
    </li>
  );
}

export default HttpLogListItem;
