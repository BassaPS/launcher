# CyreneHook 4.7.51更新

2026-11-16。BASSAPS Launcherの版は配布前候補の1.2.0を維持する。

- 公式Release： https://github.com/Mar7thDev/CyreneLauncher/releases/tag/v1.2.5
- 公式source commit：`14b728dd346241b95935e977d6e5032df7fc5314`
- asset：`assets/CyreneHook.dll` / Windows amd64 DLL / 2,099,200 bytes。
- Git blob SHA1：`2231b8b4e839765b913966fd2e292be6acd98d68`。
- SHA256：`f22da8584fa3c208cd9d15ddb5aed4147b6ab20b4d26100534699d98d670254a`。
- 対象：ゲーム4.7.51 / OSBETAWin4.7.51。旧buildは非互換の可能性あり。

公式告知には、簡体字テキストの復元と、保存済み中国語設定が英語へ戻る問題の修正が含まれる。言語serviceのGoコードはこの公式commitと同一、言語画面の差分はstatic classNameのみなので、BASSAPS固有UIと既存のcn選択処理は維持した。DLLの受領・出所・組込みと隔離展開を確認するが、ゲーム実行や言語設定の実ゲーム反映を検証したという意味ではない。

ユーザー提供Eドライブのcyrene-launcher.exeは公式ReleaseのSHA256 `f31c7fe896c7f1a497773705eac31e5cf89ca01626107d0a2789a40c6d186d2c` と一致した。一方、同ディレクトリpatchのDLLは旧2,089,472bytes / SHA256 `16d4f0cc8abfdc4065ddb34a982a5641866b9cca98ba8acc08407531ebfede27` のままだったため、コピー元には使用せず公式commitに固定したDLLを取り込んだ。

Eドライブのserver/config/DBは変更・コピーしていない。ゲーム起動、CA、proxy、hosts、Firewall変更も行わない。BASSAPSの自己更新は無効を維持し、上流EXEで本体を置換しない。
