# BASSAPS Launcher v1.2.0

Windows x64向けRelease構成の配布候補です。公開済みv1.1.2（tag: First）から、BASSAPS Webの情報連携と外部入力の安全性を改善しました。

**正式公開の判定は保留です。** 同梱7-Zip 24.09、Fireflyの展開後読取上限、実native RuntimeGuardの未確認、未署名など、最終報告書の残課題を解決・判断してから公開してください。パッケージ完成を正式配布承認と扱わないでください。

## 変更点

- HomeにStatus情報、Aboutにインストール版・公開版と推奨配布ZIP情報を追加。HTTP到達性とゲーム接続、未監視・古い情報を区別します。
- Portal、Account、Download、Documentation、Status、お知らせの正式サービス導線を統合。
- 運営・公式NewsのHTTP/JSON/日時/URLを検証し、画像を記事IDに対応したGo取得へ限定。形式・byte数・画素数・同時数・cache・表示近傍取得を制限。
- 外部リンクのGo検証とWails Browser runtimeの迂回防止を追加。
- Go 1.26.11、更新済みGo/frontend依存、静的翻訳を使用。今回の版更新では依存を追加更新していません。
- 製品版を1.2.0に統一し、Windows build metadataとの不一致をビルド時に拒否。EXE名と管理者manifestは維持。

Web APIは正式URLを参照します。今回Web側のコード・認証・本番設定は変更していません。情報表示は実ゲーム接続・実認証の成功を保証しません。

## 動作環境

Windows x64 / Microsoft Edge WebView2 Runtime / 起動時のWindows管理者権限が必要です。WebView2 Runtime、ゲーム、サーバーデータ、アカウント情報は同梱していません。Windows管理者とWebアカウントの管理者ロールは別です。実際の確認環境・範囲はLAUNCHER_RELEASE_FINAL_REPORT.mdを参照してください。

## インストール・手動更新

1. 公式配布元の `BASSAPSLauncher-1.2.0-windows-x64.zip` と `SHA256SUMS.txt` を取得します。
2. SHA256を確認し、ZIP全体を既存版とは別の新しいフォルダーへ展開します。
3. 既存Launcherをトレイも含めて通常終了し、`BASSAPSLauncher.exe` を起動します。ZIP内から直接実行しないでください。
4. UACの内容と配布元を確認し、ご本人の判断で承認します。DefenderやSmartScreenは無効化しないでください。

配置先を変えてもAPPDATAの設定・保存アカウントは共有されます。account.datやprofileを削除・初期化せず、必要なバックアップはご本人の非公開保存先へ行ってください。

v1.1.2/Firstは残します。v1.1.2は本体の自己更新・自動更新通知が無効で、Release公開だけでは自動更新されません。新ZIPから手動更新してください。v1.2.0のAboutはDownload APIの公開版を表示しますが、本体自己更新は引き続き無効です。

## SHA256確認

PowerShellでZIPのあるフォルダーを開いて実行し、SHA256SUMS.txtのZIP行と比較してください。

```powershell
Get-FileHash -LiteralPath .\BASSAPSLauncher-1.2.0-windows-x64.zip -Algorithm SHA256
```

この候補EXEは未署名です。checksumはコード署名の代替ではありません。

## 既知の制限

- 同梱7-Zip 24.09は後続版のセキュリティ修正を含まず、正式公開前の対応事項です。
- Fireflyの圧縮解除後の一般的な総読取上限は未対応です。
- dev依存のmoderate脆弱性、RuntimeGuard/cancel等のnative確認、同梱ツールの来歴・ライセンス対応は最終報告書に従ってください。
- 今回のパッケージ検証で実ゲーム接続・Relay・CA導入・全ゲーム互換性は確認していません。
- API障害やキャッシュ経過時は取得失敗・古い情報として表示します。未監視UDPを正常と判定しません。

ソース: https://github.com/BassaPS/bassaPSLauncher

正式Web: https://bassaps.com/ / https://register.bassaps.com/ / https://download.bassaps.com/ / https://docs.bassaps.com/ / https://status.bassaps.com/
