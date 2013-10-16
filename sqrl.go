/*
	The sqrl package implements the Secure QR Login (SQRL) protocal for both server and client. See https://www.grc.com/sqrl/sqrl.htm for more information regarding SQRL.

	!!!!!!!!!!!!!!!!!!!!!!!!! WARNING !!!!!!!!!!!!!!!!!!!!!!!!!
	!!       EXPERIMENTAL CODE - USE AT YOUR OWN RISK!       !!
	!!!!!!!!!!!!!!!!!!!!!!!!! WARNING !!!!!!!!!!!!!!!!!!!!!!!!!

	The MIT License (MIT)

	Copyright (c) 2013 sedalu

	Permission is hereby granted, free of charge, to any person obtaining a copy of this software and associated documentation files (the "Software"), to deal in the Software without restriction, including without limitation the rights to use, copy, modify, merge, publish, distribute, sublicense, and/or sell copies of the Software, and to permit persons to whom the Software is furnished to do so, subject to the following conditions:

	The above copyright notice and this permission notice shall be included in all copies or substantial portions of the Software.

	THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY, FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM, OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE SOFTWARE.
*/

package sqrl

import (
	"bytes"
	"code.google.com/p/go.crypto/scrypt"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"fmt"
	// "github.com/agl/ed25519"
	"github.com/dustyburwell/ed25519"
	"net/http"
	"net/url"
	"strconv"
	"strings"
)

type Version int

const (
	SQRL1 Version = iota
)

const keyLen int = 32

type Option int

const (
	None Option = 1 << iota - 1
	Enforce
)

type Identity struct {
	masterKey []byte // 256-bit master key - useless without XORing with user passcode
	check     []byte // 128-bit passcode check - must match lower 128-bits of hashed user passcode 
	salt      []byte //  64-bit salt
	n, r, p   int
}

func NewIdentity(passcode []byte) *Identity {
	id := new(Identity)
	id.n, id.r, id.p = 16384, 8, 1

	// Generate new 256-bit master id
	// masterId := cryptoRand(32)

	// Generate new 64-bit salt
	id.salt = cryptoRand(8)
	
	// Generate new 128-bit passcode check

	// Generate new 256-bit masster key

	return id
}

func (this *Identity) ExportMasterKey(passcode []byte) []byte {
	// Get SQRL masterID
	// masterId, err := this.masterID(passcode)

	// TODO Refactor steps into separate functions
	// STEP 4: Create a new password salt
	// STEP 5: SCrypt the current password and newPasswordSalt with WAY more difficult SCryptParameters
	// STEP 6: SHA256 the SCrypt result from STEP 5 to create the new password verifier
	// STEP 7: XOR the original master key with the SCrypt result from STEP 5 to create the new master identity key
	// Return a new Identity with the new password salt, password verify, password parameters and master identity key
	return nil
}

func (this *Identity) ChangePasscode(passcode []byte) {
	// Get SQRL masterID
	// masterId, err := this.masterID(passcode)

	// TODO Refactor steps into separate functions
	// STEP 4: Create a new password salt
	// STEP 5: SCrypt the newPassword and newPasswordSalt
	// STEP 6: SHA256 the SCrypt result from STEP 5 to create the new password verifier
	// STEP 7: XOR the original master key with the SCrypt result from STEP 5 to create the new master identity key
	// Return a new SQRLIdentity with the new password salt, password verify, and master identity key
	// Note: the password is not permanently changed until this new identity object is written over the old identity on disk
}

type Client struct {
	id *Identity
}

func NewClient() *Client {
	client := new(Client)
	client.id = new(Identity)
	return client
}

func (this *Client) Authenticate(siteURL string, passcode []byte, options Option) (request *http.Request, err error) {
	// Get masterID
	masterId, err := this.id.masterID(passcode)

	if err != nil {
		return
	}

	// Get siteID
	siteId, err := generateSiteId(masterId, siteURL)

	if err != nil {
		return
	}

	// STEP 5: Synthesize a public key by using the result from STEP 4
	// Alternative:
	//	privateKeyBuf := bytes.NewBuffer(siteId)
	//	sqrlkey, _, err := ed25519.GenerateKey(privateKeyBuf)
	publicKey := ed25519.GeneratePublicKey(&siteId)[:]
	sqrlkey := base64.URLEncoding.EncodeToString(publicKey)
	sqrlkey = strings.TrimRight(sqrlkey, "=")

	// STEP 6: Built the signable URL
	sqrlURL := siteURL
	sqrlURL += fmt.Sprintf("&%s=%s", "sqrlver", SQRL1)

	if options != None {
		sqrlURL += fmt.Sprintf("&%s=%s", "sqrlopt", "")
	}

	sqrlURL += fmt.Sprintf("&%s=%s", "sqrlkey", sqrlkey)

	// STEP 7: Sign the signable URL with the private key from STEP 4
	sig := ed25519.Sign(&siteId, []byte(sqrlURL))[:]
	sqrlsig := base64.URLEncoding.EncodeToString(sig)
	sqrlsig = strings.TrimRight(sqrlsig, "=")

	// Return authentication object containing all the outputs which are to be sent to the server
	body := bytes.NewBufferString(fmt.Sprintf("&%s=%s", "sqrlsig", sqrlsig))
	request, err = http.NewRequest("POST", sqrlURL, body)
	// request.RequestURI = siteURL
	return
}

func (this *Identity) masterID(passcode []byte) (masterId []byte, err error) {
	// TODO Refactor steps into separate functions

	// Derive key of lenth keyLen from passcode and salt using SCrypt with cost parameters N, r, and P.
	// This is the expensive operation and its parameters should be tuned so that this operation takes between 1-2 seconds to perform.
	key, err := scrypt.Key(passcode, this.salt, this.n, this.r, this.p, keyLen)

	// If there was an error, return err
	if err != nil {
		return
	}

	// Verify the key using this.check. Return an error on failure.
	if !verifyKey(key, this.check) {
		err = errors.New("Passcode hash does not match check.")
		return
	}

	// STEP 3: XOR the master identity key from the Identity with the result from STEP 1 to create the original master key
	if len(this.masterKey) != 32 || len(key) != 32 {
		// this.masterKey and userkey are not of equal length
		return
	}

	subtle.ConstantTimeCopy(1, masterId, this.masterKey)

	for i, _ := range masterId {
		masterId[i] ^= key[i]
	}

	return
}

//////////////////////// HELPER FUNCTIONS ////////////////////////

// cryptoRand generates n random bytes using crypto/rand.
func cryptoRand(n uint) (bytes []byte) {
	bytes = make([]byte, n)
	rand.Read(bytes)
	return
}

// hashKey generates the SHA256 hash of key.
func hashKey(key []byte) (hash []byte) {
	h := sha256.New()
	h.Write(key)
	return h.Sum(nil)
}

// verifyHash compares the first len(check) bytes of hash against check.
func verifyHash(hash, check []byte) (ok bool) {
	n := len(check)

	// Perform a constant time comparison. Return false if different.
	if subtle.ConstantTimeCompare(hash[:n], check) != 1 {
		return
	}

	// Everything is ok, return true.
	return true
}

// verifyKey compares the SHA256 hash of key against check.
func verifyKey(key, check []byte) (ok bool) {
	// Get the SHA256 hash of key.
	hash := hashKey(key)

	// Verify hash against check. Return false on failure.
	if verifyHash(hash, check) {
		return
	}

	// Everything is ok, return true.
	return true
}

func generateSiteId(masterId []byte, siteURL string) (siteId [64]byte, err error) {
	// HMACSHA-256 masterId with the host from siteURL
	url, _ := url.Parse(siteURL)
	host := url.Host
	d, _ := strconv.Atoi(url.Query().Get("d"))

	if d > 1 && d <= len(url.Path) {
		host += url.Path[:d]
	}

	mac := hmac.New(sha256.New, masterId)
	mac.Write([]byte(host))
	key := mac.Sum(nil)
	subtle.ConstantTimeCopy(1, siteId[:32], key)
	return
}