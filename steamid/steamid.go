package steamid

import (
	"errors"
	"strconv"

	"github.com/rotisserie/eris"
)

type Universe uint
type Type uint
type Instance uint

//goland:noinspection GoUnusedConst
const (
	UniverseInvalid Universe = iota
	UniversePublic
	UniverseBeta
	UniverseInternal
	UniverseDev
)

//goland:noinspection GoUnusedConst
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

//goland:noinspection GoUnusedConst
const (
	InstanceAll Instance = iota
	InstanceDesktop
	InstanceConsole
	InstanceWeb
)

//goland:noinspection GoUnusedConst
const (
	AccountIDMask       uint64 = 0xFFFFFFFF
	AccountInstanceMask uint64 = 0x000FFFFF
	AccountTypeMask     uint64 = 0xF
)

var (
	ErrorEmpty = errors.New("can't parse empty string as SteamID64")
)

type SteamID struct {
	original  string
	universe  Universe
	idType    Type
	instance  Instance
	accountID uint32
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

	parsedID, err := strconv.ParseUint(s, 10, 64)
	if err != nil {
		return steamID, eris.Wrapf(err, "can't parse steamID into int64")
	}

	steamID.accountID = uint32(parsedID & AccountIDMask)
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

func (id SteamID) AccountId() uint32 {
	return id.accountID
}
