import { ReactNode } from "react";

type Props = {
  children: ReactNode;
};

export function Layout({ children }: Props) {
  return <div className="container pt-4 mx-auto">{children}</div>;
}

export default Layout;
