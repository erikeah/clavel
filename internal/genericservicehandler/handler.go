package genericservicehandler

import (
	"context"

	"connectrpc.com/connect"
)

type ServiceCreate[Request, Response any] interface {
	Create(context.Context, *Request) (*Response, error)
}
type ServiceDelete[Request, Response any] interface {
	Delete(context.Context, *Request) (*Response, error)
}
type ServiceList[Request, Response any] interface {
	List(context.Context, *Request) (*Response, error)
}
type ServiceShow[Request, Response any] interface {
	Show(context.Context, *Request) (*Response, error)
}

type Service[
	CreateRequest, CreateResponse,
	DeleteRequest, DeleteResponse,
	ListRequest, ListResponse,
	ShowRequest, ShowResponse any,
] interface {
	ServiceCreate[CreateRequest, CreateResponse]
	ServiceList[ListRequest, ListResponse]
	ServiceDelete[DeleteRequest, DeleteResponse]
	ServiceShow[ShowRequest, ShowResponse]
}

type genericServiceHandler[
	CreateRequest, CreateResponse,
	DeleteRequest, DeleteResponse,
	ListRequest, ListResponse,
	ShowRequest, ShowResponse any,
] struct {
	service Service[
		CreateRequest, CreateResponse,
		DeleteRequest, DeleteResponse,
		ListRequest, ListResponse,
		ShowRequest, ShowResponse,
	]
}

func (handler *genericServiceHandler[
	CreateRequest, CreateResponse,
	DeleteRequest, DeleteResponse,
	ListRequest, ListResponse,
	ShowRequest, ShowResponse,
]) Create(
	ctx context.Context,
	request *connect.Request[CreateRequest],
) (*connect.Response[CreateResponse], error) {
	res, err := handler.service.Create(ctx, request.Msg)
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(res), nil
}

func (handler *genericServiceHandler[
	CreateRequest, CreateResponse,
	DeleteRequest, DeleteResponse,
	ListRequest, ListResponse,
	ShowRequest, ShowResponse,
]) Delete(
	ctx context.Context,
	request *connect.Request[DeleteRequest],
) (*connect.Response[DeleteResponse], error) {
	res, err := handler.service.Delete(ctx, request.Msg)
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(res), nil
}

func (handler *genericServiceHandler[
	CreateRequest, CreateResponse,
	DeleteRequest, DeleteResponse,
	ListRequest, ListResponse,
	ShowRequest, ShowResponse,
]) List(
	ctx context.Context,
	request *connect.Request[ListRequest],
) (*connect.Response[ListResponse], error) {
	res, err := handler.service.List(ctx, request.Msg)
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(res), nil
}

func (handler *genericServiceHandler[
	CreateRequest, CreateResponse,
	DeleteRequest, DeleteResponse,
	ListRequest, ListResponse,
	ShowRequest, ShowResponse,
]) Show(
	ctx context.Context,
	request *connect.Request[ShowRequest],
) (*connect.Response[ShowResponse], error) {
	res, err := handler.service.Show(ctx, request.Msg)
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(res), nil
}

func NewServiceHandler[
	CreateRequest, CreateResponse,
	DeleteRequest, DeleteResponse,
	ListRequest, ListResponse,
	ShowRequest, ShowResponse any,
](
	service Service[
		CreateRequest, CreateResponse,
		DeleteRequest, DeleteResponse,
		ListRequest, ListResponse,
		ShowRequest, ShowResponse,
	],
) *genericServiceHandler[
	CreateRequest, CreateResponse,
	DeleteRequest, DeleteResponse,
	ListRequest, ListResponse,
	ShowRequest, ShowResponse,
] {
	return &genericServiceHandler[
		CreateRequest, CreateResponse,
		DeleteRequest, DeleteResponse,
		ListRequest, ListResponse,
		ShowRequest, ShowResponse,
	]{
		service: service,
	}
}
