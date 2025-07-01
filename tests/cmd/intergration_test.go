package cmd

import (
	"testing"

	"github.com/stretchr/testify/suite"
	"gitlab.ozon.dev/safariproxd/homework/tests/suites/e2e"
	pg "gitlab.ozon.dev/safariproxd/homework/tests/suites/order"
)

func TestOrderRepositorySuite(t *testing.T) {
	suite.Run(t, new(pg.OrderRepositorySuite))
}

func TestOrdersE2ESuite(t *testing.T) {
	suite.Run(t, new(e2e.OrdersE2ESuite))
}
