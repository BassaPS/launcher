import type { Page } from "@playwright/test";

// Test-only Wails HTTP boundary. No live game, accounts, or credentials are used.
export async function mockDesktop(page: Page, signedIn = true, serverNews: unknown[] = []) {
  await page.addInitScript(() => {
    // Wails normally injects this before either bundled runtime starts.
    Object.assign(window, { _wails: { environment: { OS: 'windows' } } });
    localStorage.setItem("i18nextLng", "ja");
    if (!localStorage.getItem("setting-storage")) localStorage.setItem(
      "setting-storage",
      JSON.stringify({
        version: 1,
        state: {
          locale: "ja",
          gamePath: "C:/Games/StarRail.exe",
          gameDir: "C:/Games",
          serverTarget: "hoyotoon",
          gameProfile: "starrail",
        },
      }),
    );
  });
  await page.route("**/wails/custom.js", (route) =>
    route.fulfill({ status: 404, body: "" }),
  );
  await page.route("**/wails/runtime**", (route) => {
    const url = new URL(route.request().url());
    const body =
      route.request().method() === "POST"
        ? route.request().postDataJSON()
        : {
            object: Number(url.searchParams.get("object")),
            args: JSON.parse(url.searchParams.get("args") || "{}"),
          };
    let result: unknown = null;
    if (body.object === 0) {
      const results: Record<number, unknown> = {
        941664977: [
          signedIn,
          { id: "test", name: "Test Player", image: "", role: "user" },
          "",
        ],
        2767927275: [true, "1.1.2"],
        391023589: [true, "1.1.2", ""],
        607772389: true,
        160707185: true,
        3297030196: true,
        3670222014: [true, serverNews, ""],
        1482086086: [true, ""],
        2914043854: [false, "", "fixture image unavailable"],
        1638005996: [true, [], ""],
        3317106280: "https://register.bassaps.com",
        652787832: { overallStatus: "unknown", updatedAt: null, generatedAt: new Date().toISOString(), services: [] },
        4128974838: { latest: null, recommendedAsset: null, assetState: "not_provided", fetchedAt: new Date().toISOString(), stale: false, downloadUrl: "https://download.bassaps.com/" },
      };
      result = results[body.args.methodID] ?? [
        false,
        "Test-only boundary: method not mocked",
      ];
    }
    return route.fulfill({ json: result });
  });
}
