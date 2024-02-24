package main

import (
	"encoding/base64"
	"encoding/json"
	"time"
)

type SyncToken struct {
	Timestamp         time.Time
	DeclarationsToken string
}

type DeclarativeManagementData struct {
	SyncTokens []SyncToken
}

func (d DeclarativeManagementData) Encode64() (string, error) {
	rawJSON, err := json.Marshal(d)
	if err != nil {
		return "", err
	}

	return base64.StdEncoding.EncodeToString(rawJSON), nil
}

type Enrollment struct {
	UDID             string
	EnrollmentUserID string
	UserID           string `plist:",omitempty"`
	UserShortName    string `plist:",omitempty"`
	UserLongName     string `plist:",omitempty"`
	EnrollmentID     string `plist:",omitempty"`
	Topic            string `plist:",omitempty"`
}

type Device struct {
	UDID string
	// The device's build version.
	BuildVersion string
	// The device's name
	Name string
	// The device's IMEI (International Mobile Station Equipment Identity).
	IMEI string
	// The device's MEID (Mobile Equipment Identifier).
	MEID string
	// The device's model.
	Model string
	// The device's model name.
	ModelName string
	// The device's OS version.
	OSVersion string
	// The device's product name (e.g., iPhone3,1).
	ProductName string
	// The device's serial number.
	SerialNumber string
	// The magic string that has to be included in the push
	// notification message.
	PushMagic string
	// The Push token for the device.
	Token []byte
	// The data that can be used to unlock the device. If
	// provided, the server should remember this data and send it when
	// trying to Clear the Passcode.
	UnlockToken    []byte
	BootstrapToken []byte `plist:",omitempty"`
}

type Authenticate struct {
	// The device's build version.
	BuildVersion string `plist:",omitempty"`
	// The device's name. (Required)
	DeviceName string
	// The device's IMEI (International Mobile Station Equipment Identity).
	IMEI string `plist:",omitempty"`
	// The device's MEID (Mobile Equipment Identifier).
	MEID string `plist:",omitempty"`
	// The device's model. (Required)
	Model string
	// The device's model name. (Required)
	ModelName string
	// The device's OS version.
	OSVersion string `plist:",omitempty"`
	// The device's product name (e.g., iPhone3,1).
	ProductName string `plist:",omitempty"`
	// The device's serial number.
	SerialNumber string `plist:",omitempty"`
}

type TokenUpdate struct {
	// If true from the device channel, the device
	// is awaiting a Release Device from Await Configuration MDM command
	// before proceeding through Setup Assistant.
	AwaitingConfiguration bool `plist:",omitempty"`
	// If true, the device is not on console.
	NotOnConsole bool
	// The magic string that has to be included in the push
	// notification message.
	PushMagic string
	// The Push token for the device.
	Token []byte
	// The data that can be used to unlock the device. If
	// provided, the server should remember this data and send it when
	// trying to Clear the Passcode.
	UnlockToken []byte `plist:",omitempty"`
}

type SetBootstrapToken struct {
	// If true, the device is awaiting a
	// DeviceConfigured MDM command before proceeding through Setup
	// Assistant. Default: false.
	AwaitingConfiguration bool `plist:",omitempty"`
	// The device's bootstrap token data. If this field is
	// missing or zero length, the bootstrap token should be removed for
	// this device.
	BootstrapToken []byte `plist:",omitempty"`
}

type DeclarativeManagement struct {
	// A Base64-encoded JSON object using the SynchronizationTokens schema.
	Data []byte `plist:",omitempty"`
	// The type of operation the declaration is requesting. Must
	// be one of 'tokens', 'declaration-items', 'status', or
	// 'declaration/…/…'.
	Endpoint string
}

type MDMRequest struct {
	Enrollment
	MessageType string `plist:"MessageType"`
	Authenticate
	TokenUpdate
	SetBootstrapToken
	DeclarativeManagement
	CommandResult
}

// ErrorChain represents errors that occured on the client executing an MDM command.
type ErrorChain struct {
	ErrorCode            int
	ErrorDomain          string
	LocalizedDescription string
	USEnglishDescription string
}

// CommandResults represents a 'command and report results' request.
// See https://developer.apple.com/documentation/devicemanagement/implementing_device_management/sending_mdm_commands_to_a_device
type CommandResult struct {
	CommandUUID string
	Status      string
	ErrorChain  []ErrorChain
	RequestType string
}
