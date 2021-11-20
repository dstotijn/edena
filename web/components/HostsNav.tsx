import { useRouter } from "next/router";

import { useHost } from "../hooks/useHost";
import MenuItem from "./MenuItem";
import { NavBar } from "./NavBar";

export function HostsNav(): JSX.Element | null {
  const router = useRouter();
  const { hostId } = router.query;

  const { host } = useHost(hostId as string | undefined);

  const menuMap = new Map([
    ["Overview", "/host"],
    ["HTTP logs", "/http-logs"],
    ["DNS logs", "/dns-logs"],
    ["SMTP logs", "/smtp-logs"],
  ]);

  const menuItems: JSX.Element[] = [];

  menuMap.forEach((pathname, title) => {
    menuItems.push(
      <MenuItem
        key={pathname}
        text={title}
        href={{
          pathname,
          query: { hostId },
        }}
      />
    );
  });

  return (
    <div className="float-left w-full border-b">
      {host && (
        <div className="float-right py-3">
          <strong>Host:</strong> {host.hostname}
        </div>
      )}
      <NavBar>{menuItems}</NavBar>
    </div>
  );
}
