module github.com/launchdarkly/ld-relay/v6

go 1.18

require (
	contrib.go.opencensus.io/exporter/prometheus v0.4.0
	contrib.go.opencensus.io/exporter/stackdriver v0.13.6
	github.com/DataDog/opencensus-go-exporter-datadog v0.0.0-20210527074920-9baf37265e83
	github.com/antihax/optional v1.0.0
	github.com/aws/aws-sdk-go v1.40.45
	github.com/fsnotify/fsnotify v1.4.9
	github.com/go-redis/redis/v8 v8.8.0
	github.com/gomodule/redigo v1.8.2
	github.com/gorilla/mux v1.8.0
	github.com/gregjones/httpcache v0.0.0-20171119193500-2bcd89a1743f
	github.com/hashicorp/consul/api v1.12.0
	github.com/kardianos/minwinsvc v0.0.0-20151122163309-cad6b2b879b0
	github.com/launchdarkly/api-client-go v5.0.3+incompatible
	github.com/launchdarkly/eventsource v1.7.1
	github.com/launchdarkly/go-configtypes v1.1.0
	github.com/launchdarkly/go-jsonstream/v2 v2.0.0
	github.com/launchdarkly/go-sdk-common/v3 v3.0.0-alpha.pub.7
	github.com/launchdarkly/go-sdk-events/v2 v2.0.0-alpha.pub.5
	github.com/launchdarkly/go-server-sdk-consul/v2 v2.0.0-alpha.4
	github.com/launchdarkly/go-server-sdk-dynamodb/v2 v2.0.0-alpha.4
	github.com/launchdarkly/go-server-sdk-evaluation/v2 v2.0.0-alpha.pub.4
	github.com/launchdarkly/go-server-sdk-redis-redigo/v2 v2.0.0-alpha.4
	github.com/launchdarkly/go-server-sdk/v6 v6.0.0-alpha.pub.7
	github.com/launchdarkly/go-test-helpers/v2 v2.3.1
	github.com/pborman/uuid v1.2.0
	github.com/stretchr/testify v1.7.0
	go.opencensus.io v0.23.0
	golang.org/x/sync v0.0.0-20220513210516-0976fa681c29
	gopkg.in/gcfg.v1 v1.2.3
	gopkg.in/launchdarkly/go-server-sdk.v5 v5.9.0
)

require (
	cloud.google.com/go v0.75.0 // indirect
	github.com/DataDog/datadog-go v3.7.2+incompatible // indirect
	github.com/armon/go-metrics v0.3.9 // indirect
	github.com/beorn7/perks v1.0.1 // indirect
	github.com/census-instrumentation/opencensus-proto v0.3.0 // indirect
	github.com/cespare/xxhash/v2 v2.1.2 // indirect
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/dgryski/go-rendezvous v0.0.0-20200823014737-9f7001d12a5f // indirect
	github.com/fatih/color v1.12.0 // indirect
	github.com/go-kit/log v0.2.0 // indirect
	github.com/go-logfmt/logfmt v0.5.1 // indirect
	github.com/golang/groupcache v0.0.0-20210331224755-41bb18bfe9da // indirect
	github.com/golang/protobuf v1.5.2 // indirect
	github.com/google/go-cmp v0.5.6 // indirect
	github.com/google/uuid v1.1.2 // indirect
	github.com/googleapis/gax-go/v2 v2.0.5 // indirect
	github.com/hashicorp/go-cleanhttp v0.5.2 // indirect
	github.com/hashicorp/go-hclog v0.16.2 // indirect
	github.com/hashicorp/go-immutable-radix v1.3.1 // indirect
	github.com/hashicorp/go-rootcerts v1.0.2 // indirect
	github.com/hashicorp/golang-lru v0.5.4 // indirect
	github.com/hashicorp/serf v0.9.6 // indirect
	github.com/jmespath/go-jmespath v0.4.0 // indirect
	github.com/josharian/intern v1.0.0 // indirect
	github.com/kr/text v0.2.0 // indirect
	github.com/launchdarkly/ccache v1.1.0 // indirect
	github.com/launchdarkly/go-ntlm-proxy-auth v1.0.1 // indirect
	github.com/launchdarkly/go-ntlmssp v1.0.1 // indirect
	github.com/launchdarkly/go-semver v1.0.2 // indirect
	github.com/mailru/easyjson v0.7.6 // indirect
	github.com/mattn/go-colorable v0.1.8 // indirect
	github.com/mattn/go-isatty v0.0.14 // indirect
	github.com/matttproud/golang_protobuf_extensions v1.0.1 // indirect
	github.com/miekg/dns v1.1.43 // indirect
	github.com/mitchellh/go-homedir v1.1.0 // indirect
	github.com/mitchellh/mapstructure v1.4.2 // indirect
	github.com/onsi/ginkgo v1.16.2 // indirect
	github.com/onsi/gomega v1.13.0 // indirect
	github.com/patrickmn/go-cache v2.1.0+incompatible // indirect
	github.com/philhofer/fwd v1.0.0 // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	github.com/prometheus/client_golang v1.11.1 // indirect
	github.com/prometheus/client_model v0.2.0 // indirect
	github.com/prometheus/common v0.30.0 // indirect
	github.com/prometheus/procfs v0.7.3 // indirect
	github.com/prometheus/statsd_exporter v0.21.0 // indirect
	github.com/tinylib/msgp v1.1.2 // indirect
	go.opentelemetry.io/otel v0.19.0 // indirect
	go.opentelemetry.io/otel/metric v0.19.0 // indirect
	go.opentelemetry.io/otel/trace v0.19.0 // indirect
	golang.org/x/crypto v0.0.0-20220411220226-7b82a4e95df4 // indirect
	golang.org/x/net v0.0.0-20211209124913-491a49abca63 // indirect; fixes CVE-2021-44716
	golang.org/x/oauth2 v0.0.0-20210514164344-f6687ab2804c // indirect
	golang.org/x/sys v0.0.0-20220412211240-33da011f77ad // indirect; fixes CVE-2022-29526
	golang.org/x/text v0.3.7 // indirect
	google.golang.org/api v0.37.0 // indirect
	google.golang.org/appengine v1.6.7 // indirect
	google.golang.org/genproto v0.0.0-20210126160654-44e461bb6506 // indirect
	google.golang.org/grpc v1.35.0 // indirect
	google.golang.org/protobuf v1.26.0 // indirect
	gopkg.in/DataDog/dd-trace-go.v1 v1.22.0 // indirect
	gopkg.in/launchdarkly/go-jsonstream.v1 v1.0.1 // indirect
	gopkg.in/launchdarkly/go-sdk-common.v2 v2.4.0 // indirect
	gopkg.in/launchdarkly/go-sdk-events.v1 v1.1.1 // indirect
	gopkg.in/launchdarkly/go-server-sdk-evaluation.v1 v1.5.0 // indirect
	gopkg.in/warnings.v0 v0.1.2 // indirect
	gopkg.in/yaml.v2 v2.4.0 // indirect
	gopkg.in/yaml.v3 v3.0.0-20210107192922-496545a6307b // indirect
)
