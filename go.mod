module git.rootprojects.org/root/telebit

go 1.14

replace github.com/caddyserver/certmagic => github.com/coolaj86/certmagic v0.12.1-pre.2

require (
	git.rootprojects.org/root/go-gitver/v2 v2.0.2
	github.com/coolaj86/certmagic v0.12.1-pre.2
	github.com/denisbrodbeck/machineid v1.0.1
	github.com/dgrijalva/jwt-go v3.2.0+incompatible
	github.com/go-acme/lego/v3 v3.7.0
	github.com/go-chi/chi v4.1.1+incompatible
	github.com/gorilla/websocket v1.4.2
	github.com/jmoiron/sqlx v1.2.1-0.20190826204134-d7d95172beb5
	github.com/joho/godotenv v1.3.0
	github.com/kr/text v0.2.0 // indirect
	github.com/lib/pq v1.6.0
	github.com/mholt/acmez v0.1.1
	github.com/niemeyer/pretty v0.0.0-20200227124842-a10e7caefd8e // indirect
	github.com/shurcooL/httpfs v0.0.0-20190707220628-8d4bc4ba7749 // indirect
	github.com/shurcooL/vfsgen v0.0.0-20181202132449-6a9ea43bcacd
	github.com/stretchr/testify v1.6.1
	golang.org/x/sys v0.0.0-20200625212154-ddb9806d33ae
	golang.org/x/text v0.3.3 // indirect
	golang.org/x/tools v0.0.0-20200626171337-aa94e735be7f // indirect
	gopkg.in/check.v1 v1.0.0-20200227125254-8fa46927fb4f // indirect
	gopkg.in/yaml.v3 v3.0.0-20200615113413-eeeca48fe776 // indirect
)
