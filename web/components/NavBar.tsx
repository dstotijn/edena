import Link from "next/link";
import { ReactNode } from "react";

type Props = {
  children: ReactNode;
};

export function NavBar({ children }: Props): JSX.Element {
  return (
    <header className="float-left w-full pl-6 pb-4 border-b">
      <Link href="/">
        <a>
          <h1 className="float-left text-indigo-600 text-4xl font-bold">
            Edena
          </h1>
        </a>
      </Link>

      <div className="float-left mt-1 ml-8 flex flex-row space-x-2">
        {children}
      </div>
    </header>
  );
}

export default NavBar;
