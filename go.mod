module github.com/ledgerhq/bitcoin-keychain

go 1.15

require (
	github.com/golang/protobuf v1.4.2
	github.com/google/uuid v1.1.2
	github.com/ledgerhq/bitcoin-keychain/pb v0.1.0
	github.com/magefile/mage v1.10.0
	github.com/mattn/go-colorable v0.1.8 // indirect
	github.com/mgutz/ansi v0.0.0-20200706080929-d51e80ef957d // indirect
	github.com/pkg/errors v0.9.1
	github.com/sirupsen/logrus v1.4.1
	github.com/spf13/viper v1.3.2
	github.com/x-cray/logrus-prefixed-formatter v0.5.2
	golang.org/x/net v0.0.0-20201002202402-0a1ea396d57c // indirect
	golang.org/x/sys v0.0.0-20201005172224-997123666555 // indirect
	golang.org/x/text v0.3.3 // indirect
	google.golang.org/genproto v0.0.0-20201006033701-bcad7cf615f2 // indirect
	google.golang.org/grpc v1.32.0
	google.golang.org/protobuf v1.25.0
)

replace github.com/ledgerhq/bitcoin-keychain/pb => ./pb
