import type { NextPage } from "next";

import MenuItem from "../components/MenuItem";
import { NavBar } from "../components/NavBar";

const Home: NextPage = () => {
  return (
    <>
      <NavBar>
        <MenuItem text="Hosts" href="/hosts/" />
      </NavBar>
      <div className="p-6 clear-both">
        Welcome to <strong>Edena</strong>…
      </div>
    </>
  );
};

export default Home;
