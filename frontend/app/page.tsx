import { HomeView } from "@/components/home/home-view";
import { SiteHeader } from "@/components/site-header";

/**
 * The landing page. The first screen leads with getting help: "Get
 * first-aid help" opens the chat at /patient/ with no sign-in, and 102 is
 * in the hero and stays in the header as you scroll.
 */
export default function Home() {
  return (
    <>
      <SiteHeader variant="home" />
      <main id="main">
        <HomeView />
      </main>
    </>
  );
}
