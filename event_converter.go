package s2seventlib

import (
	"context"
	"fmt"
	"os"
	"strconv"

	"github.com/awa/go-iap/appstore"
	"github.com/awa/go-iap/playstore"
	"github.com/pkg/errors"
)

type EventConverter struct {
	userIDProvider    UserIDProvider
	playStoreVerifier PlayStoreVerifier
}

func NewEventConverter(userIDProvider UserIDProvider, playStoreVerifier PlayStoreVerifier) EventConverter {
	return EventConverter{
		userIDProvider:    userIDProvider,
		playStoreVerifier: playStoreVerifier,
	}
}

func (g EventConverter) ConvertPlayStoreEvent(ctx context.Context, noti playstore.DeveloperNotification) (Event, error) {
	token := noti.SubscriptionNotification.PurchaseToken
	notiType := noti.SubscriptionNotification.NotificationType
	userID, err := g.userIDProvider.UserID(token)

	if err != nil {
		return Event{}, err
	}

	fmt.Printf("token: %s, notiType: %d, userID: %s\n", token, notiType, userID)

	var eventType EventType
	switch notiType {
	case playstore.SubscriptionNotificationTypePurchased:
		eventType = EventPurchase
	case playstore.SubscriptionNotificationTypeRenewed:
		eventType = EventRenew
	case playstore.SubscriptionNotificationTypeRecovered:
		eventType = EventRecover
	case playstore.SubscriptionNotificationTypeRestarted:
		eventType = EventRestart
	case playstore.SubscriptionNotificationTypeRevoked:
		eventType = EventCancel
	case playstore.SubscriptionNotificationTypeCanceled:
		eventType = EventTurnOffAutoRenew
	default:
		return Event{}, errors.Errorf("not purchase event: %d", notiType)
	}

	purchase, err := g.playStoreVerifier.Verify(ctx, noti.PackageName, noti.SubscriptionNotification.SubscriptionID, token)

	if err != nil {
		return Event{}, err
	}

	timestamp, err := strconv.Atoi(noti.EventTimeMillis)
	if err != nil {
		return Event{}, err
	}

	props := EventProperties{
		PaymentState: PaymentState(purchase.PaymentState),
		AppID:        os.Getenv("BRAZE_APP_ID"),
		ProductID:    noti.SubscriptionNotification.SubscriptionID,
		Currency:     purchase.PriceCurrencyCode,
		Price:        float64(purchase.PriceAmountMicros) / 1_000_000,
		Quantity:     1,
	}

	if eventType == EventCancel || eventType == EventTurnOffAutoRenew {
		props.CancellationReason = strconv.FormatInt(purchase.CancelReason, 10)
	}

	return Event{
		EventType:       eventType,
		UserID:          userID,
		Platform:        "android",
		EventTimeMillis: timestamp,
		Env:             "prod",
		Properties:      props,
	}, nil
}

var productIDtoPriceMap = map[string]float64{
	"droom.sleepIfUCanFree.premium.monthly.0":        4.99,
	"droom.sleepIfUCanFree.premium.yearly.0":         54.99,
	"droom.sleepIfUCanFree.premium.monthly.1":        4.99,
	"droom.sleepIfUCanFree.premium.yearly.1":         54.99,
	"droom.sleepIfUCanFree.premium.yearlyPromo.0":    46.99,
	"droom.sleepIfUCanFree.premium.yearlyPromo.1":    46.99,
	"com.productname.premium.monthly":                10.49,
	"droom.sleepIfUCanFree.premium.monthly.4":        4.99,
	"droom.sleepIfUCanFree.premium.monthlyPromo.4":   3.49,
	"droom.sleepIfUCanFree.premium.yearly.4":         41.99,
	"droom.sleepIfUCanFree.premium.monthlyDecoy01.4": 6.99,
	"droom.sleepIfUCanFree.premium.monthlyDecoy02.4": 7.49,
	"droom.sleepIfUCanFree.premium.yearly01.4":       59.99,
	"droom.sleepIfUCanFree.premium.monthlyDecoy03.4": 9.99,
}

func priceForAppStoreProduct(productID string) (float64, error) {
	price, ok := productIDtoPriceMap[productID]
	if !ok {
		return 0, errors.Errorf("cannot find price info for the productID: %s", productID)
	}
	return price, nil
}

func (g EventConverter) ConvertAppStoreEvent(ctx context.Context, noti appstore.SubscriptionNotification) (Event, error) {

	notiType := noti.NotificationType
	var env string
	if noti.Environment == "PROD" {
		env = "prod"
	} else {
		env = "dev"
	}

	if notiType == appstore.NotificationTypeCancel {
		webOrderLineItemID := noti.WebOrderLineItemID

		for _, receiptInfo := range noti.UnifiedReceipt.LatestReceiptInfo {
			if receiptInfo.WebOrderLineItemID == webOrderLineItemID {
				token := receiptInfo.OriginalTransactionID

				userID, err := g.userIDProvider.UserID(token)
				if err != nil {
					return Event{}, err
				}

				timestamp, err := strconv.Atoi(receiptInfo.CancellationDateMS)
				if err != nil {
					return Event{}, err
				}

				// Get price based on Product ID
				price, err := priceForAppStoreProduct(receiptInfo.ProductID)
				if err != nil {
					return Event{}, err
				}

				return Event{
					EventType:       EventCancel,
					UserID:          userID,
					Platform:        "ios",
					EventTimeMillis: timestamp,
					Env:             env,
					Properties: EventProperties{
						Price:              price,
						Currency:           "USD",
						Quantity:           1,
						ProductID:          receiptInfo.ProductID,
						CancellationReason: receiptInfo.CancellationReason,
					},
				}, err
			}
		}

		return Event{}, errors.Errorf("failed to find receiptInfo for webOrderLineItemID: %s", webOrderLineItemID)
	} else if notiType == appstore.NotificationTypeDidChangeRenewalStatus {
		inApp := noti.UnifiedReceipt.LatestReceiptInfo[0]
		token := inApp.OriginalTransactionID
		userID, err := g.userIDProvider.UserID(token)
		if err != nil {
			return Event{}, err
		}

		var eventType EventType
		if noti.AutoRenewStatus == "true" {
			eventType = EventTurnOnAutoRenew
		} else {
			eventType = EventTurnOffAutoRenew
		}

		timestamp, err := strconv.Atoi(noti.AutoRenewStatusChangeDateMS)
		if err != nil {
			return Event{}, err
		}

		return Event{
			EventType:       eventType,
			UserID:          userID,
			Platform:        "ios",
			EventTimeMillis: timestamp,
			Env:             env,
			Properties: EventProperties{
				ProductID: noti.AutoRenewProductID,
			},
		}, err
	} else if notiType == appstore.NotificationTypeInitialBuy || notiType == appstore.NotificationTypeDidRenew || notiType == appstore.NotificationTypeDidRecover || notiType == appstore.NotificationTypeInteractiveRenewal {
		inApp := noti.UnifiedReceipt.LatestReceiptInfo[0]
		token := inApp.OriginalTransactionID
		userID, err := g.userIDProvider.UserID(token)
		if err != nil {
			return Event{}, err
		}

		// Determine event type
		var eventType EventType
		switch notiType {
		case appstore.NotificationTypeInitialBuy:
			eventType = EventPurchase
		case appstore.NotificationTypeDidRecover:
			eventType = EventRecover
		case appstore.NotificationTypeInteractiveRenewal:
			eventType = EventRestart
		default:
			eventType = EventRenew
		}

		// Get payment state
		var paymentState PaymentState = PaymentStateReceived
		if inApp.IsTrialPeriod == "true" {
			paymentState = PaymentStateTrial
		}

		// Parse timestamp
		timestamp, err := strconv.Atoi(inApp.PurchaseDateMS)
		if err != nil {
			return Event{}, err
		}

		// Get price based on Product ID
		price, err := priceForAppStoreProduct(inApp.ProductID)
		if err != nil {
			return Event{}, err
		}

		return Event{
			EventType:       eventType,
			UserID:          userID,
			Platform:        "ios",
			EventTimeMillis: timestamp,
			Env:             "prod",
			Properties: EventProperties{
				PaymentState: paymentState,
				AppID:        os.Getenv("BRAZE_APP_ID"),
				ProductID:    inApp.ProductID,
				Currency:     "USD",
				Price:        price,
				Quantity:     1,
			},
		}, nil
	}

	return Event{}, errors.Errorf("processing notification type %s is not supported yet", notiType)
}
