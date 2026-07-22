# わが家の漫画

家族用漫画サイトの開発ドキュメントです。

## まず見るもの

- [ユーザーフロー](product/user-flow.md) — 利用者が画面をどう移動するか
- [v1 ワイヤーフレーム](design/wireframes/README.md) — 画面構成を確認する
- [v1 アーキテクチャ](architecture/v1.md) — 技術構成と前提
- [フロントエンド構成](architecture/frontend.md) — 静的画面とJavaScriptの保守方針
- [漫画カタログの管理](operations/catalog.md) — YAML と SQLite の編集・生成手順
- [GCS 漫画画像の運用](operations/gcs-images.md) — バケット内の画像キー構成と投入手順
- [開発ドキュメントの運用](operations/documentation.md) — Docsifyと文書構成の保守手順
- [ADR 0001](adr/0001-local-media-url.md) — ローカル画像配信方針（廃止）
- [ADR 0002](adr/0002-email-allowlist-authorization.md) — 家族限定の認可方針
- [ADR 0003](adr/0003-mobile-vertical-reader.md) — モバイルでの閲覧形式
- [ADR 0004](adr/0004-volume-list.md) — 巻一覧の表示方針
- [ADR 0005](adr/0005-volume-is-reading-unit.md) — 巻とページのデータ構造
- [ADR 0006](adr/0006-multi-page-navigation.md) — URL を持つ画面遷移
- [ADR 0008](adr/0008-static-frontend-organization.md) — 実行用画面と設計資料の分離
- [ADR 0009](adr/0009-docsify-development-documentation.md) — Docsifyによる設計書の公開方針
- [ADR 0010](adr/0010-private-gcs-manga-images.md) — 実際の漫画画像の保管方針
- [ADR 0011](adr/0011-local-gcs-signed-url.md) — ローカルで実画像を安全に読む署名 URL の方針
- [ADR 0012](adr/0012-lazy-reader-image-loading.md) — リーダーの画像通信量を抑える読込方針
- [ADR 0013](adr/0013-gcs-only-image-source.md) — 全環境でGCS画像を使う方針
- [用語集](glossary.md) — 設計で使う用語

## 文書の扱い

- この `docs/` を、プロダクトと設計の正本とする。
- 実装タスクは GitHub Issues で管理し、該当する文書へのリンクを含める。
- 画面や仕様を変える PR では、関連する文書とワイヤーフレームも同時に更新する。
- 秘密情報、家族の個人情報、実際の漫画画像はここへ置かない。

このサイトは GitHub Pages で公開する想定です。公開前に内容が共有可能か必ず確認してください。
