package cmd

import (
	"testing"

	"github.com/stretchr/testify/suite"
	pg "gitlab.ozon.dev/safariproxd/homework/tests/suites/order"
)

func TestOrderRepositorySuite(t *testing.T) {
	suite.Run(t, new(pg.OrderRepositorySuite))
}
