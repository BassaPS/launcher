# BASSAPS Launcher v1.1.2 — Windows x64

Cyrene LauncherをベースにしたBASSAPS向けランチャーです。

- ライト／ダーク／OS連動テーマ、ゲームライブラリ、狭幅ナビゲーションを追加。
- ゲーム起動、更新、言語・差分ツールなど既存機能を維持。
- 標準接続先をhttp://hsr.bassaps.com:21000、登録・認証・お知らせをregister.bassaps.comへ設定。
- 管理者コンソールを一般ユーザーから制限し、バックエンドでも権限を再確認。
- 非昇格で起動された場合のUAC再起動を追加し、CreateProcessの権限不足を防止。
- 上流Cyreneの実行ファイルで上書きされないよう、専用配布先が未設定の自己更新を停止。
- Cyrene、Magic UI、ObsidianUIのMITライセンスを保持。

## ダウンロード

通常は `BASSAPSLauncher-1.1.2-windows-x64.zip` を選択し、展開後に `BASSAPSLauncher.exe` を起動してください。SHA-256は `SHA256SUMS.txt` で確認できます。

Windows x64 / WebView2 Runtimeが必要です。ゲーム・サーバー・個人設定は含みません。既存環境はバックアップし、サーバー設定を上書きせず更新してください。Windowsの管理者権限と、Webアカウントの管理者ロールは別です。

## 検証・制限

TypeScript、Lint、フロントエンド本番ビルド、UI回帰テスト16件、Goテスト、PEリソースの管理者権限要求を検証しています。UIテストはテスト専用のWails応答を使用しています。

実ゲーム接続・実アカウントでの承認は未検証です。実行ファイルは未署名です。

## 外部ツール

標準ゲーム接続先を `http://hsr.bassaps.com:21000` に統一しました。SRToolsの公開接続ではランチャーのローカル中継と「March7thHoney」を使用します。SRAnalysisはゲーム端末のVeritasへNative接続します。プラグインの自動配布や、M7Hに存在しないSocket.IO APIの追加は行っていません。


SRToolsの502対策: 「ローカル中継を開始」→「March7thHoney」→ 表示されたURL。末尾 /sr-tools は不要です。ユーザー名は数字のゲームUIDを使用します。
