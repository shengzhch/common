package router

import (
	"context"
	"net/http"
	"net/url"
	"io"
	"mime/multipart"
	"github.com/gorilla/mux"
	kithttp "github.com/go-kit/kit/transport/http"
	"reflect"
	"github.com/gin-gonic/gin/json"
)

type Request struct {
	Vars          map[string]string
	Header        http.Header
	Body          interface{}
	RawBody       io.ReadCloser
	Form          url.Values
	PostForm      url.Values
	MultipartForm *multipart.Form
}

func DecodeRequestFunc_IntoJson(expect interface{}) kithttp.DecodeRequestFunc {
	return func(ctx context.Context, r *http.Request) (re interface{}, err error) {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
			r.ParseForm()
			r.ParseMultipartForm(10e6)
			body := makeAexpect(expect, r.Body, "json")
			return &Request{
				Vars:          mux.Vars(r),
				Header:        r.Header,
				Body:          body,
				RawBody:       r.Body,
				Form:          r.Form,
				PostForm:      r.PostForm,
				MultipartForm: r.MultipartForm,
			}, nil
		}
	}
}

func makeAexpect(in interface{}, body io.ReadCloser, med string) interface{} {
	if in == nil {
		return nil
	}
	switch med {
	case "json":
		typ := reflect.TypeOf(in)
		out := reflect.New(typ).Interface()
		err := json.NewDecoder(body).Decode(out)
		if err != nil {
			return nil
		}
		return out
	default:
		return nil

	}
}
