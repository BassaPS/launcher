import licenseText from "../../../../LICENSE?raw";
import { ExternalLink } from "lucide-react";
import { AppService } from "@bindings/cyrene-launcher/internal/app-service";
import { toast } from "react-toastify";
import PlatformRelease from "@/components/platformRelease";

export default function AboutPage() {
  const openSource = async () => {
    try {
      await AppService.OpenURL("https://github.com/Mar7thDev/CyreneLauncher");
    } catch {
      toast.error("リポジトリを開けませんでした。");
    }
  };
  return (
    <div className="bassaps-page bassaps-about">
      <header className="bassaps-page-heading">
        <div>
          <h1>このアプリについて</h1>
          <p>クレジットとオープンソースライセンス</p>
        </div>
      </header>
      <section className="bassaps-panel">
        <div className="bassaps-panel-heading">
          <h2>BASSAPS Launcher</h2>
        </div>
        <p>
          Cyrene
          Launcherをベースに、BASSAPS向けのUIと接続設定を追加したランチャーです。
        </p>
        <dl className="bassaps-about-meta">
          <div>
            <dt>ベースプロジェクト</dt>
            <dd>Cyrene Launcher / Mar7thDev</dd>
          </div>
          <div>
            <dt>著作権表記</dt>
            <dd>Copyright (c) 2025 Firefly Shelter</dd>
          </div>
          <div>
            <dt>ライセンス</dt>
            <dd>MIT License</dd>
          </div>
        </dl>
        <button className="btn btn-sm btn-outline" onClick={() => AppService.OpenURL("https://github.com/BassaPS/launcher")}>
          <ExternalLink size={14} />
          BassaPS LauncherをGitHubで見る
        </button>
        <button className="btn btn-sm btn-outline" onClick={openSource}>
          <ExternalLink size={14} />
          元プロジェクトをGitHubで見る
        </button>
      </section>
      <PlatformRelease />
      <section className="bassaps-panel">
        <div className="bassaps-panel-heading">
          <h2>MIT License</h2>
          <span className="bassaps-muted">原文</span>
        </div>
        <p>
          元プロジェクトの著作権表記とライセンス全文を以下に掲載しています。
        </p>
        <pre className="bassaps-license" tabIndex={0}>
          {licenseText}
        </pre>
      </section>
    </div>
  );
}
