package tradeoffer

import "errors"

var (
	InvalidStateError = errors.New("this trade offer is in an invalid state, and cannot be acted upon; usually you'll need to send a new trade offer")
	AccessDeniedError = errors.New(`You can't send or accept this trade offer because either you can't trade with the other user, or one of the parties in this trade can't send or receive one of the items in the trade. Possible causes:

    You aren't friends with the other user and you didn't provide a trade token
    The provided trade token was wrong
    You are trying to send or receive an item for a game in which you or the other user can't trade (e.g. due to a VAC ban)
    You are trying to send an item and the other user's inventory is full for that game`)
	TimeoutError                    = errors.New("the Steam Community web server did not receive a timely reply from the trade offers server while sending/accepting this trade offer. It is possible (and not unlikely) that the operation actually succeeded")
	ServiceUnavailableError         = errors.New("the trade offers service is currently unavailable")
	TooManyTradeOffersError         = errors.New("you are exceeding your limit of 5 active offers per partner, or 30 active offers total")
	ItemsDontExistError             = errors.New("one or more of the items in this trade offer does not exist in the inventory from which it was requested")
	ChangedPersonaNameRecentlyError = errors.New("ou cannot send this trade offer because you have recently changed your persona name")
)
