package s2seventlib

import (
	"context"
	"fmt"
	"os"
	"sync"

	"github.com/awa/go-iap/playstore"

	"google.golang.org/api/androidpublisher/v3"
)

type PlayStoreVerifier interface {
	Verify(
		ctx context.Context,
		packageName string,
		productID string,
		token string,
	) (*androidpublisher.SubscriptionPurchase, error)
}

// AndroidPublisherAPIClient is a verifier implementation for App Store
type AndroidPublisherAPIClient struct {
	jsonKey []byte
}

var androidPublisherAPIClient *AndroidPublisherAPIClient
var onceGooglePlay sync.Once

// GetAndroidPublisherAPIClient returns singleton object of googlePlayVerifier
func GetAndroidPublisherAPIClient() *AndroidPublisherAPIClient {
	onceGooglePlay.Do(func() {
		jsonKey := []byte(os.Getenv("GOOGLE_JSON_KEY"))
		androidPublisherAPIClient = &AndroidPublisherAPIClient{jsonKey: jsonKey}
	})

	return androidPublisherAPIClient
}

// Verify verifies Play Store purchase token via Play Store Server.
func (c AndroidPublisherAPIClient) Verify(
	ctx context.Context,
	packageName string,
	productID string,
	token string,
) (*androidpublisher.SubscriptionPurchase, error) {

	client, err := playstore.New(c.jsonKey)

	if err != nil {
		fmt.Printf("fail to initialize GooglePlay client: %v", err)
		return nil, err
	}

	return client.VerifySubscription(ctx, packageName, productID, token)
}
