import type { NextPage } from "next";

import { HostsNav } from "../components/HostsNav";

const Host: NextPage = () => {
  return (
    <>
      <HostsNav />
      <div className="p-6 clear-both">Host overview page…</div>
    </>
  );
};

export default Host;
