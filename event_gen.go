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

type EventGenerator struct {
	userIDProvider    UserIDProvider
	playStoreVerifier PlayStoreVerifier
}

func NewEventGenerator(userIDProvider UserIDProvider, playStoreVerifier PlayStoreVerifier) EventGenerator {
	return EventGenerator{
		userIDProvider:    userIDProvider,
		playStoreVerifier: playStoreVerifier,
	}
}

func (g EventGenerator) GeneratePlayStoreEvent(ctx context.Context, noti playstore.DeveloperNotification) (CommonEvent, error) {
	token := noti.SubscriptionNotification.PurchaseToken
	notiType := noti.SubscriptionNotification.NotificationType
	userID, err := g.userIDProvider.UserID(token)

	if err != nil {
		return CommonEvent{}, err
	}

	fmt.Printf("token: %s, notiType: %d, userID: %s\n", token, notiType, userID)

	var eventType CommonEventType
	switch notiType {
	case playstore.SubscriptionNotificationTypePurchased:
		eventType = CommonEventPurchase
	case playstore.SubscriptionNotificationTypeRenewed:
		eventType = CommonEventRenew
	case playstore.SubscriptionNotificationTypeRecovered:
		eventType = CommonEventRecover
	case playstore.SubscriptionNotificationTypeRestarted:
		eventType = CommonEventRestart
	case playstore.SubscriptionNotificationTypeRevoked:
		eventType = CommonEventCancel
	case playstore.SubscriptionNotificationTypeCanceled:
		eventType = CommonEventTurnOffAutoRenew
	default:
		return CommonEvent{}, errors.Errorf("not purchase event: %d", notiType)
	}

	purchase, err := g.playStoreVerifier.Verify(ctx, noti.PackageName, noti.SubscriptionNotification.SubscriptionID, token)

	if err != nil {
		return CommonEvent{}, err
	}

	timestamp, err := strconv.Atoi(noti.EventTimeMillis)
	if err != nil {
		return CommonEvent{}, err
	}

	props := CommonEventProperties{
		PaymentState: PaymentState(purchase.PaymentState),
		AppID:        os.Getenv("BRAZE_APP_ID"),
		ProductID:    noti.SubscriptionNotification.SubscriptionID,
		Currency:     purchase.PriceCurrencyCode,
		Price:        float64(purchase.PriceAmountMicros) / 1_000_000,
		Quantity:     1,
	}

	if eventType == CommonEventCancel || eventType == CommonEventTurnOffAutoRenew {
		props.CancellationReason = strconv.FormatInt(purchase.CancelReason, 10)
	}

	return CommonEvent{
		EventType:       eventType,
		UserID:          userID,
		Platform:        "android",
		EventTimeMillis: timestamp,
		Env:             "prod",
		Properties:      props,
	}, nil
}

func priceForAppStoreProduct(productID string) (float64, error) {
	switch productID {
	case "droom.sleepIfUCanFree.premium.monthly.1":
		return 4.99, nil
	case "droom.sleepIfUCanFree.premium.monthly.4":
		return 4.99, nil
	case "droom.sleepIfUCanFree.premium.monthlyPromo.4":
		return 3.49, nil
	case "droom.sleepIfUCanFree.premium.yearly.4":
		return 41.99, nil
	case "droom.sleepIfUCanFree.premium.monthlyDecoy01.4":
		return 6.99, nil
	case "droom.sleepIfUCanFree.premium.monthlyDecoy02.4":
		return 7.49, nil
	case "droom.sleepIfUCanFree.premium.monthlyDecoy03.4":
		return 9.99, nil
	case "droom.sleepIfUCanFree.premium.yearly01.4":
		return 59.99, nil
	default:
		return 0, errors.Errorf("cannot find price info for the productID: %s", productID)
	}
}

func (g EventGenerator) GenerateAppStoreEvent(ctx context.Context, noti appstore.SubscriptionNotification) (CommonEvent, error) {

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
					return CommonEvent{}, err
				}

				timestamp, err := strconv.Atoi(receiptInfo.CancellationDateMS)
				if err != nil {
					return CommonEvent{}, err
				}

				// Get price based on Product ID
				price, err := priceForAppStoreProduct(receiptInfo.ProductID)
				if err != nil {
					return CommonEvent{}, err
				}

				return CommonEvent{
					EventType:       CommonEventCancel,
					UserID:          userID,
					Platform:        "ios",
					EventTimeMillis: timestamp,
					Env:             env,
					Properties: CommonEventProperties{
						Price:              price,
						Currency:           "USD",
						Quantity:           1,
						ProductID:          receiptInfo.ProductID,
						CancellationReason: receiptInfo.CancellationReason,
					},
				}, err
			}
		}

		return CommonEvent{}, errors.Errorf("failed to find receiptInfo for webOrderLineItemID: %s", webOrderLineItemID)
	} else if notiType == appstore.NotificationTypeDidChangeRenewalStatus {
		inApp := noti.UnifiedReceipt.LatestReceiptInfo[0]
		token := inApp.OriginalTransactionID
		userID, err := g.userIDProvider.UserID(token)
		if err != nil {
			return CommonEvent{}, err
		}

		var eventType CommonEventType
		if noti.AutoRenewStatus == "true" {
			eventType = CommonEventTurnOnAutoRenew
		} else {
			eventType = CommonEventTurnOffAutoRenew
		}

		timestamp, err := strconv.Atoi(noti.AutoRenewStatusChangeDateMS)
		if err != nil {
			return CommonEvent{}, err
		}

		return CommonEvent{
			EventType:       eventType,
			UserID:          userID,
			Platform:        "ios",
			EventTimeMillis: timestamp,
			Env:             env,
			Properties: CommonEventProperties{
				ProductID: noti.AutoRenewProductID,
			},
		}, err
	} else if notiType == appstore.NotificationTypeInitialBuy || notiType == appstore.NotificationTypeDidRenew || notiType == appstore.NotificationTypeDidRecover || notiType == appstore.NotificationTypeInteractiveRenewal {
		inApp := noti.UnifiedReceipt.LatestReceiptInfo[0]
		token := inApp.OriginalTransactionID
		userID, err := g.userIDProvider.UserID(token)
		if err != nil {
			return CommonEvent{}, err
		}

		// Determine event type
		var eventType CommonEventType
		switch notiType {
		case appstore.NotificationTypeInitialBuy:
			eventType = CommonEventPurchase
		case appstore.NotificationTypeDidRecover:
			eventType = CommonEventRecover
		case appstore.NotificationTypeInteractiveRenewal:
			eventType = CommonEventRestart
		default:
			eventType = CommonEventRenew
		}

		// Get payment state
		var paymentState PaymentState = PaymentStateReceived
		if inApp.IsTrialPeriod == "true" {
			paymentState = PaymentStateTrial
		}

		// Parse timestamp
		timestamp, err := strconv.Atoi(inApp.PurchaseDateMS)
		if err != nil {
			return CommonEvent{}, err
		}

		// Get price based on Product ID
		price, err := priceForAppStoreProduct(inApp.ProductID)
		if err != nil {
			return CommonEvent{}, err
		}

		return CommonEvent{
			EventType:       eventType,
			UserID:          userID,
			Platform:        "ios",
			EventTimeMillis: timestamp,
			Env:             "prod",
			Properties: CommonEventProperties{
				PaymentState: paymentState,
				AppID:        os.Getenv("BRAZE_APP_ID"),
				ProductID:    inApp.ProductID,
				Currency:     "USD",
				Price:        price,
				Quantity:     1,
			},
		}, nil
	}

	return CommonEvent{}, errors.Errorf("processing notification type %s is not supported yet", notiType)
}
