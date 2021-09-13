import { useRouter } from "next/router";
import Link from "next/link";
import { UrlObject } from "url";

type Props = {
  text: string;
  href: string | UrlObject;
};

export function MenuItem({ text, href }: Props): JSX.Element {
  const router = useRouter();
  const selectedItem =
    typeof href === "string"
      ? router.asPath === href
      : router.pathname === href.pathname;
  const bgClass = selectedItem ? "bg-gray-100" : "hover:bg-gray-100";
  const classes = `block ${bgClass} px-4 py-2 rounded transition-all duration-500`;

  return (
    <Link href={href}>
      <a className={classes}>{text}</a>
    </Link>
  );
}

export default MenuItem;
