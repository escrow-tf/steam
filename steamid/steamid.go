package steamid

import (
	"errors"
	"fmt"
	"strconv"
)

type Universe int
type Type int
type Instance int

const (
	UniverseInvalid Universe = iota
	UniversePublic
	UniverseBeta
	UniverseInternal
	UniverseDev
)

const (
	TypeInvalid Type = iota
	TypeIndividual
	TypeMultiseat
	TypeGameServer
	TypeAnonGameServer
	TypePending
	TypeContentServer
	TypeClan
	TypeChat
	TypeP2pSuperSeeder
	TypeAnonUser
)

const (
	InstanceAll Instance = iota
	InstanceDesktop
	InstanceConsole
	InstanceWeb
)

const (
	AccountIDMask       int64 = 0xFFFFFFFF
	AccountInstanceMask int64 = 0x000FFFFF
	AccountTypeMask     int64 = 0xF
)

var (
	ErrorEmpty = errors.New("can't parse empty string as SteamID64")
)

type SteamID struct {
	original  string
	universe  Universe
	idType    Type
	instance  Instance
	accountID int32
}

func ParseSteamID64(s string) (SteamID, error) {
	steamID := SteamID{
		original:  s,
		universe:  UniverseInvalid,
		idType:    TypeInvalid,
		instance:  InstanceAll,
		accountID: 0,
	}

	if s == "" {
		return steamID, ErrorEmpty
	}

	parsedID, err := strconv.ParseInt(s, 10, 64)
	if err != nil {
		return steamID, fmt.Errorf("can't parse steamID into int64: %w", err)
	}

	steamID.accountID = int32(parsedID & AccountIDMask)
	steamID.instance = Instance((parsedID >> 32) & AccountInstanceMask)
	steamID.idType = Type((parsedID >> 52) & AccountTypeMask)
	steamID.universe = Universe(parsedID >> 56)

	return steamID, nil
}

func (id SteamID) String() string {
	return id.original
}

func (id SteamID) IsValid() bool {
	switch {
	case id.idType <= TypeInvalid || id.idType > TypeAnonUser:
		return false
	case id.universe <= UniverseInvalid || id.universe > UniverseDev:
		return false
	case id.idType == TypeIndividual && (id.accountID == 0 || id.instance > InstanceWeb):
		return false
	case id.idType == TypeClan && (id.accountID == 0 || id.instance != InstanceAll):
		return false
	case id.idType == TypeGameServer && id.accountID == 0:
		return false
	}

	return true
}

func (id SteamID) IsValidIndividual() bool {
	return id.universe == UniversePublic &&
		id.idType == TypeIndividual &&
		id.instance == InstanceDesktop &&
		id.accountID != 0
}

func (id SteamID) AccountId() int {
	return int(id.accountID)
}
