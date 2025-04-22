package steamlang

import (
	"errors"
	"fmt"
	"net/http"
	"strconv"
)

type EResult int

const (
	InvalidResult                                 EResult = 0
	OKResult                                      EResult = 1
	FailResult                                    EResult = 2
	NoConnectionResult                            EResult = 3
	InvalidPasswordResult                         EResult = 5
	LoggedInElsewhereResult                       EResult = 6
	InvalidProtocolVerResult                      EResult = 7
	InvalidParamResult                            EResult = 8
	FileNotFoundResult                            EResult = 9
	BusyResult                                    EResult = 10
	InvalidStateResult                            EResult = 11
	InvalidNameResult                             EResult = 12
	InvalidEmailResult                            EResult = 13
	DuplicateNameResult                           EResult = 14
	AccessDeniedResult                            EResult = 15
	TimeoutResult                                 EResult = 16
	BannedResult                                  EResult = 17
	AccountNotFoundResult                         EResult = 18
	InvalidSteamIDResult                          EResult = 19
	ServiceUnavailableResult                      EResult = 20
	NotLoggedOnResult                             EResult = 21
	PendingResult                                 EResult = 22
	EncryptionFailureResult                       EResult = 23
	InsufficientPrivilegeResult                   EResult = 24
	LimitExceededResult                           EResult = 25
	RevokedResult                                 EResult = 26
	ExpiredResult                                 EResult = 27
	AlreadyRedeemedResult                         EResult = 28
	DuplicateRequestResult                        EResult = 29
	AlreadyOwnedResult                            EResult = 30
	IPNotFoundResult                              EResult = 31
	PersistFailedResult                           EResult = 32
	LockingFailedResult                           EResult = 33
	LogonSessionReplacedResult                    EResult = 34
	ConnectFailedResult                           EResult = 35
	HandshakeFailedResult                         EResult = 36
	IOFailureResult                               EResult = 37
	RemoteDisconnectResult                        EResult = 38
	ShoppingCartNotFoundResult                    EResult = 39
	BlockedResult                                 EResult = 40
	IgnoredResult                                 EResult = 41
	NoMatchResult                                 EResult = 42
	AccountDisabledResult                         EResult = 43
	ServiceReadOnlyResult                         EResult = 44
	AccountNotFeaturedResult                      EResult = 45
	AdministratorOKResult                         EResult = 46
	ContentVersionResult                          EResult = 47
	TryAnotherCMResult                            EResult = 48
	PasswordRequiredToKickSessionResult           EResult = 49
	AlreadyLoggedInElsewhereResult                EResult = 50
	SuspendedResult                               EResult = 51
	CancelledResult                               EResult = 52
	DataCorruptionResult                          EResult = 53
	DiskFullResult                                EResult = 54
	RemoteCallFailedResult                        EResult = 55
	PasswordNotSetOrUnsetResult                   EResult = 56
	ExternalAccountUnlinkedResult                 EResult = 57
	PSNTicketInvalidResult                        EResult = 58
	ExternalAccountAlreadyLinkedResult            EResult = 59
	RemoteFileConflictResult                      EResult = 60
	IllegalPasswordResult                         EResult = 61
	SameAsPreviousValueResult                     EResult = 62
	AccountLogonDeniedResult                      EResult = 63
	CannotUseOldPasswordResult                    EResult = 64
	InvalidLoginAuthCodeResult                    EResult = 65
	AccountLogonDeniedNoMailSentResult            EResult = 66
	HardwareNotCapableOfIPTResult                 EResult = 67
	IPTInitErrorResult                            EResult = 68
	ParentalControlRestrictedResult               EResult = 69
	FacebookQueryErrorResult                      EResult = 70
	ExpiredLoginAuthCodeResult                    EResult = 71
	IPLoginRestrictionFailedResult                EResult = 72
	AccountLockedResult                           EResult = 73
	AccountLogonDeniedVerifiedEmailRequiredResult EResult = 74
	NoMatchingURLResult                           EResult = 75
	BadResponseResult                             EResult = 76
	RequirePasswordReEntryResult                  EResult = 77
	ValueOutOfRangeResult                         EResult = 78
	UnexpectedErrorResult                         EResult = 79
	DisabledResult                                EResult = 80
	InvalidCEGSubmissionResult                    EResult = 81
	RestrictedDeviceResult                        EResult = 82
	RegionLockedResult                            EResult = 83
	RateLimitExceededResult                       EResult = 84
	AccountLoginDeniedNeedTwoFactorResult         EResult = 85
	ItemOrEntryHasBeenDeletedResult               EResult = 86
	AccountLoginDeniedThrottleResult              EResult = 87
	TwoFactorCodeMismatchResult                   EResult = 88
	TwoFactorActivationCodeMismatchResult         EResult = 89
	AccountAssociatedToMultipleAccountsResult     EResult = 90
	NotModifiedResult                             EResult = 91
	NoMobileDeviceAvailableResult                 EResult = 92
	TimeNotSyncedResult                           EResult = 93
	SMSCodeFailedResult                           EResult = 94
	AccountLimitExceededResult                    EResult = 95
	AccountActivityLimitExceededResult            EResult = 96
	PhoneActivityLimitExceededResult              EResult = 97
	RefundToWalletResult                          EResult = 98
	EmailSendFailureResult                        EResult = 99
	NotSettledResult                              EResult = 100
	NeedCaptchaResult                             EResult = 101
	GSLTDeniedResult                              EResult = 102
	GSOwnerDeniedResult                           EResult = 103
	InvalidItemTypeResult                         EResult = 104
	IPBannedResult                                EResult = 105
	GSLTExpiredResult                             EResult = 106
	InsufficientFundsResult                       EResult = 107
	TooManyPendingResult                          EResult = 108
	NoSiteLicensesFoundResult                     EResult = 109
	WGNetworkSendExceededResult                   EResult = 110
	AccountNotFriendsResult                       EResult = 111
	LimitedUserAccountResult                      EResult = 112
	CantRemoveItemResult                          EResult = 113
	AccountDeletedResult                          EResult = 114
	ExistingUserCancelledLicenseResult            EResult = 115
	DeniedDueToCommunityCooldownResult            EResult = 116
	NoLauncherSpecifiedResult                     EResult = 117
	MustAgreeToSSAResult                          EResult = 118
	ClientNoLongerSupportedResult                 EResult = 119
	SteamRealmMismatchResult                      EResult = 120
	InvalidSignatureResult                        EResult = 121
	ParseFailureResult                            EResult = 122
	NoVerifiedPhoneResult                         EResult = 123
	InsufficientBatteryResult                     EResult = 124
	ChargerRequiredResult                         EResult = 125
	CachedCredentialInvalidResult                 EResult = 126
	PhoneNumberIsVOIPResult                       EResult = 127
)

func EnsureSuccessResponse(response *http.Response) error {
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return fmt.Errorf("BeginAuthSessionViaCredentials request failed with status %v", response.StatusCode)
	}

	return nil
}

func EnsureEResultResponse(httpResponse *http.Response) error {
	eResult := InvalidResult
	eResults, hasEResult := httpResponse.Header["X-Eresult"]
	if !hasEResult {
		return nil
	}

	for _, result := range eResults {
		if parsedResult, parseErr := strconv.ParseInt(result, 10, 64); parseErr == nil {
			eResult = EResult(parsedResult)
			break
		}
	}

	if eResult != OKResult {
		if errorMessageHeaders, ok := httpResponse.Header["X-Error_message"]; ok {
			errorMessages := make([]error, len(errorMessageHeaders))
			for i, header := range errorMessageHeaders {
				errorMessages[i] = errors.New(header)
			}

			return fmt.Errorf("steam responded with non-OK Result: %v, %v", eResult, errors.Join(errorMessages...))
		}

		return fmt.Errorf("steam responded with non-OK Result: %v", eResult)
	}

	return nil
}
