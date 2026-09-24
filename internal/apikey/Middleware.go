package apikey

import (
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"net/http"
	"regexp"
	"strings"

	"github.com/ansel1/merry"
	"github.com/kduong-dev/goutil/config"
	"github.com/kduong-dev/goutil/fatal"
	"github.com/kduong-dev/goutil/httpx"
)

var namespacePattern = regexp.MustCompile(`^[a-z0-9][a-z0-9-]*$`)

// Middleware authenticates calling services by API key and scopes each
// request to the namespace that key was issued for. Only SHA-256 hashes of
// keys are held, so the storage side never needs the raw keys.
type Middleware struct {
	namespaceByKeyHash map[string]string
}

type NewMiddlewareInput struct {
	NamespaceByKeyHash map[string]string
}

func NewMiddleware(input NewMiddlewareInput) *Middleware {
	for keyHash, namespace := range input.NamespaceByKeyHash {
		fatal.Unlessf(namespacePattern.MatchString(namespace), "invalid namespace %q: must match %s", namespace, namespacePattern)
		fatal.Unlessf(len(keyHash) == sha256.Size*2, "invalid key hash for namespace %q: must be hex-encoded SHA-256", namespace)
	}
	return &Middleware{
		namespaceByKeyHash: input.NamespaceByKeyHash,
	}
}

// MiddlewareFromEnv reads STORAGE_CLIENTS_B64_JSON: base64-encoded JSON
// mapping each namespace to the hex SHA-256 hash of its API key.
func MiddlewareFromEnv() *Middleware {
	data, err := base64.StdEncoding.DecodeString(config.EnvStringOrFatal("STORAGE_CLIENTS_B64_JSON"))
	fatal.OnError(err, "decoding STORAGE_CLIENTS_B64_JSON: ")
	var keyHashByNamespace map[string]string
	fatal.UnlessUnmarshal(data, &keyHashByNamespace)
	namespaceByKeyHash := make(map[string]string, len(keyHashByNamespace))
	for namespace, keyHash := range keyHashByNamespace {
		namespaceByKeyHash[strings.ToLower(keyHash)] = namespace
	}
	for keyHash, namespace := range namespaceByKeyHash {
		fatal.Unlessf(namespacePattern.MatchString(namespace), "invalid namespace %q: must match %s", namespace, namespacePattern)
		fatal.Unlessf(len(keyHash) == sha256.Size*2, "invalid key hash for namespace %q: must be hex-encoded SHA-256", namespace)
	}
	return NewMiddleware(NewMiddlewareInput{
		NamespaceByKeyHash: namespaceByKeyHash,
	})
}

func HashAPIKey(apiKey string) string {
	sum := sha256.Sum256([]byte(apiKey))
	return hex.EncodeToString(sum[:])
}

func (middleware *Middleware) Handle(next http.Handler) http.Handler {
	return http.HandlerFunc(func(responseWriter http.ResponseWriter, request *http.Request) {
		var err error
		defer func() {
			if err != nil {
				httpx.SendErrorResponse(responseWriter, err)
			}
		}()
		authorization := request.Header.Get("Authorization")
		apiKey, found := strings.CutPrefix(authorization, "Bearer ")
		if !found || apiKey == "" {
			err = merry.New("missing api key").WithHTTPCode(http.StatusUnauthorized).WithUserMessage("unauthorized")
			return
		}
		namespace, ok := middleware.namespaceByKeyHash[HashAPIKey(apiKey)]
		if !ok {
			err = merry.New("invalid api key").WithHTTPCode(http.StatusUnauthorized).WithUserMessage("unauthorized")
			return
		}
		ctx := WithNamespace(request.Context(), namespace)
		next.ServeHTTP(responseWriter, request.WithContext(ctx))
	})
}
