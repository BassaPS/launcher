# BASSAPS Launcher v1.3.0

BASSAPS共通デザインシステムを採用した、Windows x64版LauncherのUI更新です。

## 変更内容

- Portal・Register・Download・Docs・Statusと共通のY2K Popカラー、フォント、ボタン、入力フォーム、状態表示を適用しました。
- ライブラリ画面、サイドバー、テーマ選択、設定画面を統一し、ライト／ダークテーマとウィンドウ幅への対応を整えました。
- フォントをローカルに同梱し、外部フォントCDNなしでも表示できます。フォントと共通アセットのライセンス文書も同梱しています。
- キーボード操作、フォーカス、文字のコントラスト、モーションを調整しました。

既存の認証、接続先、ゲーム起動、ツール中継、更新処理、保存設定は維持しています。自己更新は引き続き無効です。

## ダウンロード・更新手順

1. `BASSAPSLauncher-1.3.0-windows-x64.zip` と `SHA256SUMS.txt` をダウンロードします。
2. `Get-FileHash -LiteralPath .\BASSAPSLauncher-1.3.0-windows-x64.zip -Algorithm SHA256` の結果を、SHA256SUMS.txtのZIP行と照合してください。
3. 旧Launcherをトレイも含めて終了し、ZIP全体を新しいフォルダーへ展開します。
4. 展開した `BASSAPSLauncher.exe` を起動してください。ZIP内から直接実行しないでください。

APPDATA内の設定・保存アカウントは共有されます。既存のゲームフォルダー、サーバー設定、DBを削除する必要はありません。

## 動作環境・確認範囲

Windows x64、Microsoft Edge WebView2 Runtime、起動時の管理者権限が必要です。ゲーム本体、サーバー、アカウント情報は同梱していません。バイナリは未署名です。

フロントエンドのLint・型チェック・ビルド、Wailsモックを用いた85件の操作テスト、Goテスト、Windows版のビルド、EXEのバージョンと管理者manifestを検証しています。実アカウントでのログイン、実ゲーム起動、管理者権限下のWebView2動作、ゲーム互換性は未検証です。

本Releaseは公開前のドラフトです。公開済みv1.2.1のファイルは変更していません。

サポート: support@bassaps.com

ソース: https://github.com/BassaPS/launcher/tree/design/y2k-renewal
