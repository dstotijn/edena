import type { AppProps } from "next/app";

import Layout from "../components/Layout";
import "../styles/globals.css";

function App({ Component, pageProps }: AppProps): JSX.Element {
  return (
    <Layout>
      <Component {...pageProps} />
    </Layout>
  );
}

export default App;
