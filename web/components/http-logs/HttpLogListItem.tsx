import { DateTime } from "luxon";
import { useRouter } from "next/router";
import { MouseEventHandler } from "react";

import { HttpLogEntry } from "../../pages/http-logs";

type Props = {
  httpLogEntry: HttpLogEntry;
};

export function HttpLogListItem({ httpLogEntry }: Props): JSX.Element {
  const router = useRouter();
  const href = `/http-logs/?${new URLSearchParams({
    hostId: httpLogEntry.hostId,
    id: httpLogEntry.id,
  })}`;
  const handleClick: MouseEventHandler = (e) => {
    e.preventDefault();
    router.push(href);
  };
  const selectedItem = router.asPath === href;

  const createdAt = DateTime.fromISO(httpLogEntry.createdAt);

  return (
    <li className={selectedItem ? "bg-indigo-50" : ""}>
      <a href="#" onClick={handleClick} className="block px-6 py-4">
        <h3 className="font-bold">
          {httpLogEntry.request.method} {httpLogEntry.request.url}
        </h3>
        <p>{createdAt.toLocaleString(DateTime.DATETIME_MED_WITH_SECONDS)}</p>
        <p className="text-gray-400">{createdAt.toRelative()}</p>
      </a>
    </li>
  );
}

export default HttpLogListItem;
