module github.com/ledgerhq/bitcoin-keychain

go 1.16

require (
	github.com/cosmtrek/air v1.27.3 // indirect
	github.com/creack/pty v1.1.17 // indirect
	github.com/fatih/color v1.13.0 // indirect
	github.com/fsnotify/fsnotify v1.5.1 // indirect
	github.com/go-redis/redis/v8 v8.2.3
	github.com/golang/protobuf v1.5.2
	github.com/google/uuid v1.1.2
	github.com/grpc-ecosystem/grpc-gateway/v2 v2.6.0
	github.com/ledgerhq/bitcoin-keychain/pb v0.1.0
	github.com/magefile/mage v1.10.0
	github.com/mattn/go-colorable v0.1.11 // indirect
	github.com/mgutz/ansi v0.0.0-20200706080929-d51e80ef957d // indirect
	github.com/pelletier/go-toml v1.9.4 // indirect
	github.com/pkg/errors v0.9.1
	github.com/sirupsen/logrus v1.4.1
	github.com/spf13/viper v1.3.2
	github.com/x-cray/logrus-prefixed-formatter v0.5.2
	golang.org/x/sys v0.0.0-20211116061358-0a5406a5449c // indirect
	google.golang.org/genproto v0.0.0-20211001223012-bfb93cce50d9
	google.golang.org/grpc v1.41.0
	google.golang.org/protobuf v1.27.1
)

replace github.com/ledgerhq/bitcoin-keychain/pb => ./pb
