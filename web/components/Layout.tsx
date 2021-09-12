import { ReactNode } from "react";
import Link from "next/link";
import { useRouter } from "next/router";

type Props = {
  children: ReactNode;
};

type MenuItemProps = {
  text: string;
  href: string;
};

function MenuItem({ text, href }: MenuItemProps): JSX.Element {
  const router = useRouter();
  const selectedItem = router.asPath === href;

  return (
    <Link href={href}>
      <a
        className={`block ${
          selectedItem ? "bg-gray-100" : "hover:bg-gray-100"
        } px-4 py-2 rounded transition-all duration-500`}
      >
        {text}
      </a>
    </Link>
  );
}

export function Layout({ children }: Props) {
  return (
    <div className="container pt-4 mx-auto">
      <header className="float-left w-full pl-6 pb-4 border-b">
        <Link href="/">
          <a>
            <h1 className="float-left text-indigo-600 text-4xl font-bold">
              Edena
            </h1>
          </a>
        </Link>
        <div className="float-left mt-1 ml-8 flex flex-row space-x-2">
          <MenuItem text="Overview" href="/" />
          <MenuItem text="HTTP logs" href="/http-logs/" />
          <MenuItem text="DNS logs" href="/dns-logs/" />
          <MenuItem text="SMTP logs" href="/smtp-logs/" />
        </div>
      </header>
      <main className="clear-both">{children}</main>
    </div>
  );
}

export default Layout;
