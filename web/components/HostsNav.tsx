import { useRouter } from "next/router";

import MenuItem from "./MenuItem";
import { NavBar } from "./NavBar";

export function HostsNav(): JSX.Element | null {
  const router = useRouter();
  const { hostId } = router.query;

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

  return <NavBar>{menuItems}</NavBar>;
}
