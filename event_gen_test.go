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

func TestPlayStorePurchasedTrialEventGeneration(t *testing.T) {
	testPlayStoreEventGeneration(t, "purchased_trial")
}

func TestPlayStoreRenewedEventGeneration(t *testing.T) {
	testPlayStoreEventGeneration(t, "renewed")
}

func TestPlayStoreRecoveredEventGeneration(t *testing.T) {
	testPlayStoreEventGeneration(t, "recovered")
}

func TestPlayStoreRestartedEventGeneration(t *testing.T) {
	testPlayStoreEventGeneration(t, "restarted")
}

func TestAppStoreInitialBuyPaidEventGeneration(t *testing.T) {
	testAppStoreEventGeneration(t, "initial_buy_paid")
}

func TestAppStoreInitialBuyTrialEventGeneration(t *testing.T) {
	testAppStoreEventGeneration(t, "initial_buy_trial")
}

func TestAppStoreCancelEventGeneration(t *testing.T) {
	testAppStoreEventGeneration(t, "cancel")
}

func TestAppStoreDidRecoverEventGeneration(t *testing.T) {
	testAppStoreEventGeneration(t, "did_recover")
}

func TestAppStoreDidRenewEventGeneration(t *testing.T) {
	testAppStoreEventGeneration(t, "did_renew")
}

func TestAppStoreDidChangeRenewalStatusEventGeneration(t *testing.T) {
	testAppStoreEventGeneration(t, "did_change_renewal_status")
}

func TestAppStoreInteractiveRenewalEventGeneration(t *testing.T) {
	testAppStoreEventGeneration(t, "interactive_renewal")
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

func loadPlayStoreTestFixture(notificationType string) (playstore.DeveloperNotification, androidpublisher.SubscriptionPurchase, CommonEvent, error) {
	noti := playstore.DeveloperNotification{}

	if err := loadJSONFile(fmt.Sprintf("test_data/playstore/%s/notification.json", notificationType), &noti); err != nil {
		return playstore.DeveloperNotification{}, androidpublisher.SubscriptionPurchase{}, CommonEvent{}, err
	}

	purchase := androidpublisher.SubscriptionPurchase{}

	if err := loadJSONFile(fmt.Sprintf("test_data/playstore/%s/purchase.json", notificationType), &purchase); err != nil {
		return playstore.DeveloperNotification{}, androidpublisher.SubscriptionPurchase{}, CommonEvent{}, err
	}

	event := CommonEvent{}

	if err := loadJSONFile(fmt.Sprintf("test_data/playstore/%s/expected.json", notificationType), &event); err != nil {
		return playstore.DeveloperNotification{}, androidpublisher.SubscriptionPurchase{}, CommonEvent{}, err
	}

	return noti, purchase, event, nil
}

func loadAppStoreTestFixture(notificationType string) (AppStoreNotificationV2, CommonEvent, error) {
	noti := AppStoreNotificationV2{}

	if err := loadJSONFile(fmt.Sprintf("test_data/appstore/%s/notification.json", notificationType), &noti); err != nil {
		return AppStoreNotificationV2{}, CommonEvent{}, err
	}

	expected := CommonEvent{}

	if err := loadJSONFile(fmt.Sprintf("test_data/appstore/%s/expected.json", notificationType), &expected); err != nil {
		return AppStoreNotificationV2{}, CommonEvent{}, err
	}

	return noti, expected, nil
}

func eventGeneratorForPlayStoreTesting(purchase androidpublisher.SubscriptionPurchase) EventGenerator {
	return NewEventGenerator(MockUserStore{}, MockPlayStoreVerifier{purchase})
}

func eventGeneratorForAppStoreTesting() EventGenerator {
	return NewEventGenerator(MockUserStore{}, MockPlayStoreVerifier{})
}

func testPlayStoreEventGeneration(t *testing.T, notificationType string) {
	assert := assert.New(t)

	noti, purchase, expected, err := loadPlayStoreTestFixture(notificationType)
	assert.NoError(err)

	eventGen := eventGeneratorForPlayStoreTesting(purchase)

	ctx := context.Background()
	event, err := eventGen.GeneratePlayStorePurchaseEvent(ctx, noti)
	assert.NoError(err)
	assert.Equal(expected, event)
}

func testAppStoreEventGeneration(t *testing.T, notificationType string) {
	assert := assert.New(t)

	noti, expected, err := loadAppStoreTestFixture(notificationType)
	assert.NoError(err)

	eventGen := eventGeneratorForAppStoreTesting()

	ctx := context.Background()
	event, err := eventGen.GenerateAppStoreEvent(ctx, noti)
	assert.NoError(err)
	assert.Equal(expected, event)
}

func printPlayStorePurchase(notification playstore.DeveloperNotification) {
	client := GetAndroidPublisherAPIClient()
	ctx := context.Background()
	purchase, _ := client.Verify(ctx, notification.PackageName, notification.SubscriptionNotification.SubscriptionID, notification.SubscriptionNotification.SubscriptionID)
	bytes, _ := json.MarshalIndent(purchase, "", "  ")
	fmt.Printf("%s\n", string(bytes))
}
