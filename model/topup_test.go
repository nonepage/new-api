package model

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestTopUpGetEffectiveCurrency_InfersLegacyStripeCurrency(t *testing.T) {
	topup := &TopUp{
		PaymentMethod: "stripe",
	}

	assert.Equal(t, "USD", topup.GetEffectiveCurrency())
}

func TestTopUpGetEffectiveCurrency_DefaultsToCNYForLegacyDomesticOrders(t *testing.T) {
	topup := &TopUp{
		PaymentMethod: "alipay",
	}

	assert.Equal(t, "CNY", topup.GetEffectiveCurrency())
}
