module code.dumpstack.io/tools/donate

replace code.dumpstack.io/tools/donate/database => ./database

go 1.12

require (
	code.dumpstack.io/lib/cryptocurrency v1.5.1
	code.dumpstack.io/tools/donate/database v0.0.0-20200119115012-a4556df0c12e
	github.com/google/go-github/v29 v29.0.2
	golang.org/x/oauth2 v0.0.0-20200107190931-bf48bf16ab8d
	gopkg.in/alecthomas/kingpin.v2 v2.2.6
)
