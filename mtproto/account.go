package mtproto

import (
	"context"
	"errors"
	"fmt"
	"strings"
)

const (
	constructorAuthSendCode                = 0xa677244f
	constructorAuthSignIn                  = 0x8d52a951
	constructorCodeSettings                = 0xad253d78
	constructorAuthSentCode                = 0x5e002502
	constructorAuthSentCodeSuccess         = 0x2390fe44
	constructorAuthSentCodePaymentRequired = 0xd7a2fcf9

	constructorSentCodeTypeApp              = 0x3dbb5986
	constructorSentCodeTypeSMS              = 0xc000bba2
	constructorSentCodeTypeCall             = 0x5353e5a7
	constructorSentCodeTypeFlashCall        = 0xab03c6d9
	constructorSentCodeTypeMissedCall       = 0x82006484
	constructorSentCodeTypeEmailCode        = 0xf450f59b
	constructorSentCodeTypeSetUpEmailReq    = 0xa5491dea
	constructorSentCodeTypeFragmentSMS      = 0xd9565c39
	constructorSentCodeTypeFirebaseSMS      = 0x009fd736
	constructorSentCodeTypeSMSWord          = 0xa416ac81
	constructorSentCodeTypeSMSPhrase        = 0xb37794af
)

// SentCodeResult is the parsed auth.SentCode response returned by AuthSendCode.
type SentCodeResult struct {
	PhoneNumber         string
	PhoneCodeHash       string
	Constructor         uint32
	CodeTypeConstructor uint32
	Timeout             int32
	Body                []byte
	Message             string
}

func (r SentCodeResult) ConstructorHex() string { return fmt.Sprintf("0x%08x", r.Constructor) }
func (r SentCodeResult) CodeTypeHex() string    { return fmt.Sprintf("0x%08x", r.CodeTypeConstructor) }

// AuthSendCode sends a login code to a user account phone number using pure encrypted MTProto.
// The returned PhoneCodeHash must be passed to AuthSignIn with the received login code.
func (c *EncryptedClient) AuthSendCode(ctx context.Context, apiID int, apiHash, phoneNumber string) (*SentCodeResult, error) {
	if apiID <= 0 {
		return nil, errors.New("api id is required")
	}
	if strings.TrimSpace(apiHash) == "" {
		return nil, errors.New("api hash is required")
	}
	if strings.TrimSpace(phoneNumber) == "" {
		return nil, errors.New("phone number is required")
	}
	result, err := c.Invoke(ctx, makeAuthSendCodeQuery(apiID, apiHash, phoneNumber))
	if err != nil {
		return nil, err
	}
	sent, err := parseSentCodeResult(phoneNumber, result.Body)
	if err != nil {
		return nil, err
	}
	sent.Message = "auth.sendCode succeeded over encrypted MTProto"
	return sent, nil
}

// AuthSignIn signs in a user account using the login code and phone_code_hash returned by AuthSendCode.
func (c *EncryptedClient) AuthSignIn(ctx context.Context, apiID int, phoneNumber, phoneCodeHash, phoneCode string) (*InvokeResult, error) {
	if apiID <= 0 {
		return nil, errors.New("api id is required")
	}
	if strings.TrimSpace(phoneNumber) == "" {
		return nil, errors.New("phone number is required")
	}
	if strings.TrimSpace(phoneCodeHash) == "" {
		return nil, errors.New("phone_code_hash is required")
	}
	if strings.TrimSpace(phoneCode) == "" {
		return nil, errors.New("phone code is required")
	}
	result, err := c.Invoke(ctx, makeAuthSignInQuery(apiID, phoneNumber, phoneCodeHash, phoneCode))
	if err != nil {
		return nil, err
	}
	result.Message = "auth.signIn succeeded over encrypted MTProto"
	return result, nil
}

// AuthSendCodeDefault creates an encrypted session, sends auth.sendCode, and follows PHONE_MIGRATE_X automatically.
// The returned client must be reused for AuthSignIn and closed by the caller.
func AuthSendCodeDefault(ctx context.Context, apiID int, apiHash, phoneNumber string) (*EncryptedClient, *SentCodeResult, error) {
	client, err := DialEncryptedDefault(ctx)
	if err != nil {
		return nil, nil, err
	}
	result, err := client.AuthSendCode(ctx, apiID, apiHash, phoneNumber)
	if err == nil {
		return client, result, nil
	}

	migrateTo, ok := MigrationDCID(err)
	if !ok {
		_ = client.Close()
		return nil, nil, err
	}
	_ = client.Close()

	migratedClient, migratedResult, migratedErr := authSendCodeOnDC(ctx, migrateTo, apiID, apiHash, phoneNumber)
	if migratedErr != nil {
		return nil, nil, migratedErr
	}
	migratedResult.Message = fmt.Sprintf("auth.sendCode succeeded after migrating to DC %d", migrateTo)
	return migratedClient, migratedResult, nil
}

func authSendCodeOnDC(ctx context.Context, dcID int, apiID int, apiHash, phoneNumber string) (*EncryptedClient, *SentCodeResult, error) {
	options := dcOptionsByID(dcID)
	if len(options) == 0 {
		return nil, nil, fmt.Errorf("mtproto: no default address configured for migrated DC %d", dcID)
	}
	var last error
	for _, dc := range options {
		client, err := DialEncrypted(ctx, dc)
		if err != nil {
			last = err
			continue
		}
		result, err := client.AuthSendCode(ctx, apiID, apiHash, phoneNumber)
		if err == nil {
			return client, result, nil
		}
		_ = client.Close()
		last = err
		if nextDC, ok := MigrationDCID(err); ok && nextDC != dcID {
			return authSendCodeOnDC(ctx, nextDC, apiID, apiHash, phoneNumber)
		}
	}
	if last == nil {
		last = fmt.Errorf("mtproto: failed to connect to migrated DC %d", dcID)
	}
	return nil, nil, last
}

func makeAuthSendCodeQuery(apiID int, apiHash, phoneNumber string) []byte {
	var inner tlBuffer
	inner.putInt(constructorAuthSendCode)
	inner.putString(phoneNumber)
	inner.putInt(uint32(int32(apiID)))
	inner.putString(apiHash)
	inner.putInt(constructorCodeSettings)
	inner.putInt(0) // flags: minimal codeSettings, no optional fields.
	return wrapWithLayerAndInitConnection(apiID, appVersionV8, inner.bytes())
}

func makeAuthSignInQuery(apiID int, phoneNumber, phoneCodeHash, phoneCode string) []byte {
	var inner tlBuffer
	inner.putInt(constructorAuthSignIn)
	inner.putInt(1) // flags.0 -> phone_code is present.
	inner.putString(phoneNumber)
	inner.putString(phoneCodeHash)
	inner.putString(phoneCode)
	return wrapWithLayerAndInitConnection(apiID, appVersionV8, inner.bytes())
}

func parseSentCodeResult(phoneNumber string, body []byte) (*SentCodeResult, error) {
	r := newTLReader(body)
	constructor, err := r.int()
	if err != nil {
		return nil, err
	}
	result := &SentCodeResult{PhoneNumber: phoneNumber, Constructor: constructor, Body: append([]byte(nil), body...)}
	switch constructor {
	case constructorAuthSentCode:
		flags, err := r.int()
		if err != nil {
			return nil, err
		}
		codeType, err := r.int()
		if err != nil {
			return nil, err
		}
		result.CodeTypeConstructor = codeType
		if err := skipSentCodeType(codeType, r); err != nil {
			return nil, err
		}
		hash, err := r.string()
		if err != nil {
			return nil, err
		}
		result.PhoneCodeHash = hash
		if flags&(1<<1) != 0 {
			if _, err := r.int(); err != nil { // auth.CodeType constructor; all current constructors have no fields.
				return nil, err
			}
		}
		if flags&(1<<2) != 0 {
			timeout, err := r.int()
			if err != nil {
				return nil, err
			}
			result.Timeout = int32(timeout)
		}
		return result, nil
	case constructorAuthSentCodeSuccess:
		result.Message = "auth.sentCodeSuccess: already authorized by a future auth token"
		return result, nil
	case constructorAuthSentCodePaymentRequired:
		if _, err := r.string(); err != nil { // store_product
			return nil, err
		}
		hash, err := r.string()
		if err != nil {
			return nil, err
		}
		result.PhoneCodeHash = hash
		return result, nil
	default:
		return nil, fmt.Errorf("mtproto: expected auth.SentCode, got 0x%08x", constructor)
	}
}

func skipSentCodeType(constructor uint32, r *tlReader) error {
	switch constructor {
	case constructorSentCodeTypeApp, constructorSentCodeTypeSMS, constructorSentCodeTypeCall:
		_, err := r.int()
		return err
	case constructorSentCodeTypeFlashCall:
		_, err := r.string()
		return err
	case constructorSentCodeTypeMissedCall:
		if _, err := r.string(); err != nil {
			return err
		}
		_, err := r.int()
		return err
	case constructorSentCodeTypeEmailCode:
		flags, err := r.int()
		if err != nil {
			return err
		}
		if _, err := r.string(); err != nil { // email_pattern
			return err
		}
		if _, err := r.int(); err != nil { // length
			return err
		}
		if flags&(1<<3) != 0 {
			if _, err := r.int(); err != nil {
				return err
			}
		}
		if flags&(1<<4) != 0 {
			if _, err := r.int(); err != nil {
				return err
			}
		}
		return nil
	case constructorSentCodeTypeSetUpEmailReq:
		_, err := r.int() // flags
		return err
	case constructorSentCodeTypeFragmentSMS:
		if _, err := r.string(); err != nil { // url
			return err
		}
		_, err := r.int()
		return err
	case constructorSentCodeTypeFirebaseSMS:
		flags, err := r.int()
		if err != nil {
			return err
		}
		if flags&(1<<0) != 0 {
			if _, err := r.bytes(); err != nil { // nonce
				return err
			}
		}
		if flags&(1<<2) != 0 {
			if _, err := r.long(); err != nil { // play_integrity_project_id
				return err
			}
			if _, err := r.bytes(); err != nil { // play_integrity_nonce
				return err
			}
		}
		if flags&(1<<1) != 0 {
			if _, err := r.string(); err != nil { // receipt
				return err
			}
			if _, err := r.int(); err != nil { // push_timeout
				return err
			}
		}
		_, err = r.int() // length
		return err
	case constructorSentCodeTypeSMSWord, constructorSentCodeTypeSMSPhrase:
		flags, err := r.int()
		if err != nil {
			return err
		}
		if flags&(1<<0) != 0 {
			_, err = r.string()
			return err
		}
		return nil
	default:
		return fmt.Errorf("mtproto: unsupported sent code type 0x%08x", constructor)
	}
}
