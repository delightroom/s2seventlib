package s2seventlib

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"testing"

	"github.com/awa/go-iap/appstore"
	"github.com/awa/go-iap/playstore"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	"google.golang.org/api/androidpublisher/v3"
)

// Mocks

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

// Tests

func TestPlayStorePurchasedTrialEventConversion(t *testing.T) {
	testPlayStoreEventConversion(t, "purchased_trial")
}

func TestPlayStoreRenewedEventConversion(t *testing.T) {
	testPlayStoreEventConversion(t, "renewed")
}

func TestPlayStoreRecoveredEventConversion(t *testing.T) {
	testPlayStoreEventConversion(t, "recovered")
}

func TestPlayStoreRestartedEventConversion(t *testing.T) {
	testPlayStoreEventConversion(t, "restarted")
}

func TestPlayStoreRevokedEventConversion(t *testing.T) {
	testPlayStoreEventConversion(t, "revoked")
}

func TestPlayStoreCanceledEventConversion(t *testing.T) {
	testPlayStoreEventConversion(t, "canceled")
}

func TestAppStoreInitialBuyPaidEventConversion(t *testing.T) {
	testAppStoreEventConversion(t, "initial_buy_paid")
}

func TestAppStoreInitialBuyTrialEventConversion(t *testing.T) {
	testAppStoreEventConversion(t, "initial_buy_trial")
}

func TestAppStoreCancelEventConversion(t *testing.T) {
	testAppStoreEventConversion(t, "cancel")
}

func TestAppStoreDidRecoverEventConversion(t *testing.T) {
	testAppStoreEventConversion(t, "did_recover")
}

func TestAppStoreDidRenewEventConversion(t *testing.T) {
	testAppStoreEventConversion(t, "did_renew")
}

func TestAppStoreDidChangeRenewalStatusEventConversion(t *testing.T) {
	testAppStoreEventConversion(t, "did_change_renewal_status")
}

func TestAppStoreInteractiveRenewalEventConversion(t *testing.T) {
	testAppStoreEventConversion(t, "interactive_renewal")
}

// Helpers

func loadJSONFile(filepath string, dest interface{}) error {
	bytes, err := ioutil.ReadFile(filepath)
	if err != nil {
		return errors.Wrapf(err, "failed to read notification file: %s", filepath)
	}

	if err := json.Unmarshal(bytes, dest); err != nil {
		return err
	}

	return nil
}

func loadPlayStoreTestFixture(notificationType string) (playstore.DeveloperNotification, androidpublisher.SubscriptionPurchase, Event, error) {
	noti := playstore.DeveloperNotification{}

	if err := loadJSONFile(fmt.Sprintf("test_data/playstore/%s/notification.json", notificationType), &noti); err != nil {
		return playstore.DeveloperNotification{}, androidpublisher.SubscriptionPurchase{}, Event{}, err
	}

	purchase := androidpublisher.SubscriptionPurchase{}

	if err := loadJSONFile(fmt.Sprintf("test_data/playstore/%s/purchase.json", notificationType), &purchase); err != nil {
		return playstore.DeveloperNotification{}, androidpublisher.SubscriptionPurchase{}, Event{}, err
	}

	event := Event{}

	if err := loadJSONFile(fmt.Sprintf("test_data/playstore/%s/expected.json", notificationType), &event); err != nil {
		return playstore.DeveloperNotification{}, androidpublisher.SubscriptionPurchase{}, Event{}, err
	}

	return noti, purchase, event, nil
}

func loadAppStoreTestFixture(notificationType string) (appstore.SubscriptionNotification, Event, error) {
	noti := appstore.SubscriptionNotification{}

	if err := loadJSONFile(fmt.Sprintf("test_data/appstore/%s/notification.json", notificationType), &noti); err != nil {
		return appstore.SubscriptionNotification{}, Event{}, err
	}

	expected := Event{}

	if err := loadJSONFile(fmt.Sprintf("test_data/appstore/%s/expected.json", notificationType), &expected); err != nil {
		return appstore.SubscriptionNotification{}, Event{}, err
	}

	return noti, expected, nil
}

func eventConverterForPlayStoreTesting(purchase androidpublisher.SubscriptionPurchase) EventConverter {
	return NewEventConverter(MockUserStore{}, MockPlayStoreVerifier{purchase})
}

func eventConverterForAppStoreTesting() EventConverter {
	return NewEventConverter(MockUserStore{}, MockPlayStoreVerifier{})
}

func testPlayStoreEventConversion(t *testing.T, notificationType string) {
	t.Helper()
	assert := assert.New(t)

	noti, purchase, expected, err := loadPlayStoreTestFixture(notificationType)
	assert.NoError(err)
	// fmt.Println("🌻")
	// printPlayStorePurchase(noti)
	// fmt.Println("🌻")

	eventGen := eventConverterForPlayStoreTesting(purchase)

	ctx := context.Background()
	event, err := eventGen.ConvertPlayStoreEvent(ctx, noti)
	assert.NoError(err)
	assert.Equal(expected, event)
}

func testAppStoreEventConversion(t *testing.T, notificationType string) {
	assert := assert.New(t)

	noti, expected, err := loadAppStoreTestFixture(notificationType)
	assert.NoError(err)

	eventGen := eventConverterForAppStoreTesting()

	ctx := context.Background()
	event, err := eventGen.ConvertAppStoreEvent(ctx, noti)
	assert.NoError(err)
	assert.Equal(expected, event)
}

func printPlayStorePurchase(notification playstore.DeveloperNotification) {
	client := GetAndroidPublisherAPIClient()
	ctx := context.Background()
	purchase, err := client.Verify(ctx, notification.PackageName, notification.SubscriptionNotification.SubscriptionID, notification.SubscriptionNotification.PurchaseToken)
	fmt.Println("notification.PackageName:", notification.PackageName)
	fmt.Println("notification.SubscriptionNotification.NotificationType:", notification.SubscriptionNotification.NotificationType)
	fmt.Println("notification.SubscriptionNotification.SubscriptionID:", notification.SubscriptionNotification.SubscriptionID)
	fmt.Println("notification.SubscriptionNotification.PurchaseToken:", notification.SubscriptionNotification.PurchaseToken)

	if err != nil {
		fmt.Println("🐛", err)
	}
	bytes, _ := json.MarshalIndent(purchase, "", "  ")
	fmt.Printf("%s\n", string(bytes))
}
