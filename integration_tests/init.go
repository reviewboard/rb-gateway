package integration_tests

import (
	"os"
)

func init() {
	rbGatewayPath := os.Getenv("RBGATEWAY_PATH")
	if rbGatewayPath == "" {
		panic("RBGATEWAY_PATH not set")
	}

	os.Args[0] = rbGatewayPath
}
