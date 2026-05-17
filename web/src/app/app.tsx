import { AppBootLoader } from "@/app/boot-loader";
import { AppProviders } from "@/app/providers";
import { AppRouter } from "@/app/router";

export function App() {
  return (
    <AppProviders>
      <AppBootLoader>
        <AppRouter />
      </AppBootLoader>
    </AppProviders>
  );
}
