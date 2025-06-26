module github.com/mohamedfawas/qubool-kallyanam/services/admin

go 1.23.4

replace github.com/mohamedfawas/qubool-kallyanam/pkg => ../../pkg

replace github.com/mohamedfawas/qubool-kallyanam/api => ../../api

require (
	github.com/mohamedfawas/qubool-kallyanam/api v0.0.0-00010101000000-000000000000
	github.com/mohamedfawas/qubool-kallyanam/pkg v0.0.0-00010101000000-000000000000
	github.com/spf13/viper v1.20.1
	google.golang.org/grpc v1.72.0
)

require (
	github.com/fsnotify/fsnotify v1.8.0 // indirect
	github.com/go-viper/mapstructure/v2 v2.2.1 // indirect
	github.com/pelletier/go-toml/v2 v2.2.3 // indirect
	github.com/sagikazarmark/locafero v0.7.0 // indirect
	github.com/sourcegraph/conc v0.3.0 // indirect
	github.com/spf13/afero v1.12.0 // indirect
	github.com/spf13/cast v1.7.1 // indirect
	github.com/spf13/pflag v1.0.6 // indirect
	github.com/subosito/gotenv v1.6.0 // indirect
	go.uber.org/multierr v1.10.0 // indirect
	go.uber.org/zap v1.27.0 // indirect
	golang.org/x/net v0.35.0 // indirect
	golang.org/x/sys v0.32.0 // indirect
	golang.org/x/text v0.24.0 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20250218202821-56aae31c358a // indirect
	google.golang.org/protobuf v1.36.6 // indirect
	gopkg.in/check.v1 v1.0.0-20201130134442-10cb98267c6c // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)
