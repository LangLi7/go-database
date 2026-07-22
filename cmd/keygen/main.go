// keygen generates Ed25519 keypairs for passwordless admin login and can sign
// a challenge nonce (SSH-style). Output is JSON so it can be piped into env or
// used by a login client.
//
//	go run ./cmd/keygen generate          # -> {"accounts":{"chang":{...},"hermes":{...}}}
//	go run ./cmd/keygen sign <priv> <nonce>   # -> base64 signature
package main

import (
	"encoding/json"
	"fmt"
	"os"

	"go-database/internal/auth"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintln(os.Stderr, "usage: keygen generate | keygen sign <privkey> <nonce>")
		os.Exit(1)
	}
	switch os.Args[1] {
	case "generate":
		bundle := auth.AdminKeyBundle{Accounts: map[string]auth.KeyPair{}}
		for _, name := range []string{"chang", "hermes"} {
			kp, err := auth.GenerateKeyPair()
			if err != nil {
				fmt.Fprintln(os.Stderr, "generate:", err)
				os.Exit(1)
			}
			bundle.Accounts[name] = kp
		}
		out, _ := json.MarshalIndent(bundle, "", "  ")
		fmt.Println(string(out))

		// Also print the GODB_ADMIN_PUBKEYS env value for bootstrapping.
		pub := map[string]string{}
		for k, v := range bundle.Accounts {
			pub[k] = v.PublicKey
		}
		pubJSON, _ := json.Marshal(pub)
		fmt.Fprintf(os.Stderr, "\n# set this env var to bootstrap passwordless admins:\nGODB_ADMIN_PUBKEYS=%s\n", string(pubJSON))

	case "sign":
		if len(os.Args) < 4 {
			fmt.Fprintln(os.Stderr, "usage: keygen sign <privkey> <nonce>")
			os.Exit(1)
		}
		sig, err := auth.SignChallenge(os.Args[2], os.Args[3])
		if err != nil {
			fmt.Fprintln(os.Stderr, "sign:", err)
			os.Exit(1)
		}
		fmt.Println(sig)
	default:
		fmt.Fprintln(os.Stderr, "unknown command:", os.Args[1])
		os.Exit(1)
	}
}
