package logger

import (
	"crypto/hmac"
	"crypto/md5"
	"encoding/hex"
	"os"
	"strings"
)

type MaskingOptionDto struct {
	MaskingField string
	MaskingType  MaskingType
	IsArray      bool
}

// MaskingType represents different types of data masking.
type MaskingType int

const (
	Msisdn      MaskingType = iota // Mobile phone number
	Fbb                            // Fixed broadband number
	CreditCard                     // Credit card number
	IDCard                         // ID card number
	BankAccount                    // Bank account number
	Firstname                      // First name
	Lastname                       // Last name
	Email                          // Email address
	Full                           // Full string masking
	Hashing                        // HMAC/Hashing
)

var hmacKey = os.Getenv("HMAC_KEY_ENV")

type MaskingServiceInterface interface {
	Masking(value string, t MaskingType) string
}

type MaskingService struct {
	maskingDisplayCharacter string
}

func NewMaskingService() MaskingServiceInterface {
	return &MaskingService{maskingDisplayCharacter: "X"}
}

func (m *MaskingService) censorEmail(email string) string {
	if len(email) < 3 {
		return email
	}
	firstThree := email[:3]
	masked := []rune(email)
	for i := 3; i < len(masked); i++ {
		if (masked[i] >= 'a' && masked[i] <= 'z') || (masked[i] >= 'A' && masked[i] <= 'Z') || (masked[i] >= '0' && masked[i] <= '9') {
			masked[i] = rune(m.maskingDisplayCharacter[0])
		}
	}
	return firstThree + string(masked[3:])
}

func (m *MaskingService) censorBankAccountNumber(accountNumber string) string {
	length := len(accountNumber)
	if length <= 7 {
		return accountNumber
	}
	firstFour := accountNumber[:4]
	lastThree := accountNumber[length-3:]
	middle := strings.Repeat(m.maskingDisplayCharacter, length-7)
	return firstFour + middle + lastThree
}

func (m *MaskingService) censorCreditCardId(creditCardId string) string {
	length := len(creditCardId)
	if length < 11 {
		return creditCardId
	}
	firstSix := creditCardId[:6]
	lastFour := creditCardId[length-4:]
	middle := strings.Repeat(m.maskingDisplayCharacter, length-10)
	return firstSix + middle + lastFour
}

func (m *MaskingService) censorPhoneNumber(phoneNumber string) string {
	length := len(phoneNumber)
	if length < 7 {
		return phoneNumber
	}
	firstThree := phoneNumber[:3]
	lastFour := phoneNumber[length-4:]
	middle := strings.Repeat(m.maskingDisplayCharacter, length-7)
	return firstThree + middle + lastFour
}

func (m *MaskingService) censorFbbNumber(fiberNumber string) string {
	length := len(fiberNumber)
	if length < 6 {
		return fiberNumber
	}
	firstTwo := fiberNumber[:2]
	lastFour := fiberNumber[length-4:]
	middle := strings.Repeat(m.maskingDisplayCharacter, length-6)
	return firstTwo + middle + lastFour
}

func (m *MaskingService) censorIDCard(idCard string) string {
	length := len(idCard)
	if length < 4 {
		return idCard
	}
	lastFour := idCard[length-4:]
	masked := strings.Repeat(m.maskingDisplayCharacter, length-4)
	return masked + lastFour
}

func (m *MaskingService) censorExcludeFirst3(value string) string {
	length := len(value)
	if length < 3 {
		return value
	}
	firstThree := value[:3]
	masked := strings.Repeat(m.maskingDisplayCharacter, length-3)
	return firstThree + masked
}

func (m *MaskingService) censorFull(data string) string {
	return strings.Repeat(m.maskingDisplayCharacter, len(data))
}

func (m *MaskingService) hmacValue(value string) string {
	defer func() {
		if r := recover(); r != nil {
			// handle panic if any
		}
	}()
	h := hmac.New(md5.New, []byte(hmacKey))
	h.Write([]byte(value))
	return hex.EncodeToString(h.Sum(nil))
}

func (m *MaskingService) Masking(value string, t MaskingType) string {
	switch t {
	case Msisdn:
		return m.censorPhoneNumber(value)
	case Fbb:
		return m.censorFbbNumber(value)
	case CreditCard:
		return m.censorCreditCardId(value)
	case BankAccount:
		return m.censorBankAccountNumber(value)
	case Email:
		return m.censorEmail(value)
	case IDCard:
		return m.censorIDCard(value)
	case Full:
		return m.censorFull(value)
	case Firstname, Lastname:
		return m.censorExcludeFirst3(value)
	case Hashing:
		return m.hmacValue(value)
	default:
		return value
	}
}
