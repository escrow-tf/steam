package mobileconf

import "context"

type Api interface {
	SendMobileConfRequest(ctx context.Context, request Request, response any) error
	GetList(ctx context.Context) (GetListResponse, error)
	GetDetailsPage(ctx context.Context, id string) (DetailsPageResponse, error)
	Accept(ctx context.Context, id, nonce string) (AcceptResponse, error)
	Decline(ctx context.Context, id, nonce string) (DeclineResponse, error)
}
