// 本地开发：仓库根的 go.work（不入库，克隆后按 README「开发」章节生成）负责把根
// module 解析到本地目录，本文件无需 replace。注意 go.work 需要 use + 带版本的
// replace 组合（仅 use 对未发布的 v0.0.0 占位依赖在 module graph 加载时覆盖不彻底，
// 见 golang/go#50750 一族问题）。
//
// 发布前唯一要做的：把下方 require 的 sud-server-sdk 改为根 module 已发布的 tag 版本
// （根 module 打 tag 如 v1.0.0 后，将 v0.0.0 改为 v1.0.0 即可，外部消费者才能解析）。
module github.com/linuxea/sud-server-sdk/adapter/gin

go 1.19

require (
	github.com/gin-gonic/gin v1.7.2
	github.com/linuxea/sud-server-sdk v0.0.0
)

require (
	github.com/gin-contrib/sse v0.1.0 // indirect
	github.com/go-playground/locales v0.13.0 // indirect
	github.com/go-playground/universal-translator v0.17.0 // indirect
	github.com/go-playground/validator/v10 v10.4.1 // indirect
	github.com/golang/protobuf v1.3.3 // indirect
	github.com/json-iterator/go v1.1.9 // indirect
	github.com/leodido/go-urn v1.2.0 // indirect
	github.com/mattn/go-isatty v0.0.12 // indirect
	github.com/modern-go/concurrent v0.0.0-20180228061459-e0a39a4cb421 // indirect
	github.com/modern-go/reflect2 v0.0.0-20180701023420-4b7aa43c6742 // indirect
	github.com/ugorji/go/codec v1.1.7 // indirect
	golang.org/x/crypto v0.0.0-20200622213623-75b288015ac9 // indirect
	golang.org/x/sys v0.0.0-20200116001909-b77594299b42 // indirect
	gopkg.in/yaml.v2 v2.2.8 // indirect
)

