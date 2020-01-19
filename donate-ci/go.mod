module code.dumpstack.io/tools/donate/donate-ci

replace code.dumpstack.io/tools/donate/database => ../database

go 1.12

require (
	code.dumpstack.io/lib/cryptocurrency v1.4.0
	code.dumpstack.io/tools/donate/database v0.0.0-00010101000000-000000000000
	github.com/alecthomas/template v0.0.0-20190718012654-fb15b899a751 // indirect
	github.com/alecthomas/units v0.0.0-20190924025748-f65c72e2690d // indirect
	github.com/google/go-github/v29 v29.0.2
	golang.org/x/oauth2 v0.0.0-20200107190931-bf48bf16ab8d
	gopkg.in/alecthomas/kingpin.v2 v2.2.6
)
