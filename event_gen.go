package s2seventlib

import (
	"context"
	"os"
	"strconv"

	"github.com/awa/go-iap/appstore"
	"github.com/awa/go-iap/playstore"
	"github.com/pkg/errors"
)

type AppStoreNotificationV2 struct {
	// original notification
	appstore.SubscriptionNotification

	// missing properties
	AutoRenewStatusChangeDateMS string `json:"auto_renew_status_change_date_ms"`
}

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

func (g EventGenerator) GeneratePlayStorePurchaseEvent(ctx context.Context, noti playstore.DeveloperNotification) (CommonEvent, error) {
	token := noti.SubscriptionNotification.PurchaseToken
	notiType := noti.SubscriptionNotification.NotificationType
	// msg := fmt.Sprintf("%s, %d", token, notiType)
	// fmt.Println(msg)

	userID, err := g.userIDProvider.UserID(token)

	// fmt.Printf("userId: %s\n", userID)

	if err != nil {
		return CommonEvent{}, err
	}

	var eventType CommonEventType
	switch notiType {
	case playstore.SubscriptionNotificationTypePurchased:
		eventType = CommonEventPurchase
	case playstore.SubscriptionNotificationTypeRenewed:
		eventType = CommonEventRenew
	case playstore.SubscriptionNotificationTypeRecovered:
		eventType = CommonEventRecover
	case playstore.SubscriptionNotificationTypeRestarted:
		eventType = CommonEventReEnable
	default:
		return CommonEvent{}, errors.Errorf("not purchase event: %d", notiType)
	}

	purchase, err := g.playStoreVerifier.Verify(ctx, noti.PackageName, noti.SubscriptionNotification.SubscriptionID, token)

	// fmt.Printf("%+v\n", purchase)

	// bytes, _ := json.MarshalIndent(purchase, "", "\t")
	// fmt.Printf("%s\n", string(bytes))

	if err != nil {
		return CommonEvent{}, err
	}

	// fmt.Printf("type: %d, state: %s, price: %d, currency:%s\n", notiType, paymentState, purchase.PriceAmountMicros, purchase.PriceCurrencyCode)

	timestamp, err := strconv.Atoi(noti.EventTimeMillis)
	if err != nil {
		return CommonEvent{}, err
	}

	return CommonEvent{
		EventType:       eventType,
		UserID:          userID,
		Platform:        "android",
		EventTimeMillis: timestamp,
		Env:             "prod",
		Properties: CommonEventProperties{
			PaymentState: PaymentState(purchase.PaymentState),
			AppID:        os.Getenv("BRAZE_APP_ID"),
			ProductID:    noti.SubscriptionNotification.SubscriptionID,
			Currency:     purchase.PriceCurrencyCode,
			Price:        float64(purchase.PriceAmountMicros) / 1_000_000,
			Quantity:     1,
		},
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
	default:
		return 0, errors.Errorf("cannot find price info for the productID: %s", productID)
	}
}

func (g EventGenerator) GenerateAppStoreEvent(ctx context.Context, noti AppStoreNotificationV2) (CommonEvent, error) {

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
			eventType = CommonEventReEnable
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
