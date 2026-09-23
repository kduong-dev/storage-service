// api-key-generator issues a new API key for a storage namespace. The raw key
// goes to the calling service (STORAGE_SERVICE_API_KEY); the hash goes into
// the storage service's STORAGE_CLIENTS_B64_JSON.
package main

import (
	"crypto/rand"
	"encoding/hex"
	"flag"
	"fmt"

	"github.com/kduong-dev/goutil/fatal"
	"github.com/kduong-dev/storage-service/internal/apikey"
)

func main() {
	namespace := flag.String("namespace", "", "namespace the key grants access to, e.g. trading-core")
	flag.Parse()
	fatal.Unless(*namespace != "", "-namespace is required")
	secret := make([]byte, 32)
	_, err := rand.Read(secret)
	fatal.OnError(err)
	apiKey := hex.EncodeToString(secret)
	keyHash := apikey.HashAPIKey(apiKey)
	fmt.Printf("namespace:            %s\n", *namespace)
	fmt.Printf("api key (client):     %s\n", apiKey)
	fmt.Printf("key hash (storage):   %s\n", keyHash)
	fmt.Printf("clients entry (JSON): {%q: %q}\n", *namespace, keyHash)
	fmt.Printf("single-client STORAGE_CLIENTS_B64_JSON: %s\n", apikey.EncodeClients(map[string]string{*namespace: keyHash}))
}
