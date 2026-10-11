package encryption

const (
	// EnvAesKey is the environment variable for the AES key.
	EnvAesKey = "KUBEXPORTER_AES_KEY"
	// EnvAgePublicKey is the environment variable for the age public key.
	EnvAgePublicKey = "KUBEXPORTER_AGE_PUBLIC_KEY"
	// EnvAgeIdentity is the environment variable for the age identity.
	EnvAgeIdentity = "KUBEXPORTER_AGE_IDENTITY"
)

// DecryptResult holds the result of decrypting fields.
type DecryptResult struct {
	Count  int
	HasAES bool
	HasAge bool
}
