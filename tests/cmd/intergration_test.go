package cmd

import (
	"testing"

	"github.com/stretchr/testify/suite"
	"gitlab.ozon.dev/safariproxd/homework/tests/suites/e2e"
	"gitlab.ozon.dev/safariproxd/homework/tests/suites/kafka"
	pg "gitlab.ozon.dev/safariproxd/homework/tests/suites/order"
)

func TestOrderRepositorySuite(t *testing.T) {
	suite.Run(t, new(pg.OrderRepositorySuite))
}

func TestKafkaIntegration(t *testing.T) {
	suite.Run(t, new(kafka.KafkaIntegrationSuite))
}

func TestOrdersE2ESuite(t *testing.T) {
	suite.Run(t, new(e2e.OrdersE2ESuite))
}
