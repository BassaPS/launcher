BASSAPS Launcher 1.2.0 - Windows x64

同梱CyreneHookは公式v1.2.5 / ゲーム4.7.51（OSBETAWin4.7.51）向けです。
旧ゲームbuildは動作しない可能性があります。実ゲーム互換性は未検証です。

Release構成の配布候補。正式公開判定はLAUNCHER_RELEASE_FINAL_REPORT.mdを参照。
新旧版は別フォルダーへ展開し、既存Launcherを通常終了してから起動してください。
配置先を変えてもAPPDATAの設定・保存アカウントは共有されます。
SHA256SUMS.txtとBUILD-INFO.txtを確認してください。

使い方
1. ZIP全体を任意のフォルダーへ展開してください。ZIP内から直接実行しないでください。
2. BASSAPSLauncher.exeを起動し、WindowsのUACを許可してください。
3. ゲームファイルを選択して起動してください。

Windows x64とMicrosoft Edge WebView2 Runtimeが必要です。
ゲーム本体・ローカルサーバー・アカウント情報は同梱していません。
標準のゲーム接続先: http://hsr.bassaps.com:21000
登録・認証ポータル: https://register.bassaps.com

既存のゲームフォルダーやサーバー設定を上書きしないでください。
一般ユーザーの管理者コンソール利用はできません。
本体の自己更新・自動更新通知は無効です。新しいZIPから手動更新してください。

ライセンス
Cyrene Launcher / Mar7thDev をベースにしています。
Copyright (c) 2025 Firefly Shelter
MITの全文はLICENSE、UIライブラリのライセンスはlicenses/に同梱しています。

Source: https://github.com/BassaPS/bassaPSLauncher
Upstream: https://github.com/Mar7thDev/CyreneLauncher

この配布物は署名されていません。配布元とSHA256SUMS.txtを確認してください。

## SRTools / SRAnalysis

SRToolsの公開接続はランチャーのローカル中継と「March7thHoney」を使用します。
SRAnalysisはゲーム端末のVeritasプラグインへ「Native」で接続します（http://localhost:1305）。リモートのゲームサーバーURLをSocket.IO接続先には指定しません。プラグインは同梱していません。
各ツール画面の「BASSAPS接続設定」も確認してください。


SRToolsの502対策: 「ローカル中継を開始」→「March7thHoney」→ 表示されたURL。末尾 /sr-tools は不要です。ユーザー名は数字のゲームUIDを使用します。
