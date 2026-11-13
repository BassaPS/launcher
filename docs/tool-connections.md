# 外部ツールの接続設定

## ゲーム接続先

標準値は `http://hsr.bassaps.com:21000`。Goのゲームプロキシ、コンソール、ハンドブックとフロントエンドの既定値を統一しています。ローカル接続先 `http://127.0.0.1:21000` とユーザー指定のカスタム設定は維持します。

登録・ログイン・Launcher用トークン・ニュースは引き続き `https://register.bassaps.com` を使用します。

## SRTools（公開M7H）

1. ランチャーのSRTools画面で「ローカル中継を開始」を押します。
2. サイトのPS接続タイプを「March7thHoney」にします。
3. 表示されたURL（通常 http://127.0.0.1:24680）を入力します。末尾 /sr-tools は不要です。
4. ユーザー名は数字のゲームUID、パスワードはゲームアカウントのものを入力します。
5. ゲームへログインして接続します。ランチャーを開いたままにし、ブラウザーのローカルネットワーク許可が必要な場合は許可します。

サイト側の「その他 → サーバー」は調査時点でCloudflareの502 HTMLを返しました。M7Hへの空JSONの直接POSTでは400と整数UIDが必要という応答を確認しました。BassaPSなどのアカウント名は使用できません。

中継は127.0.0.1のみで待受し、SRToolsのOrigin、Host、JSON POSTを検証します。転送先は設定したM7Hの /sr-tools のみです。リダイレクト禁止、同時2要求、要求4MiB/応答8MiB制限があります。資格情報は保存・ログ出力しません。HTTP接続先では通信は暗号化されません。終了時に待受を閉じます。待受開始は認証成功を意味しません。

ローカルM7Hでは中継不要です。「March7thHoney」と http://127.0.0.1:21000 で直接接続します。

## SRAnalysis（リモートで遊ぶ場合もゲーム端末に接続）

https://sranalysis.punklorde.org/ は、Veritasがゲーム端末内で配信する戦闘イベントへ接続する方式です。

1. ゲームバージョンに合ったVeritasを [公式Wiki](https://github.com/hessiser/veritas/wiki) の手順で配置します。
2. ゲームを再起動してプラグインを読み込みます。
3. SRAnalysisで接続タイプ `Native` を選び `Connect` を押します。現在のNative接続先は `http://localhost:1305` です。

ゲームサーバーがリモートでも、プラグインとブラウザーは同じゲーム端末で動かします。公開M7HのHTTPポートをSocket.IOの代わりには指定できません。ランチャーは `astrolabe.dll`、`xluau.dll`、`plugins/veritas.dll` の存在を確認しますが、DLLの実際の読み込み・バージョン互換性・接続成功までは保証しません。対応バージョンを確認せずDLLを自動ダウンロードする実装は追加していません。

## 調査・検証範囲（2026-11-12）

資格情報なしの読み取り検査で、公開サーバーの `/sr-tools` はGETに405、Allow: POSTを返しました。OPTIONSではPOSTとContent-TypeのCORS許可を確認しました。一方、`/socket.io/?EIO=4&transport=polling` と `/check-tcp` は404でした。

SRToolsの配信中スクリプトと [公式ソース](https://git.kain.id.vn/Kain344/Firefly_Srtools_Live) の `src/helper/connectEndpoint.ts`、`src/helper/connect.ts`、`src/lib/api/api.ts`、`src/app/api/proxy/route.ts` を確認しました。SRAnalysisの配信中スクリプトでもNativeはlocalhost:1305、PSはlocalhost:21000へ接続することを確認しました。

UIテストでは既定ゲームURLがWailsの起動引数に渡ること、公開SRTools URLのAPIパス、ローカルモードの維持、Native分析の案内を確認しています。実アカウントでの接続やビルド同期、実ゲームでの戦闘イベント受信は未検証です。サーバーの公開設定・ファイアウォール・認証条件・ブラウザー権限は変更していません。
