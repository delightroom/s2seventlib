package s2seventlib

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"testing"

	"github.com/awa/go-iap/playstore"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	"google.golang.org/api/androidpublisher/v3"
)

const FAKE_USER_ID = "fake-user-id"

type MockUserStore struct{}

func (s MockUserStore) UserID(token string) (string, error) {
	return FAKE_USER_ID, nil
}

// MockPlayStoreVerifier is a mock implementation of PlayStoreVerifier interface.
// We are using mock verifier since real google publisher api calls are not idempotent.
type MockPlayStoreVerifier struct {
	result androidpublisher.SubscriptionPurchase
}

func (v MockPlayStoreVerifier) Verify(
	ctx context.Context,
	packageName string,
	productID string,
	token string,
) (*androidpublisher.SubscriptionPurchase, error) {
	return &v.result, nil
}

func loadPlayStoreTestFixture(notificationType string) (playstore.DeveloperNotification, androidpublisher.SubscriptionPurchase, CommonEvent, error) {
	bytes, err := ioutil.ReadFile(fmt.Sprintf("test_data/playstore/%s/notification.json", notificationType))
	if err != nil {
		return playstore.DeveloperNotification{}, androidpublisher.SubscriptionPurchase{}, CommonEvent{}, errors.Wrapf(err, "failed to read notification file for notification type %s", notificationType)
	}

	noti := playstore.DeveloperNotification{}

	if err := json.Unmarshal(bytes, &noti); err != nil {
		return playstore.DeveloperNotification{}, androidpublisher.SubscriptionPurchase{}, CommonEvent{}, err
	}

	bytes, err = ioutil.ReadFile(fmt.Sprintf("test_data/playstore/%s/purchase.json", notificationType))
	if err != nil {
		return playstore.DeveloperNotification{}, androidpublisher.SubscriptionPurchase{}, CommonEvent{}, errors.Wrapf(err, "failed to read purchase file for notification type %s", notificationType)
	}

	purchase := androidpublisher.SubscriptionPurchase{}

	if err := json.Unmarshal(bytes, &purchase); err != nil {
		return playstore.DeveloperNotification{}, androidpublisher.SubscriptionPurchase{}, CommonEvent{}, err
	}

	bytes, err = ioutil.ReadFile(fmt.Sprintf("test_data/playstore/%s/expected.json", notificationType))

	event := CommonEvent{}

	if err := json.Unmarshal(bytes, &event); err != nil {
		return playstore.DeveloperNotification{}, androidpublisher.SubscriptionPurchase{}, CommonEvent{}, err
	}

	return noti, purchase, event, nil
}

func loadAppStoreTestFixture(notificationType string) (AppStoreNotificationV2, error) {
	bytes, err := ioutil.ReadFile(fmt.Sprintf("test_data/appstore/%s/notification.json", notificationType))
	if err != nil {
		return AppStoreNotificationV2{}, errors.Wrapf(err, "failed to read notification file for %s", notificationType)
	}

	noti := AppStoreNotificationV2{}

	if err := json.Unmarshal(bytes, &noti); err != nil {
		return AppStoreNotificationV2{}, err
	}

	return noti, nil
}

func eventGeneratorForPlayStoreTesting(purchase androidpublisher.SubscriptionPurchase) EventGenerator {
	return NewEventGenerator(MockUserStore{}, MockPlayStoreVerifier{purchase})
}

func eventGeneratorForAppStoreTesting() EventGenerator {
	return NewEventGenerator(MockUserStore{}, MockPlayStoreVerifier{})
}

func TestPlayStorePurchasedTrialEventGeneration(t *testing.T) {
	assert := assert.New(t)

	noti, purchase, expected, err := loadPlayStoreTestFixture("purchased_trial")
	assert.NoError(err)

	eventGen := eventGeneratorForPlayStoreTesting(purchase)

	ctx := context.Background()
	event, err := eventGen.GeneratePlayStorePurchaseEvent(ctx, noti)
	assert.NoError(err)
	assert.Equal(expected, event)
}

func TestPlayStoreRenewedEventGeneration(t *testing.T) {
	assert := assert.New(t)

	noti, purchase, expected, err := loadPlayStoreTestFixture("renewed")
	assert.NoError(err)

	eventGen := eventGeneratorForPlayStoreTesting(purchase)

	ctx := context.Background()
	event, err := eventGen.GeneratePlayStorePurchaseEvent(ctx, noti)
	assert.NoError(err)
	assert.Equal(expected, event)
}

func TestPlayStoreRecoveredEventGeneration(t *testing.T) {
	assert := assert.New(t)

	noti, purchase, expected, err := loadPlayStoreTestFixture("recovered")
	assert.NoError(err)

	eventGen := eventGeneratorForPlayStoreTesting(purchase)

	ctx := context.Background()
	event, err := eventGen.GeneratePlayStorePurchaseEvent(ctx, noti)
	assert.NoError(err)
	assert.Equal(expected, event)
}

func TestPlayStoreRestartedEventGeneration(t *testing.T) {
	assert := assert.New(t)

	noti, purchase, expected, err := loadPlayStoreTestFixture("restarted")
	assert.NoError(err)

	eventGen := eventGeneratorForPlayStoreTesting(purchase)

	ctx := context.Background()
	event, err := eventGen.GeneratePlayStorePurchaseEvent(ctx, noti)
	assert.NoError(err)
	assert.Equal(expected, event)
}

func TestAppStoreInitialBuyPaidEventGeneration(t *testing.T) {
	assert := assert.New(t)

	noti, err := loadAppStoreTestFixture("initial_buy_paid")
	assert.NoError(err)

	eventGen := eventGeneratorForAppStoreTesting()

	ctx := context.Background()
	event, err := eventGen.GenerateAppStoreEvent(ctx, noti)
	assert.NoError(err)

	// base
	assert.Equal("ios", event.Platform)
	assert.Equal(FAKE_USER_ID, event.UserID)
	assert.Equal(CommonEventPurchase, event.EventType)
	assert.Equal(1616266543000, event.EventTimeMillis)
	assert.Equal("prod", event.Env)

	// properties
	assert.Equal(PaymentStateReceived, event.Properties.PaymentState)
	assert.Equal(noti.UnifiedReceipt.LatestReceiptInfo[0].ProductID, event.Properties.ProductID)
	assert.Equal("USD", event.Properties.Currency)
	assert.Equal(4.99, event.Properties.Price)
	assert.Equal(1, event.Properties.Quantity)
}

func TestAppStoreInitialBuyTrialEventGeneration(t *testing.T) {
	assert := assert.New(t)

	noti, err := loadAppStoreTestFixture("initial_buy_trial")
	assert.NoError(err)

	eventGen := eventGeneratorForAppStoreTesting()

	ctx := context.Background()
	event, err := eventGen.GenerateAppStoreEvent(ctx, noti)
	assert.NoError(err)

	// base
	assert.Equal("ios", event.Platform)
	assert.Equal(FAKE_USER_ID, event.UserID)
	assert.Equal(CommonEventPurchase, event.EventType)
	assert.Equal(1614979649000, event.EventTimeMillis)
	assert.Equal("prod", event.Env)

	// properties
	assert.Equal(PaymentStateTrial, event.Properties.PaymentState)
	assert.Equal(noti.UnifiedReceipt.LatestReceiptInfo[0].ProductID, event.Properties.ProductID)
	assert.Equal("USD", event.Properties.Currency)
	assert.Equal(4.99, event.Properties.Price)
	assert.Equal(1, event.Properties.Quantity)
}

func TestAppStoreCancelEventGeneration(t *testing.T) {
	assert := assert.New(t)

	noti, err := loadAppStoreTestFixture("cancel")
	assert.NoError(err)

	eventGen := eventGeneratorForAppStoreTesting()

	ctx := context.Background()
	event, err := eventGen.GenerateAppStoreEvent(ctx, noti)
	assert.NoError(err)

	// base
	assert.Equal("ios", event.Platform)
	assert.Equal(FAKE_USER_ID, event.UserID)
	assert.Equal(CommonEventCancel, event.EventType)
	assert.Equal(1614997940000, event.EventTimeMillis)
	assert.Equal("prod", event.Env)

	// properties
	assert.Equal(PaymentStatePending, event.Properties.PaymentState)
	assert.Equal(noti.UnifiedReceipt.LatestReceiptInfo[0].ProductID, event.Properties.ProductID)
	assert.Equal("USD", event.Properties.Currency)
	assert.Equal(4.99, event.Properties.Price)
	assert.Equal(1, event.Properties.Quantity)
	assert.Equal("0", event.Properties.CancellationReason)
}

func TestAppStoreDidRecoverEventGeneration(t *testing.T) {
	assert := assert.New(t)

	noti, err := loadAppStoreTestFixture("did_recover")
	assert.NoError(err)

	eventGen := eventGeneratorForAppStoreTesting()

	ctx := context.Background()
	event, err := eventGen.GenerateAppStoreEvent(ctx, noti)
	assert.NoError(err)

	// base
	assert.Equal("ios", event.Platform)
	assert.Equal(FAKE_USER_ID, event.UserID)
	assert.Equal(CommonEventRecover, event.EventType)
	assert.Equal(1614995088000, event.EventTimeMillis)
	assert.Equal("prod", event.Env)

	// properties
	assert.Equal(PaymentStateReceived, event.Properties.PaymentState)
	assert.Equal(noti.UnifiedReceipt.LatestReceiptInfo[0].ProductID, event.Properties.ProductID)
	assert.Equal("USD", event.Properties.Currency)
	assert.Equal(4.99, event.Properties.Price)
	assert.Equal(1, event.Properties.Quantity)
}

func TestAppStoreDidRenewEventGeneration(t *testing.T) {
	assert := assert.New(t)

	noti, err := loadAppStoreTestFixture("did_renew")
	assert.NoError(err)

	eventGen := eventGeneratorForAppStoreTesting()

	ctx := context.Background()
	event, err := eventGen.GenerateAppStoreEvent(ctx, noti)
	assert.NoError(err)

	// base
	assert.Equal("ios", event.Platform)
	assert.Equal(FAKE_USER_ID, event.UserID)
	assert.Equal(CommonEventRenew, event.EventType)
	assert.Equal(1615006877000, event.EventTimeMillis)
	assert.Equal("prod", event.Env)

	// properties
	assert.Equal(PaymentStateReceived, event.Properties.PaymentState)
	assert.Equal(noti.UnifiedReceipt.LatestReceiptInfo[0].ProductID, event.Properties.ProductID)
	assert.Equal("USD", event.Properties.Currency)
	assert.Equal(4.99, event.Properties.Price)
	assert.Equal(1, event.Properties.Quantity)
}

func TestAppStoreDidChangeRenewalStatusEventGeneration(t *testing.T) {
	assert := assert.New(t)

	noti, err := loadAppStoreTestFixture("did_change_renewal_status")
	assert.NoError(err)

	eventGen := eventGeneratorForAppStoreTesting()

	ctx := context.Background()
	event, err := eventGen.GenerateAppStoreEvent(ctx, noti)
	assert.NoError(err)

	// base
	assert.Equal("ios", event.Platform)
	assert.Equal(FAKE_USER_ID, event.UserID)
	assert.Equal(CommonEventTurnOffAutoRenew, event.EventType)
	assert.Equal(1614979114000, event.EventTimeMillis)
	assert.Equal("prod", event.Env)

	// properties
	assert.Equal(noti.UnifiedReceipt.LatestReceiptInfo[0].ProductID, event.Properties.ProductID)
}

func TestAppStoreInteractiveRenewalEventGeneration(t *testing.T) {
	assert := assert.New(t)

	noti, err := loadAppStoreTestFixture("interactive_renewal")
	assert.NoError(err)

	eventGen := eventGeneratorForAppStoreTesting()

	ctx := context.Background()
	event, err := eventGen.GenerateAppStoreEvent(ctx, noti)
	assert.NoError(err)

	// base
	assert.Equal("ios", event.Platform)
	assert.Equal(FAKE_USER_ID, event.UserID)
	assert.Equal(CommonEventReEnable, event.EventType)
	assert.Equal(1614990541000, event.EventTimeMillis)
	assert.Equal("prod", event.Env)

	// properties
	assert.Equal(PaymentStateReceived, event.Properties.PaymentState)
	assert.Equal(noti.UnifiedReceipt.LatestReceiptInfo[0].ProductID, event.Properties.ProductID)
	assert.Equal("USD", event.Properties.Currency)
	assert.Equal(3.49, event.Properties.Price)
	assert.Equal(1, event.Properties.Quantity)
}

// Helpers

func printPlayStorePurchase(notification playstore.DeveloperNotification) {
	client := GetAndroidPublisherAPIClient()
	ctx := context.Background()
	purchase, _ := client.Verify(ctx, notification.PackageName, notification.SubscriptionNotification.SubscriptionID, notification.SubscriptionNotification.SubscriptionID)
	bytes, _ := json.MarshalIndent(purchase, "", "  ")
	fmt.Printf("%s\n", string(bytes))
}
