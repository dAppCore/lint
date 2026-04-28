module dappco.re/go/lint

go 1.26.0

require (
	github.com/stretchr/testify v1.11.1
	gopkg.in/yaml.v3 v3.0.1
)

require (
	github.com/kr/pretty v0.3.1 // indirect
	github.com/rogpeppe/go-internal v1.14.1 // indirect
	gopkg.in/check.v1 v1.0.0-20201130134442-10cb98267c6c // indirect
)

require (
	dappco.re/go v0.9.0
	dappco.re/go/cli v0.0.0
	dappco.re/go/i18n v0.0.0
	dappco.re/go/io v0.0.0
	dappco.re/go/log v0.0.0
	dappco.re/go/process v0.0.0
	dappco.re/go/scm v0.0.0
	github.com/davecgh/go-spew v1.1.2-0.20180830191138-d8f796af33cc // indirect
	github.com/inconshreveable/mousetrap v1.1.0 // indirect
	github.com/pmezard/go-difflib v1.0.1-0.20181226105442-5d4384ee4fb2 // indirect
	github.com/spf13/cobra v1.10.2 // indirect
	github.com/spf13/pflag v1.0.10 // indirect
)

replace dappco.re/go/cli => ./internal/gocli

replace dappco.re/go/i18n => ./internal/goi18n

replace dappco.re/go/io => ./internal/goio

replace dappco.re/go/log => ./internal/golog

replace dappco.re/go/process => ./internal/goprocess

replace dappco.re/go/scm => ./internal/goscm
