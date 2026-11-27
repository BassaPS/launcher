# BASSAPS Launcher v1.2.1

ゲーム起動時に使用するCyreneHookの同梱内容と更新判定を修正したWindows x64版です。

## 修正内容

- 旧DLLを埋め込んだ実行用EXEを更新し、M7H側と同一のCyreneHook（2,099,200 bytes）を組み込みました。
- 展開済みDLLをファイルサイズだけで判定していた処理を内容比較へ変更。同じサイズでも内容が古いDLLを更新します。
- Launcherの表示バージョンとWindowsのファイル情報を1.2.1へ統一しました。

## ダウンロード・更新手順

1. `BASSAPSLauncher-1.2.1-windows-x64.zip` と `SHA256SUMS.txt` をダウンロードします。
2. PowerShellで以下を実行し、SHA256SUMS.txtのZIP行と一致することを確認します。

   ```powershell
   Get-FileHash -LiteralPath .\BASSAPSLauncher-1.2.1-windows-x64.zip -Algorithm SHA256
   ```

3. 起動中の旧Launcherをトレイも含めて終了します。
4. ZIP全体を新しいフォルダーへ展開し、`BASSAPSLauncher.exe` を起動します。ZIP内から直接実行しないでください。

本体の自己更新は無効のため、手動更新が必要です。APPDATA内の設定・保存アカウントは共有されます。ゲーム・DB・サーバー設定の削除や初期化は不要です。

## 動作環境・確認範囲

Windows x64、Microsoft Edge WebView2 Runtime、起動時の管理者権限が必要です。ゲーム本体・サーバー・アカウント情報は同梱していません。

Go全テスト、フロントエンドLint／ビルド、Windows管理者manifest、EXE内DLLとM7H側DLLの一致を確認しています。実ゲームの起動・ログイン成功は未確認です。

同梱HookはCyreneLauncher v1.2.5由来のゲーム4.7.51向けです。4.7.52を含む他バージョンの互換性は未確認です。既存の未署名・同梱ツール等の制限は継続します。

ソース: https://github.com/BassaPS/launcher
