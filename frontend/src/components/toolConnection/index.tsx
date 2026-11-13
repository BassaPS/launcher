import { useEffect, useState } from "react";
import { Copy, ExternalLink, FolderOpen } from "lucide-react";
import { toast } from "react-toastify";
import useSettingStore, { resolveServerBaseUrl } from "@/stores/settingStore";
import { FSService } from "@bindings/cyrene-launcher/internal/fs-service";
import { AppService } from "@bindings/cyrene-launcher/internal/app-service";

import { March7thHoneyService } from "@bindings/cyrene-launcher/internal/march7thhoney-service";

type Tool = "srtools" | "analysis";

export default function ToolConnection({ tool }: { tool: Tool }) {
  const { serverTarget, patchTargetUrl, gameDir, gamePath } = useSettingStore();
  const base = resolveServerBaseUrl(serverTarget, patchTargetUrl);
  const local =
    serverTarget === "local_test" ||
    serverTarget === "local_prod" ||
    /^https?:\/\/(localhost|127\.0\.0\.1)(:|\/|$)/i.test(base);
  const endpoint =
    base.replace(/\/+$/, "").replace(/\/sr-tools$/i, "") + "/sr-tools";
  const gameFolder =
    gameDir || gamePath.replace(/\\/g, "/").replace(/\/[^/]*$/, "");
  const [relay, setRelay] = useState("");
  const [starting, setStarting] = useState(false);
  const [relayError, setRelayError] = useState("");
  useEffect(() => { setRelay(""); setRelayError(""); }, [base]);
  const startRelay = async () => {
    setStarting(true); setRelayError("");
    try {
      const [ok, address, message] = await March7thHoneyService.StartSRToolsRelay(base);
      if (!ok) throw new Error(message || "中継を開始できませんでした");
      setRelay(address);
    } catch (error) {
      setRelayError(error instanceof Error ? error.message : "中継を開始できませんでした");
    } finally { setStarting(false); }
  };
  const connectionURL = local ? base : relay;
  const [plugin, setPlugin] = useState<string | null>(null);
  const [checking, setChecking] = useState(false);
  const [checkFailed, setCheckFailed] = useState(false);
  useEffect(() => {
    if (tool !== "analysis" || !gameFolder) return;
    let cancelled = false;
    setChecking(true);
    setPlugin(null);
    setCheckFailed(false);
    const candidates = ["astrolabe.dll", "xluau.dll", "plugins/veritas.dll"];
    void Promise.all(
      candidates.map(async (name) =>
        (await FSService.FileExists(`${gameFolder}/${name}`)) ? name : null,
      ),
    )
      .then((found) => {
        if (!cancelled) setPlugin(found.find(Boolean) ?? null);
      })
      .catch(() => {
        if (!cancelled) setCheckFailed(true);
      })
      .finally(() => {
        if (!cancelled) setChecking(false);
      });
    return () => {
      cancelled = true;
    };
  }, [tool, gameFolder]);
  const copy = async (text: string) => {
    try {
      await navigator.clipboard.writeText(text);
      toast.success("接続先をコピーしました");
    } catch {
      toast.error(
        "コピーできませんでした。表示された接続先を手動で入力してください。",
      );
    }
  };
  const open = async (url: string) => {
    try {
      const [ok, message] = await AppService.OpenURL(url);
      if (!ok) throw new Error(message);
    } catch {
      toast.error("ブラウザーを開けませんでした");
    }
  };
  return (
    <section
      className="bassaps-panel space-y-4"
      aria-label={
        tool === "srtools"
          ? "BASSAPS SRTools接続設定"
          : "BASSAPS SRAnalysis接続設定"
      }
    >
      <div className="bassaps-panel-heading">
        <h2>
          {tool === "srtools"
            ? "BASSAPSへの接続設定"
            : "リモートプレイ中のSRAnalysis"}
        </h2>
        <span className="bassaps-badge">
          {tool === "srtools" ? "M7H / SRTools" : "ゲーム端末 / Native"}
        </span>
      </div>
      {tool === "srtools" ? (
        <>
          <p>PS接続の接続タイプは「March7thHoney」を選択します。</p>
          {!local && <>
            <p>サイト側の中継が502になる場合はランチャーの中継を使用します。転送先: <code>{endpoint}</code></p>
            <button className="btn btn-sm btn-outline" disabled={starting} onClick={startRelay}>
              {starting ? "開始中…" : "ローカル中継を開始"}
            </button>
            {relayError && <p role="alert">{relayError}</p>}
          </>}
          {connectionURL && <>
            <div className="flex flex-wrap items-center gap-2">
              <code className="min-w-0 break-all text-sm" data-testid="srtools-endpoint">{connectionURL}</code>
              <button className="btn btn-sm btn-outline" aria-label="SRTools接続先をコピー" onClick={() => copy(connectionURL)}><Copy size={14} />コピー</button>
            </div>
            <p role="status">{local ? "このPCのM7Hへ直接接続します。" : "中継待受中（アカウント未検証）。ランチャーを開いたまま、上記URLをサーバーURLに入力してください。"}</p>
          </>}
          <p>ユーザー名には数字のゲームUIDを入力してください（BassaPSなどのアカウント名ではありません）。パスワードはゲームアカウントのものを使用し、ゲームにログインした状態で接続します。</p>
          <p>URLの末尾に /sr-tools は不要です。ブラウザーからローカルネットワークへのアクセス許可を求められた場合は許可してください。</p>
          <button
            className="btn btn-sm btn-primary"
            onClick={() => open("https://srtools.punklorde.org/")}
          >
            <ExternalLink size={14} />
            SRToolsを開く
          </button>
        </>
      ) : (
        <>
          <p>
            SRAnalysisはサーバーのHTTP
            APIではなく、ゲーム内のVeritasプラグインが配信する戦闘情報へ接続します。リモートサーバーで遊ぶ場合も、接続タイプは「Native」です。
          </p>
          <dl className="space-y-2 text-sm">
            <div>
              <dt className="text-base-content/70">接続タイプ</dt>
              <dd>Native</dd>
            </div>
            <div>
              <dt className="text-base-content/70">ゲーム端末の接続先</dt>
              <dd>
                <code>http://localhost:1305</code>
              </dd>
            </div>
          </dl>
          <p role="status">
            {!gameFolder
              ? "ゲームファイルを選択すると、プラグインの配置を確認できます。"
              : checking
                ? "プラグインの配置を確認中…"
                : checkFailed
                  ? "ファイルの配置を確認できませんでした。ゲームフォルダーを確認してください。"
                  : plugin
                    ? `${plugin} を確認しました。配置の確認のみで、読み込み完了や接続を保証するものではありません。`
                    : "プラグインが見つかりません。ゲームのバージョンに対応するVeritasを、公式の配置手順に従って導入してください。"}
          </p>
          <p>
            公開サーバーの /socket.io と /check-tcp は現在404のため、「PS」で
            hsr.bassaps.com:21000
            を指定しても接続できません。プラグインの配置後にゲームを再起動し、SRAnalysisで「Native
            → Connect」を選びます。
          </p>
          <div className="flex flex-wrap gap-2">
            <button
              className="btn btn-sm btn-primary"
              onClick={() => open("https://sranalysis.punklorde.org/")}
            >
              <ExternalLink size={14} />
              SRAnalysisを開く
            </button>
            <button
              className="btn btn-sm btn-outline"
              onClick={() => open("https://github.com/hessiser/veritas/wiki")}
            >
              <ExternalLink size={14} />
              Veritasの公式手順
            </button>
            <button
              className="btn btn-sm btn-outline"
              disabled={!gameFolder}
              onClick={() => {
                void FSService.OpenFolder(gameFolder).catch(() =>
                  toast.error("ゲームフォルダーを開けませんでした"),
                );
              }}
            >
              <FolderOpen size={14} />
              ゲームフォルダー
            </button>
          </div>
        </>
      )}
    </section>
  );
}
