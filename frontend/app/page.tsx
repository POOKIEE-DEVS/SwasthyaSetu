import { HomeView } from "@/components/home/home-view";
import { SiteHeader } from "@/components/site-header";

/**
 * Emergency first: the first screen is the first-aid chat, with the 102
 * button in the header and the hero. No sign-in, no menu to get through.
 */
export default function Home() {
  return (
    <>
      <SiteHeader />
      <main id="main">
        <HomeView />
      </main>
    </>
  );
}
